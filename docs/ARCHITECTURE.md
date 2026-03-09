# Architecture

## System Overview

The Claude Code Mattermost Plugin provides a native integration between Mattermost and Claude Code CLI, replacing the web UI with chat-based interactions.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Mattermost Server                       │
│  ┌───────────────────────────────────────────────────────┐ │
│  │            Claude Code Plugin                         │ │
│  │  ┌──────────────┐           ┌──────────────┐         │ │
│  │  │   Go Backend │◄─────────►│React Frontend│         │ │
│  │  │  (Commands,  │           │ (UI Components)        │ │
│  │  │   Sessions)  │           │                │         │ │
│  │  └──────┬───────┘           └────────────────┘         │ │
│  └─────────┼──────────────────────────────────────────────┘ │
└────────────┼──────────────────────────────────────────────┘
             │ REST + WebSocket
             ▼
┌─────────────────────────────────────────────────────────────┐
│                     Bridge Server (Node.js)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Session    │  │  WebSocket   │  │   SQLite     │     │
│  │   Manager    │  │    Server    │  │   Database   │     │
│  └──────┬───────┘  └──────────────┘  └──────────────┘     │
│         │                                                   │
│         ▼                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              CLI Spawner & Manager                   │  │
│  └──────────────────────────┬───────────────────────────┘  │
└─────────────────────────────┼──────────────────────────────┘
                              │ spawn/stdin/stdout
                              ▼
                    ┌──────────────────┐
                    │  Claude Code CLI │
                    │   (AI Assistant) │
                    └──────────────────┘
```

## Components

### 1. Mattermost Plugin (Go Backend)

**Responsibilities:**
- Register and handle slash commands
- Manage session lifecycle per channel
- Create interactive messages and dialogs
- Communicate with bridge server
- Persist session data in KV store

**Key Files:**
- `plugin.go` - Plugin lifecycle, initialization
- `commands.go` - Slash command routing and execution
- `session_manager.go` - Session state management
- `bridge_client.go` - HTTP client for bridge server
- `websocket_client.go` - WebSocket connection handling
- `thread_context.go` - Thread history integration
- `file_operations.go` - File browsing and management

**Data Flow:**
1. User types slash command in Mattermost
2. Plugin receives command via `ExecuteCommand()`
3. Plugin validates and routes to handler
4. Handler interacts with bridge server
5. Response posted back to channel via bot

### 2. Mattermost Plugin (React Frontend)

**Responsibilities:**
- Render custom UI components
- Handle interactive button clicks
- Display dialogs and modals
- Real-time message updates

**Key Files:**
- `index.tsx` - Plugin initialization
- `components/` - React components for UI

**Data Flow:**
1. Plugin registers React components
2. Components render in Mattermost UI
3. User interactions trigger API calls
4. Updates reflected in real-time

### 3. Bridge Server (Node.js)

**Responsibilities:**
- Spawn and manage Claude Code CLI processes
- Provide REST API for session operations
- WebSocket server for real-time communication
- Session persistence in SQLite
- File system operations
- Context injection (threads, files)

**Key Files:**
- `index.ts` - Express server setup
- `session-manager.ts` - Session lifecycle
- `cli-spawner.ts` - CLI process management
- `websocket-server.ts` - Real-time messaging
- `database.ts` - SQLite operations
- `api/sessions.ts` - Session endpoints
- `api/files.ts` - File operation endpoints
- `api/context.ts` - Context injection

**API Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/sessions` | Create new session |
| GET | `/api/sessions/:id` | Get session status |
| DELETE | `/api/sessions/:id` | Stop session |
| POST | `/api/sessions/:id/prompt` | Send message to Claude |
| POST | `/api/sessions/:id/context` | Inject context (thread/file) |
| GET | `/api/sessions/:id/files` | List project files |
| GET | `/api/sessions/:id/files/:path` | Get file content |
| POST | `/api/sessions/:id/files` | Create file |
| PUT | `/api/sessions/:id/files/:path` | Update file |
| DELETE | `/api/sessions/:id/files/:path` | Delete file |
| POST | `/api/sessions/:id/approve` | Approve change |
| POST | `/api/sessions/:id/reject` | Reject change |

