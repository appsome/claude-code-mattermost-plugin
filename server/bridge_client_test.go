package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestNewBridgeClient(t *testing.T) {
	api := &plugintest.API{}
	client := NewBridgeClient("http://localhost:3002", api)
	
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:3002", client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestCreateSession_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Verify request body
		var reqBody CreateSessionRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "/test/project", reqBody.ProjectPath)
		assert.Equal(t, "user123", reqBody.MattermostUserID)
		assert.Equal(t, "channel123", reqBody.MattermostChannelID)
		
		// Send response
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"session": map[string]interface{}{
				"id":                  "session123",
				"projectPath":         "/test/project",
				"mattermostUserId":    "user123",
				"mattermostChannelId": "channel123",
				"status":              "active",
				"createdAt":           1234567890,
				"updatedAt":           1234567890,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	session, err := client.CreateSession("/test/project", "user123", "channel123")
	
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "session123", session.ID)
	assert.Equal(t, "/test/project", session.ProjectPath)
	assert.Equal(t, "user123", session.MattermostUserID)
	assert.Equal(t, "channel123", session.MattermostChannelID)
	assert.Equal(t, "active", session.Status)
}

func TestCreateSession_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid project path"))
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	session, err := client.CreateSession("", "user123", "channel123")
	
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "400")
}

func TestSendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/message", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		
		var reqBody SendMessageRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "Hello Claude", reqBody.Message)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.SendMessage("session123", "Hello Claude")
	
	assert.NoError(t, err)
}

func TestSendMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Session not found"))
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.SendMessage("invalid", "test")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGetMessages_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/messages", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"id":        1,
					"sessionId": "session123",
					"role":      "user",
					"content":   "Hello",
					"timestamp": 1234567890,
				},
				{
					"id":        2,
					"sessionId": "session123",
					"role":      "assistant",
					"content":   "Hi there!",
					"timestamp": 1234567900,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	messages, err := client.GetMessages("session123", 10)
	
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "Hi there!", messages[1].Content)
}

func TestGetSession_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123", r.URL.Path)
		
		pid := 12345
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"session": map[string]interface{}{
				"id":                  "session123",
				"projectPath":         "/test/project",
				"mattermostUserId":    "user123",
				"mattermostChannelId": "channel123",
				"cliPid":              pid,
				"status":              "active",
				"createdAt":           1234567890,
				"updatedAt":           1234567900,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	session, err := client.GetSession("session123")
	
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "session123", session.ID)
	assert.Equal(t, "active", session.Status)
	assert.NotNil(t, session.CLIPid)
	assert.Equal(t, 12345, *session.CLIPid)
}

func TestDeleteSession_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.DeleteSession("session123")
	
	assert.NoError(t, err)
}

func TestSendContext_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/context", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		
		var reqBody ContextRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "thread", reqBody.Source)
		assert.Equal(t, "Thread context content", reqBody.Content)
		assert.Equal(t, "summarize", reqBody.Action)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	contextReq := &ContextRequest{
		Source:  "thread",
		Content: "Thread context content",
		Action:  "summarize",
		Metadata: &ContextMetadata{
			ChannelName:  "test-channel",
			MessageCount: 5,
		},
	}
	
	err := client.SendContext("session123", contextReq)
	
	assert.NoError(t, err)
}

func TestApproveChange_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/approve", r.URL.Path)
		
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "change456", reqBody["changeId"])
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.ApproveChange("session123", "change456")
	
	assert.NoError(t, err)
}

func TestRejectChange_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/reject", r.URL.Path)
		
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "change456", reqBody["changeId"])
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.RejectChange("session123", "change456")
	
	assert.NoError(t, err)
}

func TestModifyChange_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/modify", r.URL.Path)
		
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "change456", reqBody["changeId"])
		assert.Equal(t, "Add more tests", reqBody["instructions"])
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "modified"})
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	err := client.ModifyChange("session123", "change456", "Add more tests")
	
	assert.NoError(t, err)
}

func TestGetFileContent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sessions/session123/file", r.URL.Path)
		
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "src/main.go", reqBody["filename"])
		
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"content": "package main\n\nfunc main() {}\n",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	api := &plugintest.API{}
	client := NewBridgeClient(server.URL, api)
	
	content, err := client.GetFileContent("session123", "src/main.go")
	
	assert.NoError(t, err)
	assert.Contains(t, content, "package main")
	assert.Contains(t, content, "func main")
}
