package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConfiguration_Clone(t *testing.T) {
	original := &configuration{
		BridgeServerURL:      "http://localhost:3001",
		ClaudeCodePath:       "/usr/local/bin/claude-code",
		EnableFileOperations: true,
	}

	cloned := original.Clone()

	// Should be equal
	assert.Equal(t, original.BridgeServerURL, cloned.BridgeServerURL)
	assert.Equal(t, original.ClaudeCodePath, cloned.ClaudeCodePath)
	assert.Equal(t, original.EnableFileOperations, cloned.EnableFileOperations)

	// Should be different pointers
	assert.NotSame(t, original, cloned)

	// Modifying clone should not affect original
	cloned.BridgeServerURL = "http://localhost:3002"
	assert.NotEqual(t, original.BridgeServerURL, cloned.BridgeServerURL)
}

func TestGetConfiguration_Nil(t *testing.T) {
	p := &Plugin{
		configuration: nil,
	}

	config := p.getConfiguration()

	assert.NotNil(t, config)
	assert.Equal(t, "", config.BridgeServerURL)
	assert.Equal(t, "", config.ClaudeCodePath)
	assert.Equal(t, false, config.EnableFileOperations)
}

func TestGetConfiguration_Existing(t *testing.T) {
	expected := &configuration{
		BridgeServerURL:      "http://localhost:3001",
		ClaudeCodePath:       "/usr/local/bin/claude-code",
		EnableFileOperations: true,
	}

	p := &Plugin{
		configuration: expected,
	}

	config := p.getConfiguration()

	assert.Equal(t, expected, config)
	assert.Same(t, expected, config)
}

func TestSetConfiguration_New(t *testing.T) {
	p := &Plugin{}

	newConfig := &configuration{
		BridgeServerURL: "http://localhost:3001",
	}

	p.setConfiguration(newConfig)

	assert.Equal(t, newConfig, p.configuration)
}

func TestSetConfiguration_Different(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}

	newConfig := &configuration{
		BridgeServerURL: "http://localhost:3002",
	}

	p.setConfiguration(newConfig)

	assert.Equal(t, newConfig, p.configuration)
}

func TestSetConfiguration_SamePointer(t *testing.T) {
	config := &configuration{
		BridgeServerURL: "http://localhost:3001",
	}

	p := &Plugin{
		configuration: config,
	}

	// Should panic when setting the same pointer
	assert.Panics(t, func() {
		p.setConfiguration(config)
	})
}

func TestSetConfiguration_EmptyConfig(t *testing.T) {
	emptyConfig := &configuration{}

	p := &Plugin{
		configuration: emptyConfig,
	}

	// Should not panic for empty config (Go optimization)
	assert.NotPanics(t, func() {
		p.setConfiguration(emptyConfig)
	})
}

func TestOnConfigurationChange_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := &Plugin{}
	p.SetAPI(api)

	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).
		Run(func(args mock.Arguments) {
			config := args.Get(0).(*configuration)
			config.BridgeServerURL = "http://localhost:3001"
			config.ClaudeCodePath = "/usr/local/bin/claude-code"
			config.EnableFileOperations = true
		}).
		Return(nil)

	err := p.OnConfigurationChange()

	assert.NoError(t, err)
	assert.NotNil(t, p.configuration)
	assert.Equal(t, "http://localhost:3001", p.configuration.BridgeServerURL)
	assert.Equal(t, "/usr/local/bin/claude-code", p.configuration.ClaudeCodePath)
	assert.Equal(t, true, p.configuration.EnableFileOperations)
}

func TestOnConfigurationChange_LoadError(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := &Plugin{}
	p.SetAPI(api)

	loadError := errors.New("load error")
	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).
		Return(loadError)

	err := p.OnConfigurationChange()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load plugin configuration")
}

func TestConfiguration_ConcurrentAccess(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}

	// Start multiple goroutines reading configuration
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			config := p.getConfiguration()
			assert.NotNil(t, config)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