**WebSocket Events:**

| Direction | Event | Payload | Description |
|-----------|-------|---------|-------------|
| Server→Client | `message` | `{content, type}` | Claude's response |
| Server→Client | `status` | `{status}` | Session status change |
| Server→Client | `file_change` | `{path, action}` | File modified |
| Server→Client | `error` | `{error}` | Error occurred |
| Client→Server | `prompt` | `{content}` | User message |

### 4. Claude Code CLI

**Responsibilities:**
- Process natural language instructions
- Read and modify files
- Execute shell commands
- Maintain conversation context

**Communication:**
- Bridge server spawns CLI as child process
- Stdin for sending prompts
- Stdout for receiving responses
- Stderr for errors and logs

## Data Models

### Session

```go
type Session struct {
    SessionID   string    `json:"session_id"`
    ChannelID   string    `json:"channel_id"`
    UserID      string    `json:"user_id"`
    ProjectPath string    `json:"project_path"`
    Status      string    `json:"status"` // "active", "idle", "stopped"
    CreatedAt   time.Time `json:"created_at"`
    LastActive  time.Time `json:"last_active"`
}
```

### Message

```typescript
interface Message {
    id: string;
    sessionId: string;
    type: 'user' | 'assistant' | 'system';
    content: string;
    timestamp: number;
}
```

### FileNode

```go
type FileNode struct {
    Name     string      `json:"name"`
    Path     string      `json:"path"`
    Type     string      `json:"type"` // "file" | "directory"
    Size     *int64      `json:"size,omitempty"`
    Children []FileNode  `json:"children,omitempty"`
}
```

## Communication Patterns

### 1. Synchronous Request/Response (REST)

Used for commands that need immediate response:

```
Plugin                    Bridge Server
   │                             │
   ├──POST /api/sessions─────────►
   │                             │
   ◄─────{session_id}────────────┤
   │                             │
```

### 2. Asynchronous Streaming (WebSocket)

Used for long-running operations and real-time updates:

```
Plugin                    Bridge Server
   │                             │
   ├──WebSocket Connect──────────►
   │                             │
   ├──{type: "prompt"}───────────►
   │                             │
   ◄──{type: "message"}──────────┤
   ◄──{type: "message"}──────────┤
   ◄──{type: "message"}──────────┤
   │                             │
```

### 3. Interactive Actions

User approvals flow through REST API:

```
User                 Plugin              Bridge Server
 │                      │                       │
 ├─Click Approve────────►                       │
 │                      ├─POST /api/.../approve►
 │                      │                       │
 │                      ◄─────{success}─────────┤
 │                      │                       │
 ◄─Confirmation Message─┤                       │
 │                      │                       │
```

## Security Considerations

### Authentication

- **Plugin to Bridge:** API token configured in plugin settings
- **Plugin to Mattermost:** Uses Mattermost plugin API (authenticated)
- **User to Plugin:** Mattermost handles user authentication

### Authorization

- Sessions are channel-scoped (one session per channel)
- Only users in the channel can interact with that session
- File operations restricted to project directory

### Input Validation

- File paths validated to prevent directory traversal
- Project paths must be absolute and existing
- Command arguments sanitized before execution

### Data Privacy

- Sessions stored in KV store (encrypted at rest by Mattermost)
- Bridge server database can be encrypted
- Thread context respects Mattermost permissions
- No user data sent to external services without consent

## Scalability

### Current Design

- **Single Bridge Server:** Handles all sessions
- **In-Memory State:** Session data in memory + SQLite backup
- **One CLI per Session:** Each session spawns separate Claude Code process

### Limitations

- Bridge server is single point of failure
- CLI processes consume significant memory (each runs independently)
- SQLite limits concurrent writes

### Future Improvements

