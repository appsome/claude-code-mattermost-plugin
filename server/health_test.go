package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckBridgeHealth_Success(t *testing.T) {
	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BridgeHealthResponse{
			Status:    "ok",
			Version:   "1.0.0",
			Uptime:    3600,
			Sessions:  5,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "ok", health.Status)
	assert.Equal(t, "1.0.0", health.Version)
	assert.Equal(t, 3600, health.Uptime)
	assert.Equal(t, 5, health.Sessions)
}

func TestCheckBridgeHealth_NoURL(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "",
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "bridge server URL not configured")
}

func TestCheckBridgeHealth_ConnectionError(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:99999",
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "failed to connect to bridge server")
}

func TestCheckBridgeHealth_BadStatus(t *testing.T) {
	// Setup mock bridge server returning 500
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "bridge server returned status")
}

func TestCheckBridgeHealth_InvalidJSON(t *testing.T) {
	// Setup mock bridge server returning invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "failed to decode health response")
}

func TestCheckBridgeHealth_Timeout(t *testing.T) {
	// Setup mock bridge server with delay
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	health, err := p.CheckBridgeHealth()

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "failed to connect to bridge server")
}

func TestGetHealthStatus_BridgeHealthy(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BridgeHealthResponse{
			Status:    "ok",
			Version:   "1.0.0",
			Uptime:    3600,
			Sessions:  5,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}
	p.SetAPI(api)

	status := p.GetHealthStatus()

	assert.NotNil(t, status)
	assert.Equal(t, "ok", status.Status)
	assert.True(t, status.BridgeConnected)
	assert.Equal(t, mockServer.URL, status.BridgeURL)
	assert.Equal(t, 5, status.ActiveSessions)
	assert.NotEmpty(t, status.Timestamp)
}

func TestGetHealthStatus_BridgeUnhealthy(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:99999",
		},
	}
	p.SetAPI(api)

	status := p.GetHealthStatus()

	assert.NotNil(t, status)
	assert.Equal(t, "degraded", status.Status)
	assert.False(t, status.BridgeConnected)
	assert.Equal(t, "http://localhost:99999", status.BridgeURL)
	assert.Equal(t, 0, status.ActiveSessions)
	assert.NotEmpty(t, status.Timestamp)
}

func TestIsBridgeHealthy_True(t *testing.T) {
	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BridgeHealthResponse{
			Status: "ok",
		})
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	healthy := p.IsBridgeHealthy()

	assert.True(t, healthy)
}

func TestIsBridgeHealthy_False_ConnectionError(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:99999",
		},
	}

	healthy := p.IsBridgeHealthy()

	assert.False(t, healthy)
}

func TestIsBridgeHealthy_False_BadStatus(t *testing.T) {
	// Setup mock bridge server returning degraded status
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BridgeHealthResponse{
			Status: "degraded",
		})
	}))
	defer mockServer.Close()

	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: mockServer.URL,
		},
	}

	healthy := p.IsBridgeHealthy()

	assert.False(t, healthy)
}

func TestHealthStatus_JSONSerialization(t *testing.T) {
	status := &HealthStatus{
		Status:          "ok",
		BridgeConnected: true,
		BridgeURL:       "http://localhost:3001",
		ActiveSessions:  5,
		Timestamp:       "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(status)
	assert.NoError(t, err)

	var decoded HealthStatus
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, status.Status, decoded.Status)
	assert.Equal(t, status.BridgeConnected, decoded.BridgeConnected)
	assert.Equal(t, status.BridgeURL, decoded.BridgeURL)
	assert.Equal(t, status.ActiveSessions, decoded.ActiveSessions)
	assert.Equal(t, status.Timestamp, decoded.Timestamp)
}

func TestBridgeHealthResponse_JSONSerialization(t *testing.T) {
	response := &BridgeHealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Uptime:    3600,
		Sessions:  5,
		Timestamp: "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var decoded BridgeHealthResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, response.Status, decoded.Status)
	assert.Equal(t, response.Version, decoded.Version)
	assert.Equal(t, response.Uptime, decoded.Uptime)
	assert.Equal(t, response.Sessions, decoded.Sessions)
	assert.Equal(t, response.Timestamp, decoded.Timestamp)
}
