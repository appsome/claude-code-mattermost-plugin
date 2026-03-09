package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// FileNode represents a file or directory in the file tree
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	Size     *int64     `json:"size,omitempty"`
	Children []FileNode `json:"children,omitempty"`
}

// FileActionType represents available file operations
type FileActionType string

const (
	FileActionView   FileActionType = "view"
	FileActionEdit   FileActionType = "edit"
	FileActionDelete FileActionType = "delete"
	FileActionDiff   FileActionType = "diff"
)

// registerFileCommands registers file-related slash commands
func (p *Plugin) registerFileCommands() error {
	// Register /claude-files command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          "claude-files",
		AutoComplete:     true,
		AutoCompleteDesc: "Browse project files",
		DisplayName:      "Browse Files",
		Description:      "Open file browser for the current Claude session",
	}); err != nil {
		return err
	}

	// Register /claude-new-file command
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          "claude-new-file",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a new file",
		AutoCompleteHint: "[file-path]",
		DisplayName:      "Create New File",
		Description:      "Create a new file in the project",
	}); err != nil {
		return err
	}

	return nil
}

// executeClaudeFiles handles the /claude-files command
func (p *Plugin) executeClaudeFiles(args *model.CommandArgs) *model.CommandResponse {
	// Get active session
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active session. Use `/claude-start [project-path]` to begin.")
	}

	// Show file browser via interactive message
	if err := p.showFileBrowser(args.ChannelId, args.UserId, session.SessionID); err != nil {
		p.API.LogError("Failed to show file browser", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to open file browser: %s", err.Error()))
	}

	return &model.CommandResponse{}
}

// executeClaudeNewFile handles the /claude-new-file command
func (p *Plugin) executeClaudeNewFile(args *model.CommandArgs, filePath string) *model.CommandResponse {
	// Get active session
	session, err := p.GetActiveSession(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get active session", "error", err.Error())
		return respondEphemeral("❌ Error retrieving session. Please try again.")
	}

	if session == nil {
		return respondEphemeral("No active session. Use `/claude-start [project-path]` to begin.")
	}

	// If path provided, create file directly; otherwise show dialog
	if filePath != "" {
		return p.createFileDirectly(args.ChannelId, session.SessionID, filePath)
	}

	// TODO: Show create file dialog (requires trigger_id from interactive action)
	return respondEphemeral("Please provide a file path. Usage: `/claude-new-file <path>`\n\nExample: `/claude-new-file src/components/NewComponent.tsx`")
}

