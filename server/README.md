# Claude Code Mattermost Plugin - Server

Go backend for the Claude Code Mattermost plugin.

## Architecture

The server plugin provides:

1. **Slash Command Handlers** - `/claude`, `/claude-start`, `/claude-stop`, `/claude-status`, `/claude-help`
2. **Session Management** - KV store for tracking active sessions per channel
3. **Bridge Client** - HTTP client for communicating with the bridge server REST API
4. **WebSocket Client** - Real-time streaming of Claude Code CLI output
5. **Bot Integration** - Automated bot user for posting Claude responses

## Components

### plugin.go
Main plugin entry point with lifecycle hooks:
- `OnActivate()` - Initializes bot user, bridge client, WebSocket, and commands
- `OnDeactivate()` - Cleans up WebSocket connections

### commands.go
Slash command handlers:
- `/claude <message>` - Send message to active session
- `/claude-start [project-path]` - Start new Claude Code session
- `/claude-stop` - Stop current session
- `/claude-status` - Show session details and statistics
- `/claude-help` - Display help information

### session_manager.go
Session lifecycle management:
- Store/retrieve session state in Mattermost KV store
- Create sessions via bridge server
- Link sessions to channels
- Track last message timestamps

### bridge_client.go
HTTP client for bridge server API:
- `CreateSession()` - Start new CLI session
- `SendMessage()` - Send user message to CLI
- `GetMessages()` - Retrieve message history
- `GetSession()` - Get session details
- `DeleteSession()` - Stop and remove session

### websocket_client.go
Real-time WebSocket client:
- Subscribe to CLI output for specific sessions
- Auto-reconnect on disconnect
- Handle multiple message types (output, error, status, file_change)
- Post bot messages to Mattermost channels

### configuration.go
Plugin settings schema (defined in plugin.json):
- Bridge server URL
- Claude Code CLI path
- File operations toggle

## Building

```bash
# Build for current platform
cd server
go build -o ../build/server/plugin-linux-amd64 .

# Or use Make from root
cd ..
make build
```

## Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Session Flow

1. User runs `/claude-start /path/to/project`
2. Plugin creates session via bridge server
3. Bridge server spawns Claude Code CLI process
4. Plugin stores session ID in KV store (key: `session_<channelID>`)
5. WebSocket subscribes to session for real-time updates
6. User sends `/claude <message>`
7. Plugin forwards message to bridge via REST API
8. Bridge sends to CLI stdin
9. CLI output streams via WebSocket to plugin
10. Plugin posts bot messages to Mattermost channel

## Message Types (WebSocket)

### output
Claude Code CLI stdout:
```json
{
  "type": "output",
  "sessionId": "uuid",
  "data": { "output": "..." },
  "timestamp": 1234567890
}
```

### error
CLI stderr or errors:
```json
{
  "type": "error",
  "sessionId": "uuid",
  "data": { "error": "..." },
  "timestamp": 1234567890
}
```

### status
Session status changes:
```json
{
  "type": "status",
  "sessionId": "uuid",
  "data": { 
    "status": "stopped",
    "message": "...",
    "exitCode": 0
  },
  "timestamp": 1234567890
}
```

### file_change
File system notifications:
```json
{
  "type": "file_change",
  "sessionId": "uuid",
  "data": { 
    "path": "/path/to/file",
    "action": "created|modified|deleted"
  },
  "timestamp": 1234567890
}
```

## KV Store Schema

### Session Storage
- **Key:** `session_<channelID>`
- **Value:** JSON-encoded ChannelSession

```json
{
  "session_id": "uuid",
  "project_path": "/path/to/project",
  "user_id": "user_id",
  "created_at": 1234567890,
  "last_message_at": 1234567890
}
```

## Bot User

- **Username:** `claude-code`
- **Display Name:** Claude Code
- **Description:** AI-powered coding assistant

The bot is created automatically on plugin activation. All Claude Code responses are posted as this bot user.

## Error Handling

- Bridge server connection failures → user-friendly ephemeral messages
- Session creation errors → detailed error messages
- WebSocket disconnects → automatic reconnection with exponential backoff
- Missing sessions → prompt user to start a session
- Duplicate sessions → prevent and show warning

## Configuration

Set via **System Console → Plugins → Claude Code**:

- **Bridge Server URL** (default: http://localhost:3002)
- **Claude Code CLI Path** (default: /usr/local/bin/claude-code)
- **Enable File Operations** (default: true)

## Development

### Prerequisites
- Go 1.21+
- Mattermost Server 9.0+

### Testing Locally
1. Start bridge server: `cd bridge-server && npm run dev`
2. Build plugin: `cd .. && make build`
3. Deploy to local Mattermost
4. Activate plugin
5. Test commands in any channel

### Debugging
- Check Mattermost logs: **System Console → Logs**
- Enable debug logging in plugin settings
- Monitor bridge server logs

## API Dependencies

- `github.com/mattermost/mattermost/server/public/plugin` - Plugin framework
- `github.com/mattermost/mattermost/server/public/model` - Mattermost models
- `github.com/gorilla/websocket` - WebSocket client
- `github.com/pkg/errors` - Error handling

## Next Steps

- **Issue #5:** Interactive message components (buttons, dialogs)
- **Issue #6:** File explorer and operations
- **Issue #7:** Testing and documentation
- **Issue #8:** CI/CD and deployment

## License

GPL-3.0
