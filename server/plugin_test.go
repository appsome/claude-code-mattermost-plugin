package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOnActivate_NewBot(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	plugin := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}
	plugin.SetAPI(api)

	// Mock bot creation (success case)
	api.On("CreateBot", mock.AnythingOfType("*model.Bot")).Return(&model.Bot{
		UserId: "bot_user_id",
	}, nil)

	// Mock command registration
	api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).Return(nil)

	// Mock log messages (variadic arguments)
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	err := plugin.OnActivate()
	assert.NoError(t, err)
	assert.Equal(t, "bot_user_id", plugin.botUserID)
	assert.NotNil(t, plugin.bridgeClient)
	assert.NotNil(t, plugin.wsClient)
}

func TestOnActivate_ExistingBot(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	plugin := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}
	plugin.SetAPI(api)

	// Mock bot creation failure (bot already exists)
	api.On("CreateBot", mock.AnythingOfType("*model.Bot")).Return(nil, model.NewAppError("CreateBot", "app.bot.create.error", nil, "already exists", 400))

	// Mock getting existing bot user
	api.On("GetUserByUsername", "claude-code").Return(&model.User{
		Id:       "existing_bot_id",
		Username: "claude-code",
	}, nil)

	// Mock command registration
	api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).Return(nil)

	// Mock log messages (variadic arguments)
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	err := plugin.OnActivate()
	assert.NoError(t, err)
	assert.Equal(t, "existing_bot_id", plugin.botUserID)
	assert.NotNil(t, plugin.bridgeClient)
	assert.NotNil(t, plugin.wsClient)
}

func TestOnActivate_BotCreationFailure(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	plugin := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}
	plugin.SetAPI(api)

	// Mock bot creation failure
	api.On("CreateBot", mock.AnythingOfType("*model.Bot")).Return(nil, model.NewAppError("CreateBot", "app.bot.create.error", nil, "error", 500))

	// Mock getting bot user also fails
	api.On("GetUserByUsername", "claude-code").Return(nil, model.NewAppError("GetUserByUsername", "app.user.get.error", nil, "not found", 404))

	err := plugin.OnActivate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ensure bot user exists")
}

func TestOnActivate_CommandRegistrationFailure(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	plugin := &Plugin{
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}
	plugin.SetAPI(api)

	// Mock bot creation
	api.On("CreateBot", mock.AnythingOfType("*model.Bot")).Return(&model.Bot{
		UserId: "bot_user_id",
	}, nil)

	// Mock command registration failure
	api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).Return(model.NewAppError("RegisterCommand", "app.command.register.error", nil, "error", 500))

	// Mock log messages (variadic arguments)
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	err := plugin.OnActivate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register commands")
}

func TestOnDeactivate(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	// Create a mock WebSocket client
	wsClient := &WebSocketClient{
		baseURL:       "http://localhost:3001",
		subscriptions: make(map[string]string),
		stopChan:      make(chan struct{}),
	}

	plugin := &Plugin{
		wsClient: wsClient,
	}
	plugin.SetAPI(api)

	// Mock log message (variadic)
	api.On("LogInfo", mock.Anything).Maybe().Return()

	err := plugin.OnDeactivate()
	assert.NoError(t, err)
}

func TestOnDeactivate_NoWebSocket(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	plugin := &Plugin{
		wsClient: nil,
	}
	plugin.SetAPI(api)

	// Mock log message (variadic)
	api.On("LogInfo", mock.Anything).Maybe().Return()

	err := plugin.OnDeactivate()
	assert.NoError(t, err)
}
