package main

// WebSocketClient handles WebSocket communication with the Claude Code bridge server
type WebSocketClient struct {
	baseURL string
	plugin  *Plugin
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(baseURL string, plugin *Plugin) *WebSocketClient {
	return &WebSocketClient{
		baseURL: baseURL,
		plugin:  plugin,
	}
}

// Connect establishes a WebSocket connection to the bridge server
func (ws *WebSocketClient) Connect() error {
	// TODO: Implement WebSocket connection in Issue #4
	return nil
}

// Close closes the WebSocket connection
func (ws *WebSocketClient) Close() error {
	// TODO: Implement in Issue #4
	return nil
}

// TODO: Methods will be implemented in Issue #4
// - handleMessage(msg BridgeMessage)
// - reconnect()
