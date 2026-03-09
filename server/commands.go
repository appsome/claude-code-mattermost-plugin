package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	commandTriggerClaude      = "claude"
	commandTriggerClaudeStart = "claude-start"
	commandTriggerClaudeStop  = "claude-stop"
	commandTriggerClaudeHelp  = "claude-help"
)

func (p *Plugin) registerCommands() error {
	// Register /claude command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaude,
		AutoComplete:     true,
		AutoCompleteDesc: "Send a message to Claude Code",
		AutoCompleteHint: "[message]",
		DisplayName:      "Claude Code",
		Description:      "Interact with Claude Code AI assistant",
	}); err != nil {
		return err
	}

	// Register /claude-start command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaudeStart,
		AutoComplete:     true,
		AutoCompleteDesc: "Start a new Claude Code session",
		AutoCompleteHint: "[project-path]",
		DisplayName:      "Start Claude Session",
		Description:      "Start a new coding session with Claude Code",
	}); err != nil {
		return err
	}

	// Register /claude-stop command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaudeStop,
		AutoComplete:     true,
		AutoCompleteDesc: "Stop the current Claude Code session",
		DisplayName:      "Stop Claude Session",
		Description:      "Stop the active coding session",
	}); err != nil {
		return err
	}

	// Register /claude-help command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaudeHelp,
		AutoComplete:     true,
		AutoCompleteDesc: "Show Claude Code help",
		DisplayName:      "Claude Code Help",
		Description:      "Display help information for Claude Code plugin",
	}); err != nil {
		return err
	}

	return nil
}

// ExecuteCommand handles slash command execution
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")

	switch trigger {
	case commandTriggerClaude:
		return p.executeClaude(args), nil
	case commandTriggerClaudeStart:
		return p.executeClaudeStart(args), nil
	case commandTriggerClaudeStop:
		return p.executeClaudeStop(args), nil
	case commandTriggerClaudeHelp:
		return p.executeClaudeHelp(args), nil
	default:
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Unknown command: %s", trigger),
		}, nil
	}
}

func (p *Plugin) executeClaude(args *model.CommandArgs) *model.CommandResponse {
	// TODO: Implement message sending
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "⚠️ Command not yet implemented. This will be added in Issue #4.",
	}
}

func (p *Plugin) executeClaudeStart(args *model.CommandArgs) *model.CommandResponse {
	// TODO: Implement session start
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "⚠️ Command not yet implemented. This will be added in Issue #4.",
	}
}

func (p *Plugin) executeClaudeStop(args *model.CommandArgs) *model.CommandResponse {
	// TODO: Implement session stop
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "⚠️ Command not yet implemented. This will be added in Issue #4.",
	}
}

func (p *Plugin) executeClaudeHelp(args *model.CommandArgs) *model.CommandResponse {
	helpText := `### Claude Code - AI Coding Assistant

**Available Commands:**
- \`/claude <message>\` - Send a message to Claude Code
- \`/claude-start [project-path]\` - Start a new coding session
- \`/claude-stop\` - Stop the current session
- \`/claude-help\` - Show this help message

**Getting Started:**
1. Start a session with \`/claude-start /path/to/your/project\`
2. Send messages with \`/claude <your question or request>\`
3. Approve/reject suggested changes using interactive buttons
4. Stop the session with \`/claude-stop\`

For more information, visit: https://github.com/appsome/claude-code-mattermost-plugin
`

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         helpText,
	}
}
