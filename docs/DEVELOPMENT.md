# Development Guide

## Overview

The Claude Code Mattermost Plugin integrates Claude Code AI assistant directly into Mattermost, providing a native chat-based development experience.

## Architecture

The project consists of three main components:

1. **Mattermost Plugin** (Go backend + React frontend)
   - Handles slash commands
   - Manages interactive messages and dialogs
   - Communicates with bridge server via REST and WebSocket

2. **Bridge Server** (Node.js)
   - Manages Claude Code CLI sessions
   - Provides REST API for session management
   - WebSocket server for real-time updates
   - SQLite database for session persistence

3. **Claude Code CLI**
   - Actual AI assistant doing the work
   - Spawned and controlled by bridge server

## Development Setup

### Prerequisites

- **Go 1.21+** - Backend development
- **Node.js 22+** - Frontend and bridge server
- **Docker & Docker Compose** - Development environment
- **Make** - Build automation
- **Claude Code CLI** - AI assistant (optional for testing)

### Clone and Install

```bash
# Clone repository
git clone https://github.com/appsome/claude-code-mattermost-plugin.git
cd claude-code-mattermost-plugin

# Install Go dependencies (automatic via go mod)
cd server && go mod download && cd ..

# Install frontend dependencies
cd webapp && npm install && cd ..

# Install bridge server dependencies
cd bridge-server && npm install && cd ..
```

### Development Environment

Start Mattermost and dependencies with Docker Compose:

```bash
make dev
```

This starts:
- Mattermost at http://localhost:8065
- PostgreSQL database
- Default credentials: `admin@example.com` / `admin123`

Stop the environment:

```bash
make dev-down
```

### Building the Plugin

```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Create plugin bundle (.tar.gz)
make bundle

# Create all platform bundles
make bundle-all
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Backend tests only
cd server && go test -v ./...

# Frontend tests only
cd webapp && npm test

# Bridge server tests
cd bridge-server && npm test
```

### Code Style

```bash
# Format code
make fmt

# Lint code
make lint

# Check style (format + lint)
make check-style
```

## Project Structure

```
claude-code-mattermost-plugin/
├── server/                 # Go backend
│   ├── plugin.go          # Main plugin entry point
│   ├── commands.go        # Slash command handlers
│   ├── session_manager.go # Session lifecycle management
│   ├── bridge_client.go   # Bridge server REST client
│   ├── websocket_client.go # WebSocket connection
│   ├── thread_context.go  # Thread context integration
│   ├── file_operations.go # File browsing and operations
│   └── configuration.go   # Plugin configuration
├── webapp/                # React frontend
│   ├── src/
│   │   ├── index.tsx     # Plugin entry point
│   │   └── components/   # React components
│   └── package.json
├── bridge-server/        # Node.js bridge server
│   ├── src/
│   │   ├── index.ts      # Server entry point
│   │   ├── session-manager.ts
│   │   ├── cli-spawner.ts
│   │   ├── websocket-server.ts
│   │   └── api/          # REST API endpoints
│   └── package.json
├── plugin.json           # Plugin manifest
├── Makefile             # Build automation
├── docker-compose.yml   # Development environment
└── docs/                # Documentation
```

## Plugin Development

### Adding a New Slash Command

1. **Register command** in `server/plugin.go`:

```go
func (p *Plugin) OnActivate() error {
    if err := p.API.RegisterCommand(&model.Command{
        Trigger:          "my-command",
        AutoComplete:     true,
        AutoCompleteDesc: "Description",
        DisplayName:      "My Command",
    }); err != nil {
        return err
    }
    return nil
}
```

2. **Handle command** in `server/commands.go`:

```go
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
    trigger := strings.TrimPrefix(args.Command, "/")
    
    switch trigger {
    case "my-command":
        return p.executeMyCommand(args), nil
    default:
        return respondEphemeral("Unknown command"), nil
    }
}
```

### Adding Interactive Actions

1. **Create action buttons**:

