# Installation Guide

## Prerequisites

Before installing the Claude Code Mattermost Plugin, ensure you have:

- **Mattermost Server** 9.0 or higher
- **System Admin** access to Mattermost
- **Claude Code CLI** installed on the bridge server machine
- **Node.js** 22+ (for bridge server)
- **Bridge Server** running (or ability to run it)

## Installation Methods

### Method 1: Via Mattermost Marketplace (Recommended)

> **Note:** Marketplace listing pending approval. Manual installation required currently.

1. Log in to Mattermost as System Admin
2. Go to **System Console** → **Plugins** → **Marketplace**
3. Search for "Claude Code"
4. Click **Install**
5. Enable the plugin
6. Configure settings (see Configuration section below)

### Method 2: Manual Installation from Release

#### Step 1: Download Plugin

Download the latest release for your platform:

```bash
# For Linux
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-linux-amd64.tar.gz

# For macOS
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-darwin-amd64.tar.gz

# For Windows
wget https://github.com/appsome/claude-code-mattermost-plugin/releases/latest/download/com.appsome.claudecode-windows-amd64.tar.gz
```

#### Step 2: Install Plugin in Mattermost

**Via System Console (Recommended):**

1. Log in to Mattermost as System Admin
2. Go to **System Console** → **Plugins** → **Management**
3. Click **Upload Plugin**
4. Select the downloaded `.tar.gz` file
5. Click **Upload**
6. Enable the plugin

**Via CLI:**

```bash
# On Mattermost server
sudo -u mattermost mattermost plugin add /path/to/com.appsome.claudecode.tar.gz
sudo -u mattermost mattermost plugin enable com.appsome.claudecode
```

#### Step 3: Configure Plugin

1. Go to **System Console** → **Plugins** → **Claude Code**
2. Configure the following settings:
   - **Bridge Server URL**: `http://localhost:3002` (or your bridge server URL)
   - **Claude Code CLI Path**: `/usr/local/bin/claude-code` (or your CLI path)
   - **API Token**: Generate a secure token for authentication
3. Click **Save**

## Bridge Server Setup

The bridge server manages Claude Code CLI sessions and must be running for the plugin to work.

### Option 1: Run with Node.js (Development/Single User)

```bash
# Clone repository
git clone https://github.com/appsome/claude-code-mattermost-plugin.git
cd claude-code-mattermost-plugin/bridge-server

# Install dependencies
npm install

# Build
npm run build

# Start server
npm start
```

The bridge server will start on `http://localhost:3002` by default.

### Option 2: Run with Docker (Production)

```bash
# Clone repository
git clone https://github.com/appsome/claude-code-mattermost-plugin.git
cd claude-code-mattermost-plugin

# Build and start
docker-compose -f docker-compose.prod.yml up -d
```

### Option 3: Run with systemd (Production)

Create a systemd service file:

```bash
sudo nano /etc/systemd/system/claude-code-bridge.service
```

Add the following content:

```ini
[Unit]
Description=Claude Code Bridge Server
After=network.target

[Service]
Type=simple
User=mattermost
WorkingDirectory=/opt/claude-code-bridge
ExecStart=/usr/bin/node /opt/claude-code-bridge/dist/index.js
Restart=always
Environment=NODE_ENV=production
Environment=PORT=3002
Environment=CLAUDE_CODE_PATH=/usr/local/bin/claude-code

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable claude-code-bridge
sudo systemctl start claude-code-bridge
sudo systemctl status claude-code-bridge
```

### Bridge Server Configuration

Create a `.env` file in the bridge-server directory:

```bash
# Server Configuration
PORT=3002
NODE_ENV=production

# Database
DATABASE_PATH=/var/lib/claude-code-bridge/sessions.db

# Claude Code CLI
CLAUDE_CODE_PATH=/usr/local/bin/claude-code

# Security
API_TOKEN=your-secure-token-here

# Logging
LOG_LEVEL=info
```

## Claude Code CLI Installation

The Claude Code CLI must be installed on the bridge server machine.

### Installation

```bash
# Install globally via npm
npm install -g claude-code

# Verify installation
claude-code --version
```

### Configuration

Claude Code CLI requires API credentials. Set them up:

```bash
# Configure Claude API key
claude-code config set apiKey YOUR_ANTHROPIC_API_KEY
```

