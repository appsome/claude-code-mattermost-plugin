package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// CLIProcess represents a running Claude Code CLI process
type CLIProcess struct {
	SessionID   string
	Cmd         *exec.Cmd
	Stdin       io.WriteCloser
	Stdout      io.ReadCloser
	Stderr      io.ReadCloser
	StartTime   time.Time
	ProjectPath string
	ChannelID   string
	UserID      string
	done        chan struct{}
	mu          sync.Mutex
}

// ProcessManager manages all CLI processes
type ProcessManager struct {
	processes sync.Map // map[sessionID]*CLIProcess
	plugin    *Plugin
}

// NewProcessManager creates a new ProcessManager
func NewProcessManager(plugin *Plugin) *ProcessManager {
	return &ProcessManager{
		plugin: plugin,
	}
}

// Spawn starts a new Claude Code CLI process for a session
func (pm *ProcessManager) Spawn(sessionID, projectPath, channelID, userID string) error {
	// Check if process already exists
	if pm.IsRunning(sessionID) {
		return fmt.Errorf("session %s already has a running process", sessionID)
	}

	// Get the CLI path from configuration
	config := pm.plugin.getConfiguration()
	cliPath := config.ClaudeCodePath
	if cliPath == "" {
		cliPath = "claude" // Default to PATH lookup
	}

	// Verify the CLI exists
	if _, err := exec.LookPath(cliPath); err != nil {
		return fmt.Errorf("claude code CLI not found at %s: %w", cliPath, err)
	}

	// Create the command with project path
	cmd := exec.Command(cliPath, "--print", "--output-format", "stream-json")
	cmd.Dir = projectPath

	// Set environment variables for CLI
	cmd.Env = append(os.Environ(),
		"CLAUDE_CODE_ENTRYPOINT=mattermost-plugin",
	)

	// Create pipes for stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed to start CLI process: %w", err)
	}

	process := &CLIProcess{
		SessionID:   sessionID,
		Cmd:         cmd,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
		StartTime:   time.Now(),
		ProjectPath: projectPath,
		ChannelID:   channelID,
		UserID:      userID,
		done:        make(chan struct{}),
	}

	// Store the process
	pm.processes.Store(sessionID, process)

	// Start goroutines to handle output
	go pm.handleStdout(process)
	go pm.handleStderr(process)
	go pm.waitForExit(process)

	pm.plugin.API.LogInfo("Started CLI process",
		"sessionID", sessionID,
		"projectPath", projectPath,
		"pid", cmd.Process.Pid,
	)

	return nil
}

// handleStdout reads and processes stdout from the CLI
func (pm *ProcessManager) handleStdout(process *CLIProcess) {
	scanner := bufio.NewScanner(process.Stdout)
	// Increase buffer size for large JSON outputs
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		pm.plugin.outputHandler.HandleOutput(process.SessionID, process.ChannelID, line)
	}

	if err := scanner.Err(); err != nil {
		pm.plugin.API.LogError("Error reading stdout",
			"sessionID", process.SessionID,
			"error", err.Error(),
		)
	}
}

// handleStderr reads and processes stderr from the CLI
func (pm *ProcessManager) handleStderr(process *CLIProcess) {
	scanner := bufio.NewScanner(process.Stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		pm.plugin.outputHandler.HandleError(process.SessionID, process.ChannelID, line)
	}

	if err := scanner.Err(); err != nil {
		pm.plugin.API.LogError("Error reading stderr",
			"sessionID", process.SessionID,
			"error", err.Error(),
		)
	}
}

// waitForExit waits for the process to exit and handles cleanup
func (pm *ProcessManager) waitForExit(process *CLIProcess) {
	err := process.Cmd.Wait()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	close(process.done)
	pm.processes.Delete(process.SessionID)

	pm.plugin.outputHandler.HandleExit(process.SessionID, process.ChannelID, exitCode)

	pm.plugin.API.LogInfo("CLI process exited",
		"sessionID", process.SessionID,
		"exitCode", exitCode,
	)
}