- **Horizontal Scaling:** Multiple bridge servers with load balancer
- **Session Affinity:** Sticky sessions or distributed state
- **Process Pooling:** Reuse CLI processes across sessions
- **Database:** Move to PostgreSQL for better concurrency
- **Message Queue:** Redis for WebSocket message distribution

## Error Handling

### Plugin Layer

- Command validation errors → ephemeral message
- Bridge server unavailable → retry with exponential backoff
- WebSocket disconnect → automatic reconnection
- Session not found → clear message to user

### Bridge Server Layer

- CLI spawn failure → return error to plugin
- CLI crash → cleanup session, notify plugin
- File operation errors → return descriptive error
- Database errors → log and return 500

### Recovery Strategies

- **Session Recovery:** Persist session state, restore on restart
- **CLI Crash:** Detect via process exit, optionally restart
- **Connection Loss:** Automatic WebSocket reconnection
- **Partial Updates:** Transaction-like file operations

## Performance Considerations

### Plugin

- KV store operations are async
- WebSocket connection pooling
- Rate limiting on commands (prevent spam)

### Bridge Server

- Async I/O for file operations
- Stream large responses (don't buffer in memory)
- Connection limits (max concurrent sessions)
- SQLite WAL mode for better concurrency

### Mattermost

- Bot messages throttled (respect rate limits)
- Ephemeral messages don't persist (faster)
- Attachments for large content (not inline text)

## Deployment Architecture

### Development

```
[Developer Machine]
  ├── Mattermost (Docker)
  ├── Bridge Server (npm run dev)
  └── Claude Code CLI (local install)
```

### Production (Single Server)

```
[Server]
  ├── Mattermost Server
  │   └── Claude Code Plugin
  ├── Bridge Server (systemd/PM2)
  └── Claude Code CLI
```

### Production (Distributed)

```
[Load Balancer]
       │
       ├──► [Mattermost Server 1]
       │        └── Plugin
       │
       └──► [Mattermost Server 2]
                └── Plugin
                     │
                     ▼
           [Bridge Server Cluster]
               ├── Instance 1
               ├── Instance 2
               └── Instance 3
                     │
                     ▼
              [Shared Database]
```

## Monitoring & Observability

### Metrics (Bridge Server)

The bridge server exposes Prometheus metrics at `GET /metrics`:

- `claude_code_sessions_total` - Total sessions created (counter)
- `claude_code_sessions_active` - Currently active sessions (gauge)
- `claude_code_messages_total` - Total messages processed (counter)
- `claude_code_message_duration_seconds` - Message processing time (histogram)
- `claude_code_websocket_connections` - Active WebSocket connections (gauge)
- `claude_code_http_requests_total` - HTTP request count by method/path/status (counter)
- `claude_code_http_request_duration_seconds` - HTTP request duration (histogram)
- `claude_code_errors_total` - Error count by type (counter)
- `claude_code_db_queries_total` - Database query count (counter)
- `claude_code_file_operations_total` - File operation count (counter)

### Logs

- Plugin: Mattermost plugin logs
- Bridge: Winston/Pino structured logs
- CLI: Captured and persisted

### Health Checks

- Bridge: `GET /health` endpoint
- Plugin: Mattermost plugin health API
- CLI: Process existence check

## Future Enhancements

1. **Multi-User Sessions:** Multiple users collaborating in same session
2. **Session Sharing:** Share session across channels
3. **History Browsing:** Review past conversations
4. **Custom Actions:** User-defined interactive buttons
5. **Advanced File Editor:** Full IDE-like experience in dialogs
6. **Voice Input:** Speech-to-text for prompts
7. **Mobile Optimization:** Better mobile UI components
8. **Analytics Dashboard:** Usage statistics and insights

## References

- [Mattermost Plugin Architecture](https://developers.mattermost.com/integrate/plugins/)
- [WebSocket Protocol](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Claude Code GitHub](https://github.com/siteboon/claudecode)
- [Node.js Child Processes](https://nodejs.org/api/child_process.html)