See [Claude Code documentation](https://github.com/siteboon/claudecode) for more details.

## Verification

### 1. Check Plugin Status

In Mattermost:
1. Go to **System Console** → **Plugins** → **Management**
2. Verify "Claude Code" shows as **Enabled**
3. Check for any error messages

### 2. Test Bridge Server

```bash
curl http://localhost:3002/health
```

Expected response:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 123,
  "sessions": 0,
  "timestamp": "2024-03-09T12:00:00Z"
}
```

### 3. Test Slash Command

In any Mattermost channel:
1. Type `/claude-help`
2. You should see a help message from the bot
3. If it works, the plugin is properly configured!

### 4. Start a Session

```
/claude-start /path/to/your/project
```

You should see a confirmation message from the bot.

## Troubleshooting

### Plugin Not Loading

**Symptoms:** Plugin shows as disabled or doesn't appear in plugin list.

**Solutions:**
- Check Mattermost logs: `docker logs mattermost` or `/var/log/mattermost/mattermost.log`
- Verify plugin file was uploaded correctly
- Ensure Mattermost version is 9.0+
- Check plugin manifest is valid: `tar -tzf plugin.tar.gz | head`

### Bridge Server Connection Failed

**Symptoms:** Commands fail with "Bridge server unavailable" error.

**Solutions:**
- Verify bridge server is running: `curl http://localhost:3002/health`
- Check bridge server logs
- Verify plugin configuration has correct bridge URL
- Check firewall rules (port 3002 must be accessible)
- For remote bridge server, use full URL: `http://bridge.example.com:3002`

### Commands Not Responding

**Symptoms:** Slash commands don't trigger any response.

**Solutions:**
- Check bot user was created (should happen automatically)
- Verify you have permission to use slash commands in the channel
- Check Mattermost logs for command registration errors
- Try `/claude-help` first to test basic functionality

### Claude Code CLI Not Found

**Symptoms:** "Claude Code CLI not found" error from bridge server.

**Solutions:**
- Verify CLI is installed: `which claude-code`
- Update plugin configuration with correct CLI path
- For systemd service, update `CLAUDE_CODE_PATH` environment variable
- Check CLI has execute permissions: `chmod +x /path/to/claude-code`

### Session Won't Start

**Symptoms:** `/claude-start` fails with error.

**Solutions:**
- Verify project path exists and is accessible
- Check user running bridge server has read/write permissions on project
- Ensure Claude Code CLI is properly configured with API key
- Check bridge server logs for detailed error messages

### WebSocket Connection Issues

**Symptoms:** Messages don't stream in real-time.

**Solutions:**
- Check WebSocket port (3002 by default) is not blocked
- Verify no proxy is interfering with WebSocket connections
- Check browser console for WebSocket errors (if using webapp components)
- Restart bridge server and plugin

## Security Considerations

### Production Deployment

For production use:

1. **Use HTTPS:** Configure Mattermost with TLS certificate
2. **Secure Bridge Server:** Use firewall to restrict access to bridge server port
3. **Authentication:** Use strong API token for bridge server authentication
4. **File Permissions:** Restrict project directories to necessary users only
5. **Regular Updates:** Keep plugin, bridge server, and CLI updated

### Network Configuration

**Internal Network (Recommended):**
```
[Mattermost] --internal--> [Bridge Server] --local--> [Claude Code CLI]
```

**Firewall Rules:**
```bash
# Allow Mattermost server to reach bridge server
sudo ufw allow from MATTERMOST_IP to any port 3002

# Block external access to bridge server
sudo ufw deny 3002
```

## Upgrading

### Plugin Upgrade

1. Download new version
2. Upload via System Console (same as installation)
3. Mattermost will handle the upgrade
4. Restart Mattermost if needed

### Bridge Server Upgrade

```bash
cd claude-code-mattermost-plugin/bridge-server
git pull origin main
npm install
npm run build
sudo systemctl restart claude-code-bridge
```

### CLI Upgrade

```bash
npm update -g claude-code
```

## Uninstallation

### Remove Plugin

1. **System Console** → **Plugins** → **Management**
2. Find "Claude Code" plugin
3. Click **Remove**
4. Confirm removal

### Stop Bridge Server

**systemd:**
```bash
sudo systemctl stop claude-code-bridge
sudo systemctl disable claude-code-bridge
sudo rm /etc/systemd/system/claude-code-bridge.service
```

**Docker:**
```bash
docker-compose -f docker-compose.prod.yml down
```

### Cleanup Data

```bash
# Remove plugin data from Mattermost
# (Located in Mattermost data directory, varies by installation)

# Remove bridge server data
rm -rf /var/lib/claude-code-bridge

# Remove CLI configuration (optional)
rm -rf ~/.config/claude-code
```

## Getting Help

- **GitHub Issues:** https://github.com/appsome/claude-code-mattermost-plugin/issues
- **Documentation:** https://github.com/appsome/claude-code-mattermost-plugin
- **Mattermost Community:** https://community.mattermost.com

## Next Steps

After installation:

1. Read the [User Guide](USER_GUIDE.md) to learn how to use the plugin
2. Check [Architecture](ARCHITECTURE.md) to understand how it works
3. For development, see [Development Guide](DEVELOPMENT.md)