// SendInput sends input to the CLI process stdin
func (pm *ProcessManager) SendInput(sessionID, input string) error {
	processInterface, ok := pm.processes.Load(sessionID)
	if !ok {
		return fmt.Errorf("no running process for session %s", sessionID)
	}

	process := processInterface.(*CLIProcess)
	process.mu.Lock()
	defer process.mu.Unlock()

	// Check if process is still running
	select {
	case <-process.done:
		return fmt.Errorf("process for session %s has exited", sessionID)
	default:
	}

	// Write input followed by newline
	_, err := fmt.Fprintln(process.Stdin, input)
	if err != nil {
		return fmt.Errorf("failed to send input to CLI: %w", err)
	}

	return nil
}

// SendInputJSON sends a JSON-encoded message to the CLI process stdin
func (pm *ProcessManager) SendInputJSON(sessionID string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal input to JSON: %w", err)
	}

	return pm.SendInput(sessionID, string(jsonBytes))
}

// Kill terminates a CLI process
func (pm *ProcessManager) Kill(sessionID string) error {
	processInterface, ok := pm.processes.Load(sessionID)
	if !ok {
		return nil // Process doesn't exist, nothing to kill
	}

	process := processInterface.(*CLIProcess)
	return pm.killProcess(process)
}

// killProcess performs the actual process termination
func (pm *ProcessManager) killProcess(process *CLIProcess) error {
	process.mu.Lock()
	defer process.mu.Unlock()

	// Check if already done
	select {
	case <-process.done:
		return nil
	default:
	}

	// Close stdin first to signal EOF
	process.Stdin.Close()

	// Try graceful termination with SIGTERM
	if err := process.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		pm.plugin.API.LogWarn("Failed to send SIGTERM",
			"sessionID", process.SessionID,
			"error", err.Error(),
		)
	}

	// Wait up to 5 seconds for graceful shutdown
	select {
	case <-process.done:
		return nil
	case <-time.After(5 * time.Second):
		// Force kill with SIGKILL
		pm.plugin.API.LogWarn("Process did not terminate gracefully, sending SIGKILL",
			"sessionID", process.SessionID,
		)
		if err := process.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	return nil
}

// KillAll terminates all running CLI processes
func (pm *ProcessManager) KillAll() {
	var wg sync.WaitGroup

	pm.processes.Range(func(key, value interface{}) bool {
		process := value.(*CLIProcess)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := pm.killProcess(process); err != nil {
				pm.plugin.API.LogError("Failed to kill process",
					"sessionID", process.SessionID,
					"error", err.Error(),
				)
			}
		}()
		return true
	})

	wg.Wait()
}

// IsRunning checks if a process is running for the given session
func (pm *ProcessManager) IsRunning(sessionID string) bool {
	processInterface, ok := pm.processes.Load(sessionID)
	if !ok {
		return false
	}

	process := processInterface.(*CLIProcess)
	select {
	case <-process.done:
		return false
	default:
		return true
	}
}

// GetProcess returns the CLIProcess for a session, or nil if not found
func (pm *ProcessManager) GetProcess(sessionID string) *CLIProcess {
	processInterface, ok := pm.processes.Load(sessionID)
	if !ok {
		return nil
	}
	return processInterface.(*CLIProcess)
}

// GetRunningCount returns the number of running processes
func (pm *ProcessManager) GetRunningCount() int {
	count := 0
	pm.processes.Range(func(key, value interface{}) bool {
		process := value.(*CLIProcess)
		select {
		case <-process.done:
			// Process has exited, don't count it
		default:
			count++
		}
		return true
	})
	return count
}

// GetAllProcesses returns a slice of all running processes
func (pm *ProcessManager) GetAllProcesses() []*CLIProcess {
	processes := []*CLIProcess{}
	pm.processes.Range(func(key, value interface{}) bool {
		process := value.(*CLIProcess)
		select {
		case <-process.done:
			// Process has exited, don't include it
		default:
			processes = append(processes, process)
		}
		return true
	})
	return processes
}
