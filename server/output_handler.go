package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// OutputHandler processes output from CLI processes and routes to Mattermost
type OutputHandler struct {
	plugin        *Plugin
	outputBuffers sync.Map // map[sessionID]*OutputBuffer
}

// OutputBuffer buffers partial output for a session
type OutputBuffer struct {
	sessionID   string
	channelID   string
	buffer      strings.Builder
	lastFlush   time.Time
	mu          sync.Mutex
	pendingPost *model.Post // Current post being updated
}

// CLIOutputMessage represents a JSON message from Claude Code CLI
type CLIOutputMessage struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Message   string          `json:"message,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`

	// For assistant messages with content blocks
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`

	// For tool results
	ToolName   string `json:"tool_name,omitempty"`
	ToolResult string `json:"tool_result,omitempty"`

	// For file changes
	FilePath   string `json:"file_path,omitempty"`
	ChangeType string `json:"change_type,omitempty"`
	Diff       string `json:"diff,omitempty"`

	// For result messages
	Result     string `json:"result,omitempty"`
	TotalCost  string `json:"total_cost,omitempty"`
	TotalUsage *Usage `json:"total_usage,omitempty"`

	// For errors
	Error string `json:"error,omitempty"`
}

// ContentBlock represents a content block in the output
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewOutputHandler creates a new OutputHandler
func NewOutputHandler(plugin *Plugin) *OutputHandler {
	return &OutputHandler{
		plugin: plugin,
	}
}

// getOrCreateBuffer gets or creates an output buffer for a session
func (oh *OutputHandler) getOrCreateBuffer(sessionID, channelID string) *OutputBuffer {
	bufferInterface, loaded := oh.outputBuffers.LoadOrStore(sessionID, &OutputBuffer{
		sessionID: sessionID,
		channelID: channelID,
		lastFlush: time.Now(),
	})
	buf := bufferInterface.(*OutputBuffer)
	if !loaded {
		buf.channelID = channelID
	}
	return buf
}

// HandleOutput processes stdout from the CLI
func (oh *OutputHandler) HandleOutput(sessionID, channelID, data string) {
	// Try to parse as JSON
	var msg CLIOutputMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		// Not valid JSON, treat as raw text
		oh.postRawMessage(sessionID, channelID, data)
		return
	}

	oh.processMessage(sessionID, channelID, &msg)
}

// processMessage handles a parsed CLI output message
func (oh *OutputHandler) processMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	switch msg.Type {
	case "assistant":
		oh.handleAssistantMessage(sessionID, channelID, msg)
	case "user":
		// User messages are typically echoed, we can skip or log
		oh.plugin.API.LogDebug("User message from CLI",
			"sessionID", sessionID,
			"message", msg.Message,
		)
	case "system":
		oh.handleSystemMessage(sessionID, channelID, msg)
	case "result":
		oh.handleResultMessage(sessionID, channelID, msg)
	case "tool_use":
		oh.handleToolUseMessage(sessionID, channelID, msg)
	case "tool_result":
		oh.handleToolResultMessage(sessionID, channelID, msg)
	case "error":
		oh.HandleError(sessionID, channelID, msg.Error)
	default:
		// Log unknown message types for debugging
		oh.plugin.API.LogDebug("Unknown CLI message type",
			"sessionID", sessionID,
			"type", msg.Type,
			"data", fmt.Sprintf("%+v", msg),
		)
	}
}

// handleAssistantMessage processes assistant responses
func (oh *OutputHandler) handleAssistantMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	var content string

	// Extract text from content blocks if present
	if len(msg.ContentBlocks) > 0 {
		var parts []string
		for _, block := range msg.ContentBlocks {
			if block.Type == "text" && block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		content = strings.Join(parts, "\n")
	} else if msg.Message != "" {
		content = msg.Message
	} else if len(msg.Content) > 0 {
		// Try to extract content from raw JSON
		var rawContent string
		if err := json.Unmarshal(msg.Content, &rawContent); err == nil {
			content = rawContent
		} else {
			content = string(msg.Content)
		}
	}

	if content == "" {
		return
	}

	oh.postBotMessage(channelID, content)
}

// handleSystemMessage processes system messages
func (oh *OutputHandler) handleSystemMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	if msg.Message == "" {
		return
	}

	// Post system messages as italicized
	content := fmt.Sprintf("_%s_", msg.Message)
	oh.postBotMessage(channelID, content)
}

// handleResultMessage processes result/completion messages
func (oh *OutputHandler) handleResultMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	var content string

	if msg.Result != "" {
		content = msg.Result
	}

	// Add usage info if available
	if msg.TotalCost != "" || msg.TotalUsage != nil {
		var usageInfo []string
		if msg.TotalCost != "" {
			usageInfo = append(usageInfo, fmt.Sprintf("Cost: %s", msg.TotalCost))
		}
		if msg.TotalUsage != nil {
			usageInfo = append(usageInfo, fmt.Sprintf("Tokens: %d in / %d out",
				msg.TotalUsage.InputTokens, msg.TotalUsage.OutputTokens))
		}
		if len(usageInfo) > 0 {
			if content != "" {
				content += "\n\n"
			}
			content += fmt.Sprintf("_(%s)_", strings.Join(usageInfo, ", "))
		}
	}

	if content != "" {
		oh.postBotMessage(channelID, content)
	}
}

// handleToolUseMessage processes tool use announcements
func (oh *OutputHandler) handleToolUseMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	// For tool use, we might want to show what tool is being used
	if msg.ToolName != "" {
		content := fmt.Sprintf(":wrench: Using tool: **%s**", msg.ToolName)
		oh.postBotMessage(channelID, content)
	}
}

// handleToolResultMessage processes tool results
func (oh *OutputHandler) handleToolResultMessage(sessionID, channelID string, msg *CLIOutputMessage) {
	// For file changes, show a more informative message
	if msg.FilePath != "" && msg.ChangeType != "" {
		oh.handleFileChange(sessionID, channelID, msg)
		return
	}

	// For other tool results, we typically don't need to show them
	// as the assistant will summarize them
}

// handleFileChange processes file change notifications
func (oh *OutputHandler) handleFileChange(sessionID, channelID string, msg *CLIOutputMessage) {
	var emoji string
	switch msg.ChangeType {
	case "create":
		emoji = ":new:"
	case "modify", "edit":
		emoji = ":pencil2:"
	case "delete":
		emoji = ":wastebasket:"
	default:
		emoji = ":page_facing_up:"
	}

	content := fmt.Sprintf("%s **%s**: `%s`", emoji, msg.ChangeType, msg.FilePath)

	// Add diff if available and not too long
	if msg.Diff != "" {
		diffLines := strings.Split(msg.Diff, "\n")
		if len(diffLines) <= 20 {
			content += fmt.Sprintf("\n```diff\n%s\n```", msg.Diff)
		} else {
			content += fmt.Sprintf("\n```diff\n%s\n... (%d more lines)\n```",
				strings.Join(diffLines[:15], "\n"),
				len(diffLines)-15,
			)
		}
	}

	// Create post with interactive buttons
	post := &model.Post{
		ChannelId: channelID,
		UserId:    oh.plugin.botUserID,
		Message:   content,
	}

	// Add interactive buttons for file changes
	if msg.ChangeType == "modify" || msg.ChangeType == "edit" || msg.ChangeType == "create" {
		post.SetProps(map[string]interface{}{
			"attachments": []map[string]interface{}{
				{
					"actions": []map[string]interface{}{
						{
							"id":    "view_file",
							"name":  "View Full File",
							"type":  "button",
							"style": "default",
							"integration": map[string]interface{}{
								"url": oh.plugin.getPluginURL() + "/api/action/view",
								"context": map[string]interface{}{
									"session_id": sessionID,
									"file_path":  msg.FilePath,
								},
							},
						},
					},
				},
			},
		})
	}

	if _, err := oh.plugin.API.CreatePost(post); err != nil {
		oh.plugin.API.LogError("Failed to post file change",
			"sessionID", sessionID,
			"error", err.Error(),
		)
	}
}

// HandleError processes stderr from the CLI
func (oh *OutputHandler) HandleError(sessionID, channelID, data string) {
	if data == "" {
		return
	}

	// Skip certain known non-error stderr output
	if strings.Contains(data, "Debugger listening") ||
		strings.Contains(data, "For help, see") ||
		strings.Contains(data, "Waiting for the debugger") {
		return
	}

	// Post error messages
	content := fmt.Sprintf(":warning: **Error**: %s", data)
	oh.postBotMessage(channelID, content)
}

// HandleExit processes CLI process exit
func (oh *OutputHandler) HandleExit(sessionID, channelID string, exitCode int) {
	// Clean up buffer
	oh.outputBuffers.Delete(sessionID)

	// Post exit notification
	var content string
	if exitCode == 0 {
		content = ":white_check_mark: Claude Code session completed successfully."
	} else {
		content = fmt.Sprintf(":x: Claude Code session ended with exit code %d.", exitCode)
	}

	oh.postBotMessage(channelID, content)

	// Also clean up the local session
	if err := oh.plugin.DeleteSession(channelID); err != nil {
		oh.plugin.API.LogWarn("Failed to delete session on exit",
			"sessionID", sessionID,
			"error", err.Error(),
		)
	}
}

// postBotMessage posts a message from the bot
func (oh *OutputHandler) postBotMessage(channelID, content string) {
	if content == "" {
		return
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    oh.plugin.botUserID,
		Message:   content,
	}

	if _, err := oh.plugin.API.CreatePost(post); err != nil {
		oh.plugin.API.LogError("Failed to post bot message",
			"channelID", channelID,
			"error", err.Error(),
		)
	}
}

// postRawMessage posts a raw (non-JSON) message
func (oh *OutputHandler) postRawMessage(sessionID, channelID, data string) {
	if data == "" {
		return
	}

	// Wrap in code block if it looks like code or log output
	if strings.Contains(data, "\n") || strings.HasPrefix(data, "{") {
		data = fmt.Sprintf("```\n%s\n```", data)
	}

	oh.postBotMessage(channelID, data)
}
