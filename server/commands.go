package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	commandTriggerClaude       = "claude"
	commandTriggerClaudeStart  = "claude-start"
	commandTriggerClaudeStop   = "claude-stop"
	commandTriggerClaudeStatus = "claude-status"
	commandTriggerClaudeThread = "claude-thread"
	commandTriggerClaudeHelp   = "claude-help"
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

	// Register /claude-status command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaudeStatus,
		AutoComplete:     true,
		AutoCompleteDesc: "Show current session status",
		DisplayName:      "Claude Session Status",
		Description:      "Display information about the current session",
	}); err != nil {
		return err
	}

	// Register /claude-thread command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTriggerClaudeThread,
		AutoComplete:     true,
		AutoCompleteDesc: "Add thread context to Claude session",
		AutoCompleteHint: "[action]",
		DisplayName:      "Claude Thread Context",
		Description:      "Add conversation history from this thread to Claude's context",
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
	split := strings.Fields(args.Command)
	if len(split) == 0 {
		return respondEphemeral("Invalid command"), nil
	}

	trigger := strings.TrimPrefix(split[0], "/")
	commandArgs := strings.TrimSpace(strings.TrimPrefix(args.Command, split[0]))

	switch trigger {
	case commandTriggerClaude:
		return p.executeClaude(args, commandArgs), nil
	case commandTriggerClaudeStart:
		return p.executeClaudeStart(args, commandArgs), nil
	case commandTriggerClaudeStop:
		return p.executeClaudeStop(args), nil
	case commandTriggerClaudeStatus:
		return p.executeClaudeStatus(args), nil
	case commandTriggerClaudeThread:
		return p.executeClaudeThread(args, commandArgs), nil
	case commandTriggerClaudeHelp:
		return p.executeClaudeHelp(args), nil
	default:
		return respondEphemeral(fmt.Sprintf("Unknown command: %s", trigger)), nil
	}
}

// executeClaude handles the /claude <message> command
func (p *Plugin) executeClaude(args *model.CommandArgs, message string) *model.CommandResponse {
	if message == "" {
		return respondEphemeral("Please provide a message. Usage: `/claude <your message>`")
	}

	// Get active session
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active session. Use `/claude-start [project-path]` to begin.")
	}

	// Send message to bridge server
	if err := p.bridgeClient.SendMessage(session.SessionID, message); err != nil {
		p.API.LogError("Failed to send message to bridge", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to send message: %s", err.Error()))
	}

	// Update last message timestamp
	if err := p.UpdateSessionLastMessage(args.ChannelId); err != nil {
		p.API.LogWarn("Failed to update last message timestamp", "error", err.Error())
	}

	// Post user's message as a regular post (not ephemeral)
	userPost := &model.Post{
		ChannelId: args.ChannelId,
		UserId:    args.UserId,
		Message:   fmt.Sprintf("**Claude:** %s", message),
	}
	if _, appErr := p.API.CreatePost(userPost); appErr != nil {
		p.API.LogWarn("Failed to create user message post", "error", appErr.Error())
	}

	// Response will come via WebSocket, so return empty response
	return &model.CommandResponse{}
}

// executeClaudeStart handles the /claude-start [project-path] command
func (p *Plugin) executeClaudeStart(args *model.CommandArgs, projectPath string) *model.CommandResponse {
	if projectPath == "" {
		// TODO: In Issue #5, show interactive dialog for project selection
		return respondEphemeral("Please provide a project path. Usage: `/claude-start /path/to/project`")
	}

	// Check if session already exists
	existing, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to check for existing session", "error", err.Error())
		return respondEphemeral("❌ Error checking for existing session. Please try again.")
	}

	if existing != nil {
		return respondEphemeral(fmt.Sprintf("⚠️ This channel already has an active session for project: `%s`\nUse `/claude-stop` to end it first.", existing.ProjectPath))
	}

	// Create new session
	session, err := p.CreateSession(args.ChannelId, projectPath, args.UserId)
	if err != nil {
		p.API.LogError("Failed to create session", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to start session: %s", err.Error()))
	}

	// Post success message from bot
	successMsg := fmt.Sprintf("🚀 Started Claude Code session\n\n**Project:** `%s`\n**Session ID:** `%s`\n\nYou can now send messages with `/claude <your message>`", projectPath, session.SessionID)
	p.postBotMessage(args.ChannelId, successMsg)

	return &model.CommandResponse{}
}

// executeClaudeStop handles the /claude-stop command
func (p *Plugin) executeClaudeStop(args *model.CommandArgs) *model.CommandResponse {
	// Check if session exists
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active session to stop. Use `/claude-start` to begin a new one.")
	}

	// Stop the session
	if err := p.StopSession(args.ChannelId); err != nil {
		p.API.LogError("Failed to stop session", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to stop session: %s", err.Error()))
	}

	// Post success message from bot
	p.postBotMessage(args.ChannelId, "✅ Session stopped successfully. Use `/claude-start` to begin a new one.")

	return &model.CommandResponse{}
}

