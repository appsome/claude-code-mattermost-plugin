# Bridge Server API Reference

## Overview

The Bridge Server provides a REST API for managing Claude Code sessions and a WebSocket API for real-time communication.

**Base URL:** `http://localhost:3002` (configurable)

**Authentication:** Bearer token (configured in plugin settings)

```
Authorization: Bearer YOUR_API_TOKEN
```

## REST API

### Sessions

#### Create Session

Create a new Claude Code session.

```http
POST /api/sessions
Content-Type: application/json

{
  "projectPath": "/path/to/project",
  "userId": "mattermost_user_id",
  "channelId": "mattermost_channel_id"
}
```

**Response:**
```json
{
  "sessionId": "abc123",
  "status": "active",
  "projectPath": "/path/to/project",
  "createdAt": "2024-03-09T12:00:00Z"
}
```

**Status Codes:**
- `201 Created` - Session created successfully
- `400 Bad Request` - Invalid project path or missing parameters
- `500 Internal Server Error` - Failed to spawn CLI

---

#### Get Session

Get information about a specific session.

```http
GET /api/sessions/:id
```

**Response:**
```json
{
  "sessionId": "abc123",
  "status": "active",
  "projectPath": "/path/to/project",
  "userId": "user123",
  "channelId": "channel456",
  "createdAt": "2024-03-09T12:00:00Z",
  "lastActive": "2024-03-09T12:30:00Z"
}
```

**Status Codes:**
- `200 OK` - Session found
- `404 Not Found` - Session does not exist

---

#### Stop Session

Stop and cleanup a session.

```http
DELETE /api/sessions/:id
```

**Response:**
```json
{
  "message": "Session stopped successfully",
  "sessionId": "abc123"
}
```

**Status Codes:**
- `200 OK` - Session stopped
- `404 Not Found` - Session does not exist

---

### Messages

#### Send Prompt

Send a user message to Claude Code.

```http
POST /api/sessions/:id/prompt
Content-Type: application/json

{
  "content": "Add authentication to the login page"
}
```

**Response:**
```json
{
  "message": "Prompt sent successfully",
  "timestamp": 1709985600000
}
```

**Status Codes:**
- `200 OK` - Prompt sent (response will come via WebSocket)
- `404 Not Found` - Session does not exist
- `400 Bad Request` - Empty content

---

#### Inject Context

Add context (thread history, file content) to session.

```http
POST /api/sessions/:id/context
Content-Type: application/json

{
  "source": "mattermost-thread",
  "threadId": "thread_xyz",
  "content": "Thread Context from #general...",
  "action": "summarize",
  "metadata": {
    "channelName": "general",
    "rootPostId": "xyz789",
    "messageCount": 12,
    "participants": ["@wojtek", "@ada"]
  }
}
```

**Response:**
```json
{
  "message": "Context injected successfully",
  "metadata": {
    "contentLength": 456,
    "actionRequested": true,
    "source": "mattermost-thread"
  }
}
```

**Status Codes:**
- `200 OK` - Context added
- `404 Not Found` - Session does not exist
- `400 Bad Request` - Missing required fields

---

### Files

#### List Files

Get project file tree.

```http
GET /api/sessions/:id/files
```

**Query Parameters:**
- `maxDepth` (optional) - Maximum directory depth (default: 5)
- `includeHidden` (optional) - Include hidden files (default: false)

**Response:**
```json
{
  "files": [
    {
      "name": "src",
      "path": "src",
      "type": "directory",
      "children": [
        {
          "name": "index.js",
          "path": "src/index.js",
          "type": "file",
          "size": 1234
        }
      ]
    }
  ]
}
```

**Status Codes:**
- `200 OK` - File list retrieved
- `404 Not Found` - Session does not exist

---

#### Get File Content

Read file content.

```http
GET /api/sessions/:id/files/:path
```

**Example:**
```http
GET /api/sessions/abc123/files/src/index.js
```

**Response:**
```json
{
  "path": "src/index.js",
  "content": "console.log('Hello, world!');",
  "size": 28,
  "encoding": "utf8"
}
```

**Status Codes:**
- `200 OK` - File content retrieved
- `404 Not Found` - Session or file does not exist
- `403 Forbidden` - File outside project directory

---

#### Create File

Create a new file.

```http
POST /api/sessions/:id/files
Content-Type: application/json

{
  "path": "src/components/NewComponent.tsx",
  "content": "export default function NewComponent() {\n  return <div>Hello</div>;\n}"
}
```

**Response:**
```json
{
  "message": "File created successfully",
  "path": "src/components/NewComponent.tsx"
}
```

**Status Codes:**
- `201 Created` - File created
- `400 Bad Request` - Invalid path or content
- `409 Conflict` - File already exists
- `404 Not Found` - Session does not exist

---

#### Update File

Update existing file content.

```http
PUT /api/sessions/:id/files/:path
Content-Type: application/json

{
  "content": "Updated file content..."
}
```

**Response:**
```json
{
  "message": "File updated successfully",
  "path": "src/index.js"
}
```

**Status Codes:**
- `200 OK` - File updated
- `404 Not Found` - Session or file does not exist
- `403 Forbidden` - File outside project directory

