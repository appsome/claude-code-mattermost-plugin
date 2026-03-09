package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPlugin() *Plugin {
	p := &Plugin{
		botUserID:    "bot_id",
		bridgeClient: NewBridgeClient("http://localhost:3001", nil),
		configuration: &configuration{
			BridgeServerURL: "http://localhost:3001",
		},
	}
	return p
}

func setupKVMocks(api *plugintest.API) {
	sessionData, _ := json.Marshal(&ChannelSession{SessionID: "session_123", UserID: "user_id"})
	api.On("KVGet", mock.Anything).Return(sessionData, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("KVDelete", mock.Anything).Return(nil).Maybe()
}

func TestServeHTTP_Routes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"not_found", "/api/unknown", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			defer api.AssertExpectations(t)

			p := setupPlugin()
			p.SetAPI(api)

			req := httptest.NewRequest("POST", tt.path, nil)
			w := httptest.NewRecorder()

			p.ServeHTTP(&plugin.Context{}, w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHandleApprove_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session_123/approve", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	api.On("GetUser", "user_id").Return(&model.User{
		Id:       "user_id",
		Username: "testuser",
	}, nil)

	reqBody := model.PostActionIntegrationRequest{
		UserId:    "user_id",
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	// Save session for channel
	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	req := httptest.NewRequest("POST", "/api/action/approve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleApprove(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.PostActionIntegrationResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Contains(t, response.Update.Message, "Changes approved")
}

func TestHandleApprove_InvalidRequest(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	req := httptest.NewRequest("POST", "/api/action/approve", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	p.handleApprove(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleApprove_MissingChangeID(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.PostActionIntegrationRequest{
		Context: map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/approve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleApprove(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleApprove_NoActiveSession(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	// Mock no active session (KVGet returns nil)
	api.On("KVGet", mock.Anything).Return(nil, nil)
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	reqBody := model.PostActionIntegrationRequest{
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/approve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleApprove(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleReject_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	api.On("GetUser", "user_id").Return(&model.User{
		Id:       "user_id",
		Username: "testuser",
	}, nil)

	reqBody := model.PostActionIntegrationRequest{
		UserId:    "user_id",
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	req := httptest.NewRequest("POST", "/api/action/reject", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleReject(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.PostActionIntegrationResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Contains(t, response.Update.Message, "Changes rejected")
}

func TestHandleModify_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("OpenInteractiveDialog", mock.Anything).Return(nil)
	api.On("GetConfig", mock.Anything).Return(&model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: model.NewString("http://localhost:8065"),
		},
	})

	reqBody := model.PostActionIntegrationRequest{
		TriggerId: "trigger_123",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/modify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleModify(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleContinue_Success(t *testing.T) {
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

	reqBody := model.PostActionIntegrationRequest{
		Context: map[string]interface{}{
			"session_id": "session_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/continue", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleContinue(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleExplain_Success(t *testing.T) {
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

	reqBody := model.PostActionIntegrationRequest{
		Context: map[string]interface{}{
			"session_id": "session_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/explain", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleExplain(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleUndo_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("OpenInteractiveDialog", mock.Anything).Return(nil)
	api.On("GetConfig", mock.Anything).Return(&model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: model.NewString("http://localhost:8065"),
		},
	})

	reqBody := model.PostActionIntegrationRequest{
		TriggerId: "trigger_123",
		Context: map[string]interface{}{
			"session_id": "session_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/undo", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleUndo(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleApply_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	reqBody := model.PostActionIntegrationRequest{
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	req := httptest.NewRequest("POST", "/api/action/apply", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleApply(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleDiscard_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	reqBody := model.PostActionIntegrationRequest{
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"change_id": "change_123",
		},
	}
	body, _ := json.Marshal(reqBody)

	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	req := httptest.NewRequest("POST", "/api/action/discard", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleDiscard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleView_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)

	// Setup mock bridge server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"content": "file content",
		})
	}))
	defer mockServer.Close()
	p.bridgeClient.baseURL = mockServer.URL

	api.On("SendEphemeralPost", "user_id", mock.Anything).Return(nil)

	reqBody := model.PostActionIntegrationRequest{
		UserId:    "user_id",
		ChannelId: "channel_id",
		Context: map[string]interface{}{
			"filename": "test.go",
		},
	}
	body, _ := json.Marshal(reqBody)

	p.SaveSession("channel_id", &ChannelSession{
		SessionID: "session_123",
		UserID:    "user_id",
	})

	req := httptest.NewRequest("POST", "/api/action/view", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleView(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleMenu_Success(t *testing.T) {
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

	reqBody := model.PostActionIntegrationRequest{
		Context: map[string]interface{}{
			"session_id":      "session_123",
			"selected_option": "test option",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/action/menu", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.handleMenu(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWriteError(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	w := httptest.NewRecorder()
	p.writeError(w, assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response model.PostActionIntegrationResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Contains(t, response.EphemeralText, "Error:")
}