// executeClaudeStatus handles the /claude-status command
func (p *Plugin) executeClaudeStatus(args *model.CommandArgs) *model.CommandResponse {
	// Get active session
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active session. Use `/claude-start [project-path]` to begin.")
	}

	// Get session details from bridge
	bridgeSession, err := p.bridgeClient.GetSession(session.SessionID)
	if err != nil {
		p.API.LogError("Failed to get session from bridge", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to get session details: %s", err.Error()))
	}

	// Calculate uptime
	uptime := formatDuration(session.CreatedAt)
	lastMessage := formatDuration(session.LastMessageAt)

	// Get message count
	messages, err := p.bridgeClient.GetMessages(session.SessionID, 0)
	messageCount := 0
	if err == nil {
		messageCount = len(messages)
	}

	// Format status
	statusMsg := fmt.Sprintf(`### 📊 Session Status

**Project:** %s
**Session ID:** %s
**Status:** %s
**Uptime:** %s
**Last Message:** %s
**Message Count:** %d
**CLI Process:** %s`,
		session.ProjectPath,
		session.SessionID,
		bridgeSession.Status,
		uptime,
		lastMessage,
		messageCount,
		formatPID(bridgeSession.CLIPid),
	)

	return respondEphemeral(statusMsg)
}

// executeClaudeThread handles the /claude-thread [action] command
func (p *Plugin) executeClaudeThread(args *model.CommandArgs, action string) *model.CommandResponse {
	// Check if in a thread
	if args.RootId == "" {
		return respondEphemeral("⚠️ This command must be run in a thread. Reply to a message first.")
	}

	// Get active session
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active Claude session. Start one with `/claude-start [project-path]` first.")
	}

	// Get thread context
	maxMessages := defaultMaxThreadMessages
	// TODO: Add configuration option for max messages

	threadContext, err := p.GetThreadContext(args.RootId, args.ChannelId, maxMessages)
	if err != nil {
		p.API.LogError("Failed to get thread context", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to retrieve thread context: %s", err.Error()))
	}

	// Check for privacy concerns (many participants)
	if len(threadContext.Participants) > 5 {
		warningMsg := fmt.Sprintf("⚠️ This thread has %d participants. Context includes messages from:\n%s\n\nAre you sure you want to share this with Claude?",
			len(threadContext.Participants),
			strings.Join(threadContext.Participants, ", "))
		
		// For now, just warn. In the future, could add confirmation dialog
		p.API.LogWarn("Thread context includes many participants", 
			"count", len(threadContext.Participants),
			"participants", threadContext.Participants)
		
		// Could return warning here, but for MVP let's proceed
		_ = warningMsg
	}

	// Send context to bridge server
	if err := p.SendThreadContext(session.SessionID, threadContext, action); err != nil {
		p.API.LogError("Failed to send thread context to bridge", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to send context: %s", err.Error()))
	}

	// Update last message timestamp
	if err := p.UpdateSessionLastMessage(args.ChannelId); err != nil {
		p.API.LogWarn("Failed to update last message timestamp", "error", err.Error())
	}

	// Build success message
	successMsg := fmt.Sprintf("✅ Added %d messages from this thread to Claude's context.", threadContext.MessageCount)
	if action != "" {
		successMsg += fmt.Sprintf("\nClaude will now: **%s**", action)
	}
	
	// Post confirmation from bot
	p.postBotMessage(args.ChannelId, successMsg)

	return &model.CommandResponse{}
}

// executeClaudeHelp handles the /claude-help command
func (p *Plugin) executeClaudeHelp(args *model.CommandArgs) *model.CommandResponse {
	helpText := "### Claude Code - AI Coding Assistant\n\n" +
		"**Available Commands:**\n" +
		"- `/claude <message>` - Send a message to Claude Code\n" +
		"- `/claude-start [project-path]` - Start a new coding session\n" +
		"- `/claude-stop` - Stop the current session\n" +
		"- `/claude-status` - Show current session status\n" +
		"- `/claude-thread [action]` - Add thread context to Claude (run in a thread)\n" +
		"- `/claude-help` - Show this help message\n\n" +
		"**Getting Started:**\n" +
		"1. Start a session with `/claude-start /path/to/your/project`\n" +
		"2. Send messages with `/claude <your question or request>`\n" +
		"3. Claude Code will respond with suggestions and actions\n" +
		"4. Stop the session with `/claude-stop` when done\n\n" +
		"**Examples:**\n" +
		"- `/claude add a login form with email and password`\n" +
		"- `/claude refactor the user service to use async/await`\n" +
		"- `/claude write unit tests for the auth module`\n\n" +
		"**Thread Context:**\n" +
		"- `/claude-thread` - Add thread messages to Claude's context\n" +
		"- `/claude-thread summarize` - Add context and ask Claude to summarize\n" +
		"- `/claude-thread implement` - Add context and ask Claude to implement\n" +
		"- `/claude-thread review` - Add context and ask Claude to review\n\n" +
		"**Configuration:**\n" +
		"Go to **System Console → Plugins → Claude Code** to configure the bridge server URL and settings.\n\n" +
		"For more information, visit: https://github.com/appsome/claude-code-mattermost-plugin"

	return respondEphemeral(helpText)
}

// Helper functions

func respondEphemeral(message string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         message,
	}
}

func formatDuration(unixTimestamp int64) string {
	// Simple duration formatting (could be enhanced)
	return fmt.Sprintf("<t:%d:R>", unixTimestamp) // Discord-style relative timestamp
}

func formatPID(pid *int) string {
	if pid == nil {
		return "Not running"
	}
	return fmt.Sprintf("PID %d", *pid)
}
