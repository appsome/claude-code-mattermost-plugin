# Claude Code Mattermost Plugin

Integrate Claude Code AI assistant directly into Mattermost for AI-powered coding sessions in your team chat.

## Features

- 🤖 **Native Mattermost Integration** - Use slash commands instead of separate UI
- 💬 **Interactive Messages** - Approve/reject code changes with buttons
- 📁 **File Operations** - Browse and edit files via Mattermost dialogs
- 🔄 **Real-time Updates** - WebSocket connection for instant responses
- 📱 **Mobile-Friendly** - Works on Mattermost mobile apps

## Quick Start

### Prerequisites

- Mattermost Server 9.0+
- Go 1.21+
- Node.js 22+
- Docker & Docker Compose (for development)

### Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/appsome/claude-code-mattermost-plugin.git
   cd claude-code-mattermost-plugin
   ```

2. **Install dependencies:**
   ```bash
   # Backend (Go modules will download automatically)
   cd server && go mod download && cd ..
   
   # Frontend
   cd webapp && npm install && cd ..
   ```

3. **Start development environment:**
   ```bash
   make dev
   ```

   This will start Mattermost at http://localhost:8065

4. **Build the plugin:**
   ```bash
   make build
   ```

5. **Create plugin bundle:**
   ```bash
   make bundle
   ```

6. **Upload to Mattermost:**
   - Go to System Console → Plugins → Management
   - Upload `dist/co.appsome.claudecode.tar.gz`
   - Enable the plugin

### Available Commands

```bash
make build        # Build plugin for current platform
make build-all    # Build for all platforms
make test         # Run tests
make bundle       # Create plugin bundle
make dev          # Start development environment
make dev-down     # Stop development environment
make clean        # Remove build artifacts
```

## Usage

Once installed, use these slash commands in any channel:

- `/claude <message>` - Send a message to Claude Code
- `/claude-start [project-path]` - Start a new coding session
- `/claude-stop` - Stop the current session
- `/claude-help` - Show help information

## Project Status

✅ **Issue #2: Project Setup** - COMPLETE  
⏳ **Issue #3: Bridge Server** - In Progress  
⏳ **Issue #4: Slash Commands** - Pending  
⏳ **Issue #5: Interactive Components** - Pending  
⏳ **Issue #6: File Operations** - Pending  

See [Issues](https://github.com/appsome/claude-code-mattermost-plugin/issues) for the full roadmap.

## Documentation

- [Development Guide](docs/DEVELOPMENT.md) - Coming soon
- [Architecture](docs/ARCHITECTURE.md) - Coming soon
- [Contributing](CONTRIBUTING.md) - Coming soon

## Project Structure

```
claude-code-mattermost-plugin/
├── server/              # Go backend
│   ├── plugin.go       # Main plugin entry
│   ├── commands.go     # Slash command handlers
│   └── ...
├── webapp/             # React frontend
│   ├── src/
│   │   └── index.tsx   # Plugin entry point
│   └── package.json
├── plugin.json         # Plugin manifest
├── Makefile           # Build automation
├── docker-compose.yml # Development environment
└── README.md
```

## License

GPL-3.0 - See [LICENSE](LICENSE) for details

## Support

- [GitHub Issues](https://github.com/appsome/claude-code-mattermost-plugin/issues)
- [Documentation](https://github.com/appsome/claude-code-mattermost-plugin)

## Credits

Built by [Appsome](https://github.com/appsome)  
Inspired by [claudecodeui](https://github.com/siteboon/claudecodeui)
