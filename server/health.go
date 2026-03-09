package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

// HealthStatus represents the health status of the plugin
type HealthStatus struct {
	Status          string `json:"status"`
	Mode            string `json:"mode"`
	BridgeConnected bool   `json:"bridge_connected,omitempty"`
	BridgeURL       string `json:"bridge_url,omitempty"`
	CLIAvailable    bool   `json:"cli_available,omitempty"`
	CLIPath         string `json:"cli_path,omitempty"`
	ActiveSessions  int    `json:"active_sessions"`
	Timestamp       string `json:"timestamp"`
}

// BridgeHealthResponse represents the response from the bridge server health endpoint
type BridgeHealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    int    `json:"uptime"`
	Sessions  int    `json:"sessions"`
	Timestamp string `json:"timestamp"`
}

// CheckBridgeHealth checks if the bridge server is healthy
func (p *Plugin) CheckBridgeHealth() (*BridgeHealthResponse, error) {
	config := p.getConfiguration()
	if config.BridgeServerURL == "" {
		return nil, fmt.Errorf("bridge server URL not configured")
	}

	url := config.BridgeServerURL + "/health"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bridge server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bridge server returned status %d", resp.StatusCode)
	}

	var health BridgeHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &health, nil
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
	config := p.getConfiguration()

	status := &HealthStatus{
		Status:         "ok",
		ActiveSessions: 0,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	if p.UseBridgeMode() {
		// Bridge mode
		status.Mode = "bridge"
		status.BridgeURL = config.BridgeServerURL

		bridgeHealth, err := p.CheckBridgeHealth()
		if err != nil {
			p.API.LogWarn("Bridge health check failed", "error", err.Error())
			status.Status = "degraded"
			status.BridgeConnected = false
		} else {
			status.BridgeConnected = true
			status.ActiveSessions = bridgeHealth.Sessions
		}
	} else {
		// Embedded mode
		status.Mode = "embedded"

		cliAvailable, cliPath := p.CheckCLIAvailability()
		status.CLIAvailable = cliAvailable
		status.CLIPath = cliPath

		if p.processManager != nil {
			status.ActiveSessions = p.processManager.GetRunningCount()
		}

		if !cliAvailable {
			status.Status = "degraded"
		}
	}

	return status
}

// IsBridgeHealthy returns true if the bridge server is reachable and healthy
func (p *Plugin) IsBridgeHealthy() bool {
	health, err := p.CheckBridgeHealth()
	if err != nil {
		return false
	}
	return health.Status == "ok"
}

// IsCLIHealthy returns true if the Claude Code CLI is available
func (p *Plugin) IsCLIHealthy() bool {
	available, _ := p.CheckCLIAvailability()
	return available
}
