package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mattermost/mattermost/server/public/model"
)

// WebSocketClient handles WebSocket communication with the Claude Code bridge server
type WebSocketClient struct {
	baseURL       string
	plugin        *Plugin
	conn          *websocket.Conn
	mu            sync.RWMutex
	subscriptions map[string]string // sessionID -> channelID
	stopChan      chan struct{}
	reconnecting  bool
}

// WebSocketMessage represents a message from the bridge WebSocket
type WebSocketMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

// SubscribeMessage is sent to subscribe to a session
type SubscribeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(baseURL string, plugin *Plugin) *WebSocketClient {
	return &WebSocketClient{
		baseURL:       baseURL,
		plugin:        plugin,
		subscriptions: make(map[string]string),
		stopChan:      make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the bridge server
func (ws *WebSocketClient) Connect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Convert http:// to ws:// or https:// to wss://
	wsURL := ws.baseURL
	if len(wsURL) > 7 && wsURL[:7] == "http://" {
		wsURL = "ws://" + wsURL[7:]
	} else if len(wsURL) > 8 && wsURL[:8] == "https://" {
		wsURL = "wss://" + wsURL[8:]
	}
	wsURL += "/ws"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.plugin.API.LogInfo("WebSocket connected to bridge server", "url", wsURL)

	// Start message handler
	go ws.handleMessages()

	// Start ping handler
	go ws.pingHandler()

	return nil
}

// handleMessages processes incoming WebSocket messages
func (ws *WebSocketClient) handleMessages() {
	for {
		select {
		case <-ws.stopChan:
			return
		default:
			ws.mu.RLock()
			conn := ws.conn
			ws.mu.RUnlock()

			if conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				ws.plugin.API.LogError("WebSocket read error", "error", err.Error())
				
				// Try to reconnect
				if !ws.reconnecting {
					go ws.reconnect()
				}
				return
			}

			// Process message
			ws.processMessage(&msg)
		}
	}
}

// processMessage handles different message types
func (ws *WebSocketClient) processMessage(msg *WebSocketMessage) {
	ws.mu.RLock()
	channelID, ok := ws.subscriptions[msg.SessionID]
	ws.mu.RUnlock()

	if !ok {
		// Not subscribed to this session
		return
	}

	switch msg.Type {
	case "output":
		ws.handleOutput(channelID, msg)
	case "error":
		ws.handleError(channelID, msg)
	case "status":
		ws.handleStatus(channelID, msg)
	case "file_change":
		ws.handleFileChange(channelID, msg)
	default:
		ws.plugin.API.LogDebug("Unknown message type", "type", msg.Type)
	}
}

// handleOutput processes CLI output messages
func (ws *WebSocketClient) handleOutput(channelID string, msg *WebSocketMessage) {
	var data struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		ws.plugin.API.LogError("Failed to unmarshal output data", "error", err.Error())
		return
	}

	// Post message as bot
	ws.plugin.postBotMessage(channelID, data.Output)
}

// handleError processes error messages
func (ws *WebSocketClient) handleError(channelID string, msg *WebSocketMessage) {
	var data struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		ws.plugin.API.LogError("Failed to unmarshal error data", "error", err.Error())
		return
	}

	// Post error as bot
	errorMsg := fmt.Sprintf("⚠️ Error: %s", data.Error)
	ws.plugin.postBotMessage(channelID, errorMsg)
}

// handleStatus processes status update messages
func (ws *WebSocketClient) handleStatus(channelID string, msg *WebSocketMessage) {
	var data struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		ExitCode *int   `json:"exitCode"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		ws.plugin.API.LogError("Failed to unmarshal status data", "error", err.Error())
		return
	}

	// Handle stopped status
	if data.Status == "stopped" {
		ws.plugin.postBotMessage(channelID, "⏹️ Claude Code session stopped.")
		
		// Clean up local session
		if err := ws.plugin.DeleteSession(channelID); err != nil {
			ws.plugin.API.LogWarn("Failed to delete session after stop", "error", err.Error())
		}
		
		// Unsubscribe
		ws.Unsubscribe(msg.SessionID)
	} else if data.Message != "" {
		ws.plugin.postBotMessage(channelID, data.Message)
	}
}

// handleFileChange processes file change notifications
func (ws *WebSocketClient) handleFileChange(channelID string, msg *WebSocketMessage) {
	var data struct {
		Path   string `json:"path"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		ws.plugin.API.LogError("Failed to unmarshal file change data", "error", err.Error())
		return
	}

	// Post notification
	notification := fmt.Sprintf("📝 File %s: `%s`", data.Action, data.Path)
	ws.plugin.postBotMessage(channelID, notification)
}

// Subscribe adds a session subscription
func (ws *WebSocketClient) Subscribe(sessionID, channelID string) {
	ws.mu.Lock()
	ws.subscriptions[sessionID] = channelID
	ws.mu.Unlock()

	// Send subscribe message to bridge
	if ws.conn != nil {
		msg := SubscribeMessage{
			Type:      "subscribe",
			SessionID: sessionID,
		}
		if err := ws.conn.WriteJSON(msg); err != nil {
			ws.plugin.API.LogError("Failed to send subscribe message", "error", err.Error())
		}
	}
}

// Unsubscribe removes a session subscription
func (ws *WebSocketClient) Unsubscribe(sessionID string) {
	ws.mu.Lock()
	delete(ws.subscriptions, sessionID)
	ws.mu.Unlock()

	// Send unsubscribe message to bridge
	if ws.conn != nil {
		msg := map[string]string{
			"type":      "unsubscribe",
			"sessionId": sessionID,
		}
		if err := ws.conn.WriteJSON(msg); err != nil {
			ws.plugin.API.LogError("Failed to send unsubscribe message", "error", err.Error())
		}
	}
}

// pingHandler sends periodic pings to keep connection alive
func (ws *WebSocketClient) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ws.stopChan:
			return
		case <-ticker.C:
			ws.mu.RLock()
			conn := ws.conn
			ws.mu.RUnlock()

			if conn != nil {
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					ws.plugin.API.LogError("Failed to send ping", "error", err.Error())
				}
			}
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WebSocketClient) reconnect() {
	ws.mu.Lock()
	ws.reconnecting = true
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		ws.reconnecting = false
		ws.mu.Unlock()
	}()

	// Close existing connection if any
	ws.mu.Lock()
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.mu.Unlock()

	// Exponential backoff
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ws.stopChan:
			return
		default:
			ws.plugin.API.LogInfo("Attempting to reconnect WebSocket...")

			if err := ws.Connect(); err != nil {
				ws.plugin.API.LogError("Reconnect failed", "error", err.Error())
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			// Reconnected successfully, resubscribe to all sessions
			ws.mu.RLock()
			subscriptions := make(map[string]string)
			for sessionID, channelID := range ws.subscriptions {
				subscriptions[sessionID] = channelID
			}
			ws.mu.RUnlock()

			for sessionID := range subscriptions {
				ws.Subscribe(sessionID, subscriptions[sessionID])
			}

			ws.plugin.API.LogInfo("WebSocket reconnected successfully")
			return
		}
	}
}

// Close closes the WebSocket connection
func (ws *WebSocketClient) Close() error {
	close(ws.stopChan)

	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.conn != nil {
		err := ws.conn.Close()
		ws.conn = nil
		return err
	}

	return nil
}

// postBotMessage posts a message from the bot to a channel
func (p *Plugin) postBotMessage(channelID, message string) {
	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   message,
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogError("Failed to create bot post", "error", appErr.Error())
	}
}
