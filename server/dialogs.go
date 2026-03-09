package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

// handleModifyDialog handles the modify change dialog submission
func (p *Plugin) handleModifyDialog(w http.ResponseWriter, r *http.Request) {
	var request model.SubmitDialogRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeDialogError(w, "Invalid request")
		return
	}

	instructions, ok := request.Submission["instructions"].(string)
	if !ok || instructions == "" {
		p.writeDialogError(w, "Please provide modification instructions")
		return
	}

	changeID, ok := request.Submission["change_id"].(string)
	if !ok || changeID == "" {
		p.writeDialogError(w, "Missing change ID")
		return
	}

	sessionID := p.GetSessionForChannel(request.ChannelId)
	if sessionID == "" {
		p.writeDialogError(w, "No active session")
		return
	}

	// Send modification request to CLI process
	modifyMsg := fmt.Sprintf("modify %s: %s", changeID, instructions)
	if err := p.processManager.SendInput(sessionID, modifyMsg); err != nil {
		p.writeDialogError(w, fmt.Sprintf("Failed to send modification: %s", err.Error()))
		return
	}

	// Post confirmation message
	post := &model.Post{
		ChannelId: request.ChannelId,
		UserId:    p.botUserID,
		Message:   fmt.Sprintf("Modification requested: %s", instructions),
	}
	p.API.CreatePost(post)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	response := &model.SubmitDialogResponse{}
	json.NewEncoder(w).Encode(response)
}

// handleConfirmDialog handles confirmation dialog submissions
func (p *Plugin) handleConfirmDialog(w http.ResponseWriter, r *http.Request) {
	var request model.SubmitDialogRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.writeDialogError(w, "Invalid request")
		return
	}

	sessionID, ok := request.Submission["session_id"].(string)
	if !ok || sessionID == "" {
		p.writeDialogError(w, "Missing session ID")
		return
	}

	action, ok := request.Submission["action"].(string)
	if !ok || action == "" {
		p.writeDialogError(w, "Missing action")
		return
	}

	// Execute the confirmed action
	switch action {
	case "undo":
		if err := p.processManager.SendInput(sessionID, "undo"); err != nil {
			p.writeDialogError(w, fmt.Sprintf("Failed to undo: %s", err.Error()))
			return
		}

		post := &model.Post{
			ChannelId: request.ChannelId,
			UserId:    p.botUserID,
			Message:   "Undoing last action...",
		}
		p.API.CreatePost(post)

	default:
		p.writeDialogError(w, fmt.Sprintf("Unknown action: %s", action))
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	response := &model.SubmitDialogResponse{}
	json.NewEncoder(w).Encode(response)
}

// writeDialogError writes a dialog validation error
func (p *Plugin) writeDialogError(w http.ResponseWriter, message string) {
	p.API.LogError("Dialog error", "message", message)
	w.Header().Set("Content-Type", "application/json")
	response := &model.SubmitDialogResponse{
		Error: message,
	}
	json.NewEncoder(w).Encode(response)
}
