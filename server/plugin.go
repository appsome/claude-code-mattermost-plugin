package main

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// botUserID is the ID of the bot user created by the plugin
	botUserID string

	// bridgeClient is the HTTP client for the Claude Code bridge server
	bridgeClient *BridgeClient

	// wsClient is the WebSocket client for real-time updates from the bridge
	wsClient *WebSocketClient
}

// OnActivate is invoked when the plugin is activated.
func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()

	// Ensure the bot user exists
	bot := &model.Bot{
		Username:    "claude-code",
		DisplayName: "Claude Code",
		Description: "AI-powered coding assistant",
	}
	
	// Try to create bot (will fail if it already exists, which is fine)
	createdBot, appErr := p.API.CreateBot(bot)
	if appErr != nil {
		// Bot might already exist, try to get it by username
		user, getUserErr := p.API.GetUserByUsername(bot.Username)
		if getUserErr != nil {
			return fmt.Errorf("failed to ensure bot user exists: %w", getUserErr)
		}
		p.botUserID = user.Id
		p.API.LogInfo("Using existing bot user", "user_id", user.Id)
	} else {
		p.botUserID = createdBot.UserId
		p.API.LogInfo("Created new bot user", "user_id", createdBot.UserId)
	}

	// Initialize bridge client
	p.bridgeClient = NewBridgeClient(config.BridgeServerURL, p.API)

	// Initialize WebSocket client for real-time updates
	p.wsClient = NewWebSocketClient(config.BridgeServerURL, p)
	if err := p.wsClient.Connect(); err != nil {
		p.API.LogWarn("Failed to connect to bridge WebSocket", "error", err.Error())
		// Don't fail activation if WebSocket connection fails
	}

	// Register slash commands
	if err := p.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	p.API.LogInfo("Claude Code plugin activated successfully",
		"bot_user_id", p.botUserID,
		"bridge_url", config.BridgeServerURL,
	)

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.wsClient != nil {
		p.wsClient.Close()
	}

	p.API.LogInfo("Claude Code plugin deactivated")
	return nil
}

func main() {
	plugin.ClientMain(&Plugin{})
}
