#!/bin/bash
#
# Update script for Claude Code Bridge Server
# Usage: ./scripts/update-bridge.sh
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BRIDGE_DIR="$PROJECT_ROOT/bridge-server"

echo "=== Claude Code Bridge Server Update ==="
echo ""

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "Error: Not a git repository. Please run from the project root."
    exit 1
fi

# Navigate to project root
cd "$PROJECT_ROOT"

# Pull latest changes
echo "Pulling latest changes..."
git pull origin main

# Navigate to bridge server directory
cd "$BRIDGE_DIR"

# Install dependencies
echo "Installing dependencies..."
npm install

# Build the server
echo "Building server..."
npm run build

# Check if PM2 is managing the process
if command -v pm2 &> /dev/null; then
    if pm2 list | grep -q "claude-code-bridge"; then
        echo "Restarting with PM2..."
        pm2 restart claude-code-bridge
        echo ""
        echo "=== Update complete! ==="
        pm2 status claude-code-bridge
        exit 0
    fi
fi

# Check if running with Docker
if [ -f "$PROJECT_ROOT/docker-compose.prod.yml" ]; then
    if docker ps | grep -q "claude-code-bridge"; then
        echo "Restarting Docker container..."
        cd "$PROJECT_ROOT"
        docker-compose -f docker-compose.prod.yml down
        docker-compose -f docker-compose.prod.yml up -d --build
        echo ""
        echo "=== Update complete! ==="
        docker-compose -f docker-compose.prod.yml ps
        exit 0
    fi
fi

# Check if running with systemd
if systemctl is-active --quiet claude-code-bridge 2>/dev/null; then
    echo "Restarting systemd service..."
    sudo systemctl restart claude-code-bridge
    echo ""
    echo "=== Update complete! ==="
    systemctl status claude-code-bridge --no-pager
    exit 0
fi

echo ""
echo "=== Build complete! ==="
echo ""
echo "The bridge server has been updated and rebuilt."
echo "Please restart the server manually using your preferred method:"
echo ""
echo "  npm start                    # Direct"
echo "  pm2 restart claude-code-bridge   # PM2"
echo "  docker-compose -f docker-compose.prod.yml up -d  # Docker"
echo "  sudo systemctl restart claude-code-bridge        # systemd"
echo ""
