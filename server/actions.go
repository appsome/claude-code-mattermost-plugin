package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// sendToSession sends a message to a session (bridge or embedded mode)
func (p *Plugin) sendToSession(sessionID, message string) error {
	if p.UseBridgeMode() {
		return p.bridgeClient.SendMessage(sessionID, message)
	}
	return p.processManager.SendInput(sessionID, message)
}

// ServeHTTP handles HTTP requests for the plugin (actions and dialogs)
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/action/approve":
		p.handleApprove(w, r)
	case "/api/action/reject":
		p.handleReject(w, r)
	case "/api/action/modify":
		p.handleModify(w, r)
	case "/api/action/continue":
		p.handleContinue(w, r)
	case "/api/action/explain":
		p.handleExplain(w, r)
	case "/api/action/undo":
		p.handleUndo(w, r)
	case "/api/action/apply":
		p.handleApply(w, r)
	case "/api/action/discard":
		p.handleDiscard(w, r)
	case "/api/action/view":
		p.handleView(w, r)
	case "/api/action/menu":
		p.handleMenu(w, r)
	case "/api/dialog/modify-change":
		p.handleModifyDialog(w, r)
	case "/api/dialog/confirm":
		p.handleConfirmDialog(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleApprove handles the approve button action
func (p *Plugin) handleApprove(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	changeID, ok := request.Context["change_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing change_id"))
		return
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeError(w, fmt.Errorf("no active session"))
		return
	}

	// Send approval
	if p.UseBridgeMode() {
		if err := p.bridgeClient.ApproveChange(sessionID, changeID); err != nil {
			p.writeError(w, err)
			return
		}
	} else {
		approveMsg := fmt.Sprintf("approve %s", changeID)
		if err := p.processManager.SendInput(sessionID, approveMsg); err != nil {
			p.writeError(w, err)
			return
		}
	}

	user, _ := p.API.GetUser(request.UserId)
	username := request.UserId
	if user != nil {
		username = user.Username
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: fmt.Sprintf("Changes approved by @%s", username),
			Props: model.StringInterface{
				"from_bot": "true",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleReject handles the reject button action
func (p *Plugin) handleReject(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	changeID, ok := request.Context["change_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing change_id"))
		return
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeError(w, fmt.Errorf("no active session"))
		return
	}

	// Send rejection
	if p.UseBridgeMode() {
		if err := p.bridgeClient.RejectChange(sessionID, changeID); err != nil {
			p.writeError(w, err)
			return
		}
	} else {
		rejectMsg := fmt.Sprintf("reject %s", changeID)
		if err := p.processManager.SendInput(sessionID, rejectMsg); err != nil {
			p.writeError(w, err)
			return
		}
	}

	user, _ := p.API.GetUser(request.UserId)
	username := request.UserId
	if user != nil {
		username = user.Username
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: fmt.Sprintf("Changes rejected by @%s", username),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModify handles the modify button action (opens dialog)
func (p *Plugin) handleModify(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	changeID, ok := request.Context["change_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing change_id"))
		return
	}

	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       p.getPluginURL() + "/api/dialog/modify-change",
		Dialog: model.Dialog{
			Title: "Modify Request",
			Elements: []model.DialogElement{
				{
					DisplayName: "Modification Instructions",
					Name:        "instructions",
					Type:        "textarea",
					Placeholder: "Tell Claude Code how to modify the changes...",
				},
				{
					DisplayName: "change_id",
					Name:        "change_id",
					Type:        "text",
					Default:     changeID,
					Optional:    false,
				},
			},
			SubmitLabel: "Send",
		},
	}

	if err := p.API.OpenInteractiveDialog(dialog); err != nil {
		p.writeError(w, err)
		return
	}

	response := &model.PostActionIntegrationResponse{}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleContinue handles the continue quick action
