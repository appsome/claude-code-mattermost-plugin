package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestNewProcessManager(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)

	pm := NewProcessManager(plugin)

	assert.NotNil(t, pm)
	assert.Equal(t, plugin, pm.plugin)
}

func TestProcessManagerIsRunning(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	tests := []struct {
		name      string
		sessionID string
		want      bool
	}{
		{
			name:      "non-existent session",
			sessionID: "nonexistent",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.IsRunning(tt.sessionID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessManagerGetProcess(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	tests := []struct {
		name      string
		sessionID string
		want      *CLIProcess
	}{
		{
			name:      "non-existent session",
			sessionID: "nonexistent",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.GetProcess(tt.sessionID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessManagerGetRunningCount(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Initially should be 0
	count := pm.GetRunningCount()
	assert.Equal(t, 0, count)
}

func TestProcessManagerGetAllProcesses(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Initially should be empty
	processes := pm.GetAllProcesses()
	assert.NotNil(t, processes)
	assert.Len(t, processes, 0)
}

func TestProcessManagerKillNonExistent(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Killing a non-existent process should not error
	err := pm.Kill("nonexistent")
	assert.NoError(t, err)
}

func TestProcessManagerKillAll(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// With no processes, should not panic
	pm.KillAll()

	// Should still have 0 processes after
	count := pm.GetRunningCount()
	assert.Equal(t, 0, count)
}

func TestCLIProcessStructure(t *testing.T) {
	// Test that CLIProcess structure can be created
	process := &CLIProcess{
		SessionID:   "test-session",
		StartTime:   time.Now(),
		ProjectPath: "/tmp/test",
		ChannelID:   "channel123",
		UserID:      "user123",
		done:        make(chan struct{}),
	}

	assert.NotNil(t, process)
	assert.Equal(t, "test-session", process.SessionID)
	assert.Equal(t, "/tmp/test", process.ProjectPath)
	assert.Equal(t, "channel123", process.ChannelID)
	assert.Equal(t, "user123", process.UserID)
	assert.NotNil(t, process.done)
	assert.False(t, process.StartTime.IsZero())
}

func TestCLIProcessDoneChannel(t *testing.T) {
	process := &CLIProcess{
		SessionID: "test-session",
		done:      make(chan struct{}),
	}

	// Test that done channel is open initially
	select {
	case <-process.done:
		t.Fatal("done channel should be open initially")
	default:
		// Expected: channel is open but not closed
	}

	// Close the channel
	close(process.done)

	// Test that done channel is now closed
	select {
	case <-process.done:
		// Expected: channel is closed
	default:
		t.Fatal("done channel should be closed")
	}
}

// TestProcessManagerSpawnWithoutCLI tests Spawn when CLI is not available
func TestProcessManagerSpawnWithoutCLI(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.configuration = &configuration{
		ClaudeCodePath: "/nonexistent/claude",
	}

	pm := NewProcessManager(plugin)

	err := pm.Spawn("session1", "/tmp/project", "channel1", "user1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestProcessManagerSpawnDuplicate tests that spawning duplicate session fails
func TestProcessManagerSpawnDuplicate(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Manually add a fake process to simulate existing session
	process := &CLIProcess{
		SessionID: "session1",
		done:      make(chan struct{}),
	}
	pm.processes.Store("session1", process)

	// Try to spawn duplicate
	err := pm.Spawn("session1", "/tmp/project", "channel1", "user1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already has a running process")

	// Clean up
	pm.processes.Delete("session1")
}

// TestProcessManagerSendInputNonExistent tests sending input to non-existent process
func TestProcessManagerSendInputNonExistent(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	err := pm.SendInput("nonexistent", "test input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running process")
}

// TestProcessManagerSendInputJSONNonExistent tests sending JSON to non-existent process
func TestProcessManagerSendInputJSONNonExistent(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	err := pm.SendInputJSON("nonexistent", map[string]string{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running process")
}

// TestProcessManagerSendInputJSONInvalidData tests sending invalid JSON
func TestProcessManagerSendInputJSONInvalidData(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Try to marshal invalid data (channels can't be marshaled)
	invalidData := make(chan int)
	err := pm.SendInputJSON("session1", invalidData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal")
}

// TestProcessManagerMultipleSessions tests managing multiple sessions
func TestProcessManagerMultipleSessions(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Add multiple fake processes
	sessions := []string{"session1", "session2", "session3"}
	for _, sessionID := range sessions {
		process := &CLIProcess{
			SessionID: sessionID,
			done:      make(chan struct{}),
		}
		pm.processes.Store(sessionID, process)
	}

	// Test IsRunning for all
	for _, sessionID := range sessions {
		assert.True(t, pm.IsRunning(sessionID))
	}

	// Test GetRunningCount
	count := pm.GetRunningCount()
	assert.Equal(t, 3, count)

	// Test GetAllProcesses
	processes := pm.GetAllProcesses()
	assert.Len(t, processes, 3)

	// Test GetProcess for each
	for _, sessionID := range sessions {
		process := pm.GetProcess(sessionID)
		assert.NotNil(t, process)
		assert.Equal(t, sessionID, process.SessionID)
	}

	// Close one process
	processInterface, _ := pm.processes.Load("session1")
	process1 := processInterface.(*CLIProcess)
	close(process1.done)

	// Running count should now be 2
	count = pm.GetRunningCount()
	assert.Equal(t, 2, count)

	// IsRunning should return false for closed process
	assert.False(t, pm.IsRunning("session1"))
	assert.True(t, pm.IsRunning("session2"))
	assert.True(t, pm.IsRunning("session3"))

	// Clean up
	pm.processes.Delete("session1")
	pm.processes.Delete("session2")
	pm.processes.Delete("session3")
}

// TestProcessManagerConcurrentAccess tests thread-safe access
func TestProcessManagerConcurrentAccess(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	pm := NewProcessManager(plugin)

	// Add some processes
	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session%d", i)
		process := &CLIProcess{
			SessionID: sessionID,
			done:      make(chan struct{}),
		}
		pm.processes.Store(sessionID, process)
	}

	// Concurrently access the processes
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				sessionID := fmt.Sprintf("session%d", j)
				pm.IsRunning(sessionID)
				pm.GetProcess(sessionID)
			}
			pm.GetRunningCount()
			pm.GetAllProcesses()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should still have all processes
	count := pm.GetRunningCount()
	assert.Equal(t, 10, count)

	// Clean up
	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session%d", i)
		pm.processes.Delete(sessionID)
	}
}
