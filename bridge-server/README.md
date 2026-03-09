# Claude Code Bridge Server

REST API + WebSocket server that manages Claude Code CLI sessions for the Mattermost plugin.

## Features

- **Multi-session management**: Run multiple Claude Code CLI instances simultaneously
- **Real-time streaming**: WebSocket interface for live CLI output
- **REST API**: Complete API for session, message, file, and git operations
- **Persistent storage**: SQLite database for session and message history
- **Process management**: Automatic CLI process spawning, monitoring, and cleanup
- **Logging**: Comprehensive logging with winston

## Architecture

```
┌─────────────────────┐
│ Mattermost Plugin   │
│  (Go + React)       │
└──────────┬──────────┘
           │ REST API + WebSocket
           ▼
┌─────────────────────┐
│  Bridge Server      │
│  (Node.js + Express)│
├─────────────────────┤
│ • Session Manager   │
│ • CLI Spawner       │
│ • WebSocket Handler │
│ • SQLite Database   │
└──────────┬──────────┘
           │ spawn/stdin/stdout
           ▼
┌─────────────────────┐
│  Claude Code CLI    │
│  (Multiple Processes)│
└─────────────────────┘
```

## Requirements

- Node.js 22+
- Claude Code CLI installed
- SQLite3

## Installation

```bash
cd bridge-server
npm install
```

## Configuration

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key settings:
- `PORT`: Server port (default: 3002)
- `CLAUDE_CODE_PATH`: Path to Claude Code CLI
- `MAX_SESSIONS`: Maximum concurrent sessions
- `DATABASE_PATH`: SQLite database location

## Development

```bash
# Start with hot reload
npm run dev

# Build TypeScript
npm run build

# Run tests
npm test
```

## Production

```bash
# Build and start
npm run build
npm start
```

## API Endpoints

### Sessions

- `POST /api/sessions` - Create new session
- `GET /api/sessions` - List all sessions
- `GET /api/sessions/:id` - Get session details
- `DELETE /api/sessions/:id` - Stop and delete session

### Messages

- `POST /api/sessions/:id/message` - Send message to Claude Code
- `GET /api/sessions/:id/messages` - Get message history

### Files

- `GET /api/sessions/:id/files` - List project files
- `GET /api/sessions/:id/files/*` - Get file content
- `PUT /api/sessions/:id/files/*` - Update file
- `POST /api/sessions/:id/files` - Create file
- `DELETE /api/sessions/:id/files/*` - Delete file

### Git

- `GET /api/sessions/:id/git/status` - Git status
- `POST /api/sessions/:id/git/commit` - Commit changes
- `POST /api/sessions/:id/git/push` - Push to remote

### Health

- `GET /health` - Health check

## WebSocket

Connect to `/ws` for real-time updates:

```typescript
const ws = new WebSocket('ws://localhost:3002/ws');

// Subscribe to session
ws.send(JSON.stringify({
  type: 'subscribe',
  sessionId: 'session-id-here'
}));

// Receive messages
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message.type, message.data);
};
```

Message types:
- `output` - CLI stdout
- `error` - CLI stderr
- `status` - Status updates
- `file_change` - File system changes

## Database Schema

### Sessions Table
```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  project_path TEXT NOT NULL,
  mattermost_user_id TEXT NOT NULL,
  mattermost_channel_id TEXT NOT NULL,
  cli_pid INTEGER,
  status TEXT CHECK(status IN ('active', 'stopped', 'error')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

### Messages Table
```sql
CREATE TABLE messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  role TEXT CHECK(role IN ('user', 'assistant', 'system')),
  content TEXT NOT NULL,
  timestamp INTEGER NOT NULL,
  FOREIGN KEY(session_id) REFERENCES sessions(id)
);
```

## Testing

### Manual Testing

1. Start the server:
   ```bash
   npm run dev
   ```

2. Create a session:
   ```bash
   curl -X POST http://localhost:3002/api/sessions \
     -H "Content-Type: application/json" \
     -d '{
       "projectPath": "/path/to/project",
       "mattermostUserId": "user123",
       "mattermostChannelId": "channel456"
     }'
   ```

3. Send a message:
   ```bash
   curl -X POST http://localhost:3002/api/sessions/SESSION_ID/message \
     -H "Content-Type: application/json" \
     -d '{"message": "Hello Claude!"}'
   ```

4. Connect WebSocket:
   ```javascript
   const ws = new WebSocket('ws://localhost:3002/ws');
   ws.onopen = () => {
     ws.send(JSON.stringify({
       type: 'subscribe',
       sessionId: 'SESSION_ID'
     }));
   };
   ws.onmessage = (e) => console.log(JSON.parse(e.data));
   ```

## Logging

Logs are written to:
- Console (with colors)
- File specified in `LOG_FILE` (default: `./logs/bridge-server.log`)

Log levels: `error`, `warn`, `info`, `debug`

## Error Handling

- Invalid requests return 400 with error message
- Missing resources return 404
- Server errors return 500
- All errors are logged

## Graceful Shutdown

The server handles `SIGTERM` and `SIGINT` signals:
1. Stop accepting new connections
2. Close WebSocket server
3. Kill all running CLI processes
4. Close database connections
5. Exit process

## Security Considerations

⚠️ **This server is designed for internal use only**

- No authentication (handled by Mattermost plugin)
- No rate limiting
- Accepts any CORS origin by default
- Full file system access within project paths

For production:
- Deploy behind a firewall
- Use reverse proxy with authentication
- Restrict CORS origins
- Add rate limiting
- Validate project paths

## Troubleshooting

### CLI process won't start
- Check `CLAUDE_CODE_PATH` is correct
- Verify Claude Code CLI is installed: `which claude-code`
- Check file permissions

### WebSocket not connecting
- Ensure server is running on correct port
- Check firewall rules
- Verify WebSocket path `/ws`

### Database locked errors
- Only one server instance per database
- Check file permissions on `DATABASE_PATH`

### High memory usage
- Reduce `MAX_SESSIONS`
- Implement session timeout cleanup
- Monitor CLI process memory

## License

GPL-3.0 (same as parent project)

## References

- [claudecodeui](https://github.com/siteboon/claudecodeui) - Original inspiration
- [Express.js](https://expressjs.com/)
- [ws (WebSocket)](https://github.com/websockets/ws)
- [better-sqlite3](https://github.com/WiseLibs/better-sqlite3)