```go
actions := []*model.PostAction{
    {
        Name: "✅ Approve",
        Integration: &model.PostActionIntegration{
            URL: fmt.Sprintf("%s/api/approve", p.getPluginURL()),
            Context: map[string]interface{}{
                "change_id": changeID,
            },
        },
        Style: "primary",
    },
}
```

2. **Handle action** in `ServeHTTP`:

```go
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/api/approve" {
        p.handleApprove(w, r)
        return
    }
}
```

## Bridge Server Development

### Adding API Endpoints

Edit `bridge-server/src/api/index.ts`:

```typescript
router.post('/sessions/:id/action', async (req, res) => {
    const { id } = req.params;
    const session = sessionManager.getSession(id);
    
    if (!session) {
        return res.status(404).json({ error: 'Session not found' });
    }
    
    // Handle action
    res.json({ success: true });
});
```

### WebSocket Messages

Server → Plugin:

```typescript
websocketServer.broadcast(sessionId, {
    type: 'message',
    content: 'Response from Claude',
    timestamp: Date.now(),
});
```

Plugin → Server:

```go
p.wsClient.Send(websocket.Message{
    Type:      "prompt",
    SessionID: sessionID,
    Content:   userMessage,
})
```

## Testing Guidelines

### Unit Tests (Go)

```go
func TestExecuteCommand(t *testing.T) {
    plugin := setupTestPlugin(t)
    
    resp, err := plugin.ExecuteCommand(nil, &model.CommandArgs{
        Command: "/claude-help",
    })
    
    assert.Nil(t, err)
    assert.Contains(t, resp.Text, "Claude Code commands")
}
```

### Integration Tests

```go
func TestSessionLifecycle(t *testing.T) {
    plugin := setupTestPlugin(t)
    
    // Start session
    resp := plugin.executeClaudeStart(testArgs, "/tmp/project")
    assert.NotNil(t, resp)
    
    // Verify active
    session := plugin.GetActiveSession(testChannelID)
    assert.NotNil(t, session)
    
    // Stop session
    resp = plugin.executeClaudeStop(testArgs)
    assert.Contains(t, resp.Text, "stopped")
}
```

## Debugging

### Plugin Logs

View Mattermost logs:

```bash
docker-compose logs -f mattermost
```

Or from plugin code:

```go
p.API.LogError("Error message", "key", value)
p.API.LogInfo("Info message")
p.API.LogDebug("Debug message")
```

### Bridge Server Logs

```bash
cd bridge-server
npm run dev  # Starts with debug logging
```

### Browser DevTools

For React frontend debugging, use browser DevTools with source maps enabled.

## Common Issues

### Plugin Not Loading

- Check Mattermost logs for errors
- Verify plugin.json has correct structure
- Ensure Go module dependencies are downloaded

### Bridge Server Connection Failed

- Verify bridge server is running
- Check plugin configuration for correct URL
- Test bridge health endpoint: `curl http://localhost:3002/health`

### Commands Not Responding

- Check slash command registration in logs
- Verify bot user was created
- Check channel permissions

## Contributing

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes with tests
3. Run `make check-style` and `make test`
4. Commit with clear message
5. Push and create pull request
6. Assign reviewer: @suda

## Useful Commands

```bash
# Watch mode for frontend development
cd webapp && npm run dev

# Bridge server with auto-reload
cd bridge-server && npm run dev

# Run specific test
cd server && go test -v -run TestExecuteCommand

# Build plugin and upload to Mattermost
make bundle && curl -F 'plugin=@dist/com.appsome.claudecode.tar.gz' \
  http://localhost:8065/api/v4/plugins \
  -H 'Authorization: Bearer YOUR_TOKEN'
```

## Resources

- [Mattermost Plugin Documentation](https://developers.mattermost.com/integrate/plugins/)
- [Mattermost Plugin API Reference](https://developers.mattermost.com/integrate/plugins/server/reference/)
- [Claude Code Documentation](https://github.com/siteboon/claudecode)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [React Testing Library](https://testing-library.com/react)
