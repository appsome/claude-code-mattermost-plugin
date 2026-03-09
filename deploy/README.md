# Claude Code Bridge Server Deployment

This directory contains deployment configurations for the Claude Code Bridge Server.

## Docker

### Build the image

```bash
cd bridge-server
docker build -f Dockerfile.complete -t claude-code-bridge:latest .
```

### Run with Docker Compose

```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY="your-api-key"

# Optionally set projects path
export PROJECTS_PATH="/path/to/your/projects"

# Start the bridge server
docker-compose -f docker-compose.bridge.yml up -d
```

## Kubernetes

### Prerequisites

1. A Kubernetes cluster
2. `kubectl` configured to access your cluster
3. Your Anthropic API key

### Quick Start

1. **Edit the secret** with your Anthropic API key:

   ```bash
   # Edit deploy/kubernetes/secret.yaml
   # Replace "your-anthropic-api-key-here" with your actual key
   ```

2. **Deploy using kubectl**:

   ```bash
   kubectl apply -k deploy/kubernetes/
   ```

   Or apply individually:

   ```bash
   kubectl apply -f deploy/kubernetes/namespace.yaml
   kubectl apply -f deploy/kubernetes/secret.yaml
   kubectl apply -f deploy/kubernetes/configmap.yaml
   kubectl apply -f deploy/kubernetes/pvc.yaml
   kubectl apply -f deploy/kubernetes/deployment.yaml
   kubectl apply -f deploy/kubernetes/service.yaml
   ```

3. **Verify deployment**:

   ```bash
   kubectl -n claude-code get pods
   kubectl -n claude-code get svc
   ```

4. **Get the service URL** (for Mattermost plugin configuration):

   If Mattermost is in the same cluster:
   ```
   http://claude-code-bridge.claude-code.svc.cluster.local:3002
   ```

   If using an Ingress or LoadBalancer, configure accordingly.

### Configuration

#### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ANTHROPIC_API_KEY` | Your Anthropic API key (required) | - |
| `PORT` | Server port | `3002` |
| `MAX_SESSIONS` | Maximum concurrent sessions | `100` |
| `SESSION_TIMEOUT_MS` | Session timeout in milliseconds | `3600000` (1 hour) |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

#### Storage

- **claude-code-data**: Stores the SQLite session database
- **claude-code-projects**: Mount point for project files that Claude Code will work on

### Mattermost Plugin Configuration

In the Mattermost System Console, configure the Claude Code plugin:

1. Go to **System Console** → **Plugins** → **Claude Code**
2. Set **Bridge Server URL** to:
   - Same cluster: `http://claude-code-bridge.claude-code.svc.cluster.local:3002`
   - External: Your Ingress/LoadBalancer URL

### Scaling Considerations

The bridge server maintains WebSocket connections and spawns CLI processes. For high availability:

1. Use a single replica (stateful sessions)
2. Increase resources if needed
3. Consider session affinity if scaling

### Troubleshooting

Check logs:
```bash
kubectl -n claude-code logs -f deployment/claude-code-bridge
```

Check health:
```bash
kubectl -n claude-code exec deployment/claude-code-bridge -- wget -qO- http://localhost:3002/health
```

Verify Claude Code CLI:
```bash
kubectl -n claude-code exec deployment/claude-code-bridge -- claude --version
```