// showFileBrowser displays an interactive file browser
func (p *Plugin) showFileBrowser(channelID, userID, sessionID string) error {
	// Fetch file list from bridge server
	files, err := p.bridgeClient.ListFiles(sessionID)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Build file tree representation (flattened for display)
	fileList := flattenFileTree(files, "")

	if len(fileList) == 0 {
		p.postBotMessage(channelID, "📂 No files found in project")
		return nil
	}

	// Create interactive message with file actions
	message := "📂 **Project Files**\n\nClick a file to view options:"

	var actions []*model.PostAction
	for i, file := range fileList {
		if i >= 20 { // Limit to 20 files to avoid overwhelming the UI
			break
		}

		icon := "📄"
		if file.Type == "directory" {
			icon = "📁"
		}

		actions = append(actions, &model.PostAction{
			Name: fmt.Sprintf("%s %s", icon, file.Path),
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/api/file-action", p.getPluginURL()),
				Context: map[string]interface{}{
					"session_id": sessionID,
					"file_path":  file.Path,
					"file_type":  file.Type,
				},
			},
		})
	}

	attachment := &model.SlackAttachment{
		Title:   "Project Files",
		Text:    message,
		Actions: actions,
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// flattenFileTree converts nested file tree to flat list for display
func flattenFileTree(nodes []FileNode, prefix string) []FileNode {
	var result []FileNode
	for _, node := range nodes {
		displayPath := node.Path
		if prefix != "" {
			displayPath = filepath.Join(prefix, node.Name)
		}

		flatNode := FileNode{
			Name: node.Name,
			Path: displayPath,
			Type: node.Type,
			Size: node.Size,
		}
		result = append(result, flatNode)

		if node.Type == "directory" && len(node.Children) > 0 {
			result = append(result, flattenFileTree(node.Children, displayPath)...)
		}
	}
	return result
}

// handleFileAction processes file action button clicks
func (p *Plugin) handleFileAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Context   map[string]interface{} `json:"context"`
		UserID    string                 `json:"user_id"`
		ChannelID string                 `json:"channel_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.API.LogError("Failed to decode file action request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionID := request.Context["session_id"].(string)
	filePath := request.Context["file_path"].(string)
	fileType := request.Context["file_type"].(string)

	// Handle directory vs file
	if fileType == "directory" {
		// TODO: Navigate into directory
		p.postBotMessage(request.ChannelID, fmt.Sprintf("📁 Directory navigation not yet implemented for: `%s`", filePath))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Show file action menu
	if err := p.showFileActionMenu(request.ChannelID, sessionID, filePath); err != nil {
		p.API.LogError("Failed to show file action menu", "error", err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// showFileActionMenu displays available actions for a file
func (p *Plugin) showFileActionMenu(channelID, sessionID, filePath string) error {
	message := fmt.Sprintf("**File:** `%s`\n\nChoose an action:", filePath)

	actions := []*model.PostAction{
		{
			Name: "👁️ View",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/api/file-view", p.getPluginURL()),
				Context: map[string]interface{}{
					"session_id": sessionID,
					"file_path":  filePath,
				},
			},
		},
		{
			Name: "✏️ Edit",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/api/file-edit", p.getPluginURL()),
				Context: map[string]interface{}{
					"session_id": sessionID,
					"file_path":  filePath,
				},
			},
		},
		{
			Name: "🗑️ Delete",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/api/file-delete", p.getPluginURL()),
				Context: map[string]interface{}{
					"session_id": sessionID,
					"file_path":  filePath,
				},
			},
			Style: "danger",
		},
	}

	attachment := &model.SlackAttachment{
		Title:   "File Actions",
		Text:    message,
		Actions: actions,
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// handleFileView displays file content
func (p *Plugin) handleFileView(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Context   map[string]interface{} `json:"context"`
		UserID    string                 `json:"user_id"`
		ChannelID string                 `json:"channel_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.API.LogError("Failed to decode file view request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionID := request.Context["session_id"].(string)
	filePath := request.Context["file_path"].(string)

	if err := p.viewFileContent(request.ChannelID, sessionID, filePath); err != nil {
		p.API.LogError("Failed to view file", "error", err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// viewFileContent fetches and displays file content
func (p *Plugin) viewFileContent(channelID, sessionID, filePath string) error {
	content, err := p.bridgeClient.GetFileContent(sessionID, filePath)
	if err != nil {
		return fmt.Errorf("failed to get file content: %w", err)
	}

	// Detect language for syntax highlighting
	ext := filepath.Ext(filePath)
	lang := getLanguageFromExtension(ext)

	// Truncate if too long (Mattermost message limit)
	const maxLength = 3500
	displayContent := content
	truncated := false
	if len(content) > maxLength {
		displayContent = content[:maxLength]
		truncated = true
	}

	codeBlock := fmt.Sprintf("```%s\n%s\n```", lang, displayContent)
	if truncated {
		codeBlock += "\n\n_...content truncated (file too large)_"
	}

	message := fmt.Sprintf("📄 **File:** `%s`\n\n%s", filePath, codeBlock)

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   message,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// handleFileEdit initiates file editing
func (p *Plugin) handleFileEdit(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Context   map[string]interface{} `json:"context"`
		UserID    string                 `json:"user_id"`
		ChannelID string                 `json:"channel_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.API.LogError("Failed to decode file edit request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	filePath := request.Context["file_path"].(string)

	// For now, tell user to edit via Claude or manually
	// TODO: Implement interactive dialog editing in Issue #5
	message := fmt.Sprintf("✏️ **Edit:** `%s`\n\nTo edit this file:\n1. Use `/claude` to ask Claude to make changes\n2. Or edit locally and changes will sync automatically", filePath)

	p.postBotMessage(request.ChannelID, message)
	w.WriteHeader(http.StatusOK)
}

// handleFileDelete processes file deletion
func (p *Plugin) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Context   map[string]interface{} `json:"context"`
		UserID    string                 `json:"user_id"`
		ChannelID string                 `json:"channel_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.API.LogError("Failed to decode file delete request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionID := request.Context["session_id"].(string)
	filePath := request.Context["file_path"].(string)

	if err := p.deleteFile(request.ChannelID, sessionID, filePath); err != nil {
		p.API.LogError("Failed to delete file", "error", err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// deleteFile removes a file from the project
func (p *Plugin) deleteFile(channelID, sessionID, filePath string) error {
	if err := p.bridgeClient.DeleteFile(sessionID, filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	message := fmt.Sprintf("🗑️ Deleted file: `%s`", filePath)
	p.postBotMessage(channelID, message)

	return nil
}

// createFileDirectly creates a new file with optional content
func (p *Plugin) createFileDirectly(channelID, sessionID, filePath string) *model.CommandResponse {
	if err := p.bridgeClient.CreateFile(sessionID, filePath, ""); err != nil {
		p.API.LogError("Failed to create file", "error", err.Error())
		return respondEphemeral(fmt.Sprintf("❌ Failed to create file: %s", err.Error()))
	}

	message := fmt.Sprintf("✅ Created file: `%s`\n\nYou can now edit it with `/claude` or your local editor", filePath)
	p.postBotMessage(channelID, message)

	return &model.CommandResponse{}
}

// getLanguageFromExtension maps file extensions to syntax highlighting languages
func getLanguageFromExtension(ext string) string {
	langMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "jsx",
		".tsx":   "tsx",
		".py":    "python",
		".rb":    "ruby",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cs":    "csharp",
		".php":   "php",
		".rs":    "rust",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".sh":    "bash",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".md":    "markdown",
		".sql":   "sql",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return ""
}

// registerFileHTTPHandlers registers HTTP handlers for file operations
func (p *Plugin) registerFileHTTPHandlers() {
	// These will be called from ServeHTTP
	// Routes:
	// - POST /api/file-action
	// - POST /api/file-view
	// - POST /api/file-edit
	// - POST /api/file-delete
}