---

#### Delete File

Delete a file.

```http
DELETE /api/sessions/:id/files/:path
```

**Response:**
```json
{
  "message": "File deleted successfully",
  "path": "src/old-file.js"
}
```

**Status Codes:**
- `200 OK` - File deleted
- `404 Not Found` - Session or file does not exist
- `403 Forbidden` - File outside project directory

---

### Actions

#### Approve Change

Approve a proposed code change.

```http
POST /api/sessions/:id/approve
Content-Type: application/json

{
  "changeId": "change_123"
}
```

**Response:**
```json
{
  "message": "Change approved",
  "changeId": "change_123"
}
```

**Status Codes:**
- `200 OK` - Change approved
- `404 Not Found` - Session or change does not exist

---

#### Reject Change

Reject a proposed code change.

```http
POST /api/sessions/:id/reject
Content-Type: application/json

{
  "changeId": "change_123",
  "reason": "Wrong approach, please use Redux instead"
}
```

**Response:**
```json
{
  "message": "Change rejected",
  "changeId": "change_123"
}
```

**Status Codes:**
- `200 OK` - Change rejected
- `404 Not Found` - Session or change does not exist

---

### Health

#### Health Check

Check bridge server health.

```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 123456,
  "sessions": 3,
  "timestamp": "2024-03-09T12:00:00Z"
}
```

**Status Codes:**
- `200 OK` - Server is healthy
- `503 Service Unavailable` - Server is unhealthy

---

## WebSocket API

**Endpoint:** `ws://localhost:3002/ws`

### Connection

Connect with session ID as query parameter:

```javascript
const ws = new WebSocket('ws://localhost:3002/ws?sessionId=abc123');
```

### Server → Client Events

#### Message Event

Claude's response to user prompt.

```json
{
  "type": "message",
  "sessionId": "abc123",
  "content": "I'll add authentication to the login page...",
  "timestamp": 1709985600000,
  "complete": false
}
```

**Fields:**
- `type` - Always `"message"`
- `sessionId` - Session identifier
- `content` - Response text (may be partial)
- `timestamp` - Unix timestamp (ms)
- `complete` - `true` when message is fully sent

---

#### Status Event

Session status change.

```json
{
  "type": "status",
  "sessionId": "abc123",
  "status": "idle",
  "timestamp": 1709985600000
}
```

**Status Values:**
- `"active"` - Session running, processing request
- `"idle"` - Session ready for input
- `"stopped"` - Session ended

---

#### File Change Event

File modified by Claude.

```json
{
  "type": "file_change",
  "sessionId": "abc123",
  "path": "src/auth/login.js",
  "action": "modified",
  "timestamp": 1709985600000
}
```

**Action Values:**
- `"created"` - New file created
- `"modified"` - Existing file updated
- `"deleted"` - File removed

---

#### Error Event

Error occurred during processing.

```json
{
  "type": "error",
  "sessionId": "abc123",
  "error": "File not found: src/missing.js",
  "timestamp": 1709985600000
}
```

---

### Client → Server Events

#### Prompt Event

Send user message (alternative to REST API).

```json
{
  "type": "prompt",
  "sessionId": "abc123",
  "content": "Add a logout button"
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong",
  "code": "ERROR_CODE",
  "details": {
    "additionalInfo": "Optional extra context"
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `SESSION_NOT_FOUND` | Session ID does not exist |
| `INVALID_PROJECT_PATH` | Project path is invalid or inaccessible |
| `CLI_SPAWN_FAILED` | Failed to start Claude Code CLI |
| `FILE_NOT_FOUND` | Requested file does not exist |
| `PERMISSION_DENIED` | File operation not allowed (outside project) |
| `VALIDATION_ERROR` | Request validation failed |
| `INTERNAL_ERROR` | Unexpected server error |

---

## Rate Limits

- **Global:** 100 requests/minute per IP
- **Per Session:** 30 prompts/minute
- **File Operations:** 50 operations/minute per session

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1709985660
```

---

## Examples

### Complete Flow (REST + WebSocket)

```javascript
// 1. Create session
const sessionResp = await fetch('http://localhost:3002/api/sessions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    projectPath: '/home/user/project',
    userId: 'user123',
    channelId: 'channel456'
  })
});
const { sessionId } = await sessionResp.json();

// 2. Connect WebSocket
const ws = new WebSocket(`ws://localhost:3002/ws?sessionId=${sessionId}`);

ws.on('message', (data) => {
  const event = JSON.parse(data);
  console.log('Received:', event.type, event.content);
});

// 3. Send prompt
await fetch(`http://localhost:3002/api/sessions/${sessionId}/prompt`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    content: 'Add a login page'
  })
});

// 4. Receive response via WebSocket
// (WebSocket messages will arrive asynchronously)

// 5. Stop session when done
await fetch(`http://localhost:3002/api/sessions/${sessionId}`, {
  method: 'DELETE',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
});
```

---

## Changelog

### v1.0.0 (2024-03-09)
- Initial API release
- Session management endpoints
- File operations
- Context injection
- WebSocket real-time updates
