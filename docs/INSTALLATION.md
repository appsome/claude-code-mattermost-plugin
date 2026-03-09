# Installation Guide

## Prerequisites

- Mattermost Server 9.0+
- Claude Code CLI installed on the bridge server
- Node.js 22+ (for bridge server)
- Go 1.21+ (for building from source)

## Method 1: Mattermost Marketplace (Recommended)

1. Log in to Mattermost as System Admin
2. Go to **System Console** → **Plugins** → **Marketplace**
3. Search for "Claude Code"
4. Click **Install**
5. Configure settings (see [Configuration](#configuration) section)
6. Enable the plugin

## Method 2: Manual Installation

### Download Latest Release

```bash
# For Linux
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-linux-amd64.tar.gz

# For macOS
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-darwin-amd64.tar.gz

# For Windows
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-windows-amd64.tar.gz
```

### Upload Plugin

1. Go to **System Console** → **Plugins** → **Management**
2. Click **Upload Plugin**
3. Select the downloaded `.tar.gz` file
4. Click **Upload**
5. Enable the plugin

### Configuration

1. Go to **System Console** → **Plugins** → **Claude Code**
2. Configure the following settings:

| Setting | Description | Default |
|---------|-------------|---------|
| Bridge Server URL | URL of the Claude Code bridge server | `http://localhost:3002` |
| Claude Code CLI Path | Path to the Claude Code CLI executable | `/usr/local/bin/claude` |
| Enable File Operations | Allow users to browse and edit files via dialogs | `true` |

3. Click **Save**

## Bridge Server Setup

The bridge server manages Claude Code CLI sessions and is required for the plugin to function.

### Option 1: Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/appsome/claude-code-mattermost-plugin
cd claude-code-mattermost-plugin

# Start the bridge server
docker-compose -f docker-compose.prod.yml up -d
```

The bridge server will be available at `http://localhost:3002`.

### Option 2: Manual Setup

```bash
# Clone the repository
git clone https://github.com/appsome/claude-code-mattermost-plugin
cd claude-code-mattermost-plugin/bridge-server

# Install dependencies
npm install

# Build the server
npm run build

# Start the server
npm start
```

### Option 3: Using PM2 (Process Manager)

```bash
# Install PM2 globally
npm install -g pm2

# Start the bridge server
cd bridge-server
npm run build
pm2 start dist/index.js --name claude-code-bridge

# Enable startup script
pm2 startup
pm2 save
```

### Environment Variables

Configure the bridge server using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `3002` |
| `DATABASE_PATH` | SQLite database path | `./data/sessions.db` |
| `CLAUDE_CODE_PATH` | Path to Claude Code CLI | `/usr/local/bin/claude` |
| `MAX_SESSIONS` | Maximum concurrent sessions | `100` |
| `SESSION_TIMEOUT_MS` | Session timeout in milliseconds | `3600000` (1 hour) |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## Verification

1. Open any Mattermost channel
2. Type `/claude help`
3. You should see the help message with available commands

### Available Commands

- `/claude <message>` - Send a message to Claude Code
- `/claude start` - Start a new Claude Code session
- `/claude stop` - Stop the current session
- `/claude status` - Check session status
- `/claude help` - Show help message

## Troubleshooting

### Plugin fails to connect to bridge server

1. Verify the bridge server is running:
   ```bash
   curl http://localhost:3002/health
   ```
2. Check the bridge server URL in plugin settings
3. Ensure there are no firewall rules blocking the connection

### Claude Code CLI not found

1. Verify Claude Code is installed:
   ```bash
   which claude
   claude --version
   ```
2. Update the CLI path in plugin settings or bridge server environment

### Sessions not persisting

1. Check the database path is writable
2. For Docker, ensure the volume is mounted correctly

## Updating

### Plugin Update

1. Download the latest release
2. Go to **System Console** → **Plugins** → **Management**
3. Upload the new version
4. The plugin will be automatically updated

### Bridge Server Update

Using the update script:

```bash
./scripts/update-bridge.sh
```

Or manually:

```bash
cd bridge-server
git pull origin main
npm install
npm run build
# Restart the server (method depends on how you're running it)
```

## Security Considerations

- The bridge server should only be accessible from the Mattermost server
- Consider using HTTPS for production deployments
- Review and restrict Claude Code CLI permissions as needed
- The plugin respects Mattermost channel permissions

## Support

- [GitHub Issues](https://github.com/appsome/claude-code-mattermost-plugin/issues)
- [Documentation](https://github.com/appsome/claude-code-mattermost-plugin)