func (p *Plugin) handleContinue(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	sessionID, ok := request.Context["session_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing session_id"))
		return
	}

	if err := p.sendToSession(sessionID, "continue"); err != nil {
		p.writeError(w, err)
		return
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: "Continuing...",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleExplain handles the explain quick action
func (p *Plugin) handleExplain(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	sessionID, ok := request.Context["session_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing session_id"))
		return
	}

	if err := p.sendToSession(sessionID, "explain that"); err != nil {
		p.writeError(w, err)
		return
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: "Explaining...",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUndo handles the undo quick action
func (p *Plugin) handleUndo(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	sessionID, ok := request.Context["session_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing session_id"))
		return
	}

	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       p.getPluginURL() + "/api/dialog/confirm",
		Dialog: model.Dialog{
			Title:            "Confirm Undo",
			IntroductionText: "Are you sure you want to undo the last action?",
			SubmitLabel:      "Confirm",
			NotifyOnCancel:   false,
			Elements: []model.DialogElement{
				{
					DisplayName: "session_id",
					Name:        "session_id",
					Type:        "text",
					Default:     sessionID,
					Optional:    false,
				},
				{
					DisplayName: "action",
					Name:        "action",
					Type:        "text",
					Default:     "undo",
					Optional:    false,
				},
			},
		},
	}

	if err := p.API.OpenInteractiveDialog(dialog); err != nil {
		p.writeError(w, err)
		return
	}

	response := &model.PostActionIntegrationResponse{}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleApply handles the apply code change action
func (p *Plugin) handleApply(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	changeID, ok := request.Context["change_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing change_id"))
		return
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeError(w, fmt.Errorf("no active session"))
		return
	}

	// Send apply
	if p.UseBridgeMode() {
		if err := p.bridgeClient.ApproveChange(sessionID, changeID); err != nil {
			p.writeError(w, err)
			return
		}
	} else {
		applyMsg := fmt.Sprintf("apply %s", changeID)
		if err := p.processManager.SendInput(sessionID, applyMsg); err != nil {
			p.writeError(w, err)
			return
		}
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: "Changes applied",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDiscard handles the discard code change action
func (p *Plugin) handleDiscard(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	changeID, ok := request.Context["change_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing change_id"))
		return
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeError(w, fmt.Errorf("no active session"))
		return
	}

	// Send discard
	if p.UseBridgeMode() {
		if err := p.bridgeClient.RejectChange(sessionID, changeID); err != nil {
			p.writeError(w, err)
			return
		}
	} else {
		discardMsg := fmt.Sprintf("discard %s", changeID)
		if err := p.processManager.SendInput(sessionID, discardMsg); err != nil {
			p.writeError(w, err)
			return
		}
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: "Changes discarded",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleView handles the view full file action
func (p *Plugin) handleView(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	filePath, ok := request.Context["file_path"].(string)
	if !ok {
		// Try "filename" as fallback
		filePath, ok = request.Context["filename"].(string)
		if !ok {
			p.writeError(w, fmt.Errorf("missing file_path"))
			return
		}
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeError(w, fmt.Errorf("no active session"))
		return
	}

	var message string
	if p.UseBridgeMode() {
		// Bridge mode: get file content from bridge
		content, err := p.bridgeClient.GetFileContent(sessionID, filePath)
		if err != nil {
			p.writeError(w, err)
			return
		}
		message = fmt.Sprintf("**Full content of %s:**\n```\n%s\n```", filePath, content)
	} else {
		// Embedded mode: file viewing not directly supported
		message = fmt.Sprintf("File: `%s`\n\nFile viewing is available in the project directory.", filePath)
	}

	ephemeral := &model.Post{
		ChannelId: request.ChannelId,
		UserId:    p.botUserID,
		Message:   message,
	}

	p.API.SendEphemeralPost(request.UserId, ephemeral)

	response := &model.PostActionIntegrationResponse{}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMenu handles dropdown menu selections
func (p *Plugin) handleMenu(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeError(w, fmt.Errorf("invalid request: %w", err))
		return
	}

	sessionID, ok := request.Context["session_id"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing session_id"))
		return
	}

	selectedValue, ok := request.Context["selected_option"].(string)
	if !ok {
		p.writeError(w, fmt.Errorf("missing selected_option"))
		return
	}

	if err := p.sendToSession(sessionID, selectedValue); err != nil {
		p.writeError(w, err)
		return
	}

	response := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: fmt.Sprintf("Executing: %s", selectedValue),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (p *Plugin) writeError(w http.ResponseWriter, err error) {
	p.API.LogError("Action error", "error", err.Error())
	w.WriteHeader(http.StatusInternalServerError)
	response := &model.PostActionIntegrationResponse{
		EphemeralText: fmt.Sprintf("Error: %s", err.Error()),
	}
	json.NewEncoder(w).Encode(response)
}
