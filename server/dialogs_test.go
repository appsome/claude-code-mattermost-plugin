package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleModifyDialog_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/sessions/")
		assert.Contains(t, r.URL.Path, "/modify")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	// Save session for channel
	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

	reqBody := model.SubmitDialogRequest{
		ChannelId: "channel_id",
		Submission: map[string]interface{}{
			"instructions": "Make it faster",
			"change_id":    "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Empty(t, response.Error)
}

func TestHandleModifyDialog_InvalidRequest(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Invalid request", response.Error)
}

func TestHandleModifyDialog_MissingInstructions(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Please provide modification instructions", response.Error)
}

func TestHandleModifyDialog_EmptyInstructions(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"instructions": "",
			"change_id":    "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Please provide modification instructions", response.Error)
}

func TestHandleModifyDialog_MissingChangeID(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"instructions": "Make it faster",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Missing change ID", response.Error)
}

func TestHandleModifyDialog_NoActiveSession(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	// Mock no active session (KVGet returns nil)
	api.On("KVGet", mock.Anything).Return(nil, nil)
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		ChannelId: "channel_id",
		Submission: map[string]interface{}{
			"instructions": "Make it faster",
			"change_id":    "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/modify-change", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModifyDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "No active session", response.Error)
}

func TestHandleConfirmDialog_Success_Undo(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

	reqBody := model.SubmitDialogRequest{
		ChannelId: "channel_id",
		Submission: map[string]interface{}{
			"session_id": "session_123",
			"action":     "undo",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/confirm", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleConfirmDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Empty(t, response.Error)
}

func TestHandleConfirmDialog_InvalidRequest(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	req := httptest.NewRequest("POST", "/api/dialog/confirm", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	p.handleConfirmDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Invalid request", response.Error)
}

func TestHandleConfirmDialog_MissingSessionID(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"action": "undo",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/confirm", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleConfirmDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Missing session ID", response.Error)
}

func TestHandleConfirmDialog_MissingAction(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"session_id": "session_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/confirm", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleConfirmDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Missing action", response.Error)
}

func TestHandleConfirmDialog_UnknownAction(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.SubmitDialogRequest{
		Submission: map[string]interface{}{
			"session_id": "session_123",
			"action":     "unknown_action",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dialog/confirm", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleConfirmDialog(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Contains(t, response.Error, "Unknown action")
}

func TestWriteDialogError(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	w := httptest.NewRecorder()
	p.writeDialogError(w, "Test error message")

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SubmitDialogResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Test error message", response.Error)
}
