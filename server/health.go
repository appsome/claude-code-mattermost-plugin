package main

import (
	"os/exec"
	"time"
)

// HealthStatus represents the health status of the plugin
type HealthStatus struct {
	Status         string `json:"status"`
	CLIAvailable   bool   `json:"cli_available"`
	CLIPath        string `json:"cli_path"`
	ActiveSessions int    `json:"active_sessions"`
	Timestamp      string `json:"timestamp"`
}

// CheckCLIAvailability checks if the Claude Code CLI is available
func (p *Plugin) CheckCLIAvailability() (bool, string) {
	config := p.getConfiguration()
	cliPath := config.ClaudeCodePath
	if cliPath == "" {
		cliPath = "claude"
	}

	// Check if CLI exists in PATH or at specified path
	path, err := exec.LookPath(cliPath)
	if err != nil {
		return false, cliPath
	}

	return true, path
}

// GetHealthStatus returns the overall health status of the plugin
func (p *Plugin) GetHealthStatus() *HealthStatus {
	cliAvailable, cliPath := p.CheckCLIAvailability()

	status := &HealthStatus{
		Status:         "ok",
		CLIAvailable:   cliAvailable,
		CLIPath:        cliPath,
		ActiveSessions: 0,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	// Get active session count
	if p.processManager != nil {
		status.ActiveSessions = p.processManager.GetRunningCount()
	}

	// Set status based on CLI availability
	if !cliAvailable {
		status.Status = "degraded"
	}

	return status
}

// IsCLIHealthy returns true if the Claude Code CLI is available
func (p *Plugin) IsCLIHealthy() bool {
	available, _ := p.CheckCLIAvailability()
	return available
}
