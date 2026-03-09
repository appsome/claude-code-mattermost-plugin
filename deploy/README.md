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

## Kubernetes (Helm)

### Prerequisites

1. A Kubernetes cluster
2. Helm 3.x installed
3. Your Anthropic API key

### Quick Start

```bash
# Install the chart
helm install claude-code-bridge ./charts/claude-code-bridge \
  --namespace claude-code \
  --create-namespace \
  --set anthropicApiKey="your-anthropic-api-key"
```

### Using an Existing Secret

If you prefer to manage the API key separately:

```bash
# Create the secret first
kubectl create namespace claude-code
kubectl create secret generic my-anthropic-secret \
  --namespace claude-code \
  --from-literal=ANTHROPIC_API_KEY="your-api-key"

# Install with existing secret
helm install claude-code-bridge ./charts/claude-code-bridge \
  --namespace claude-code \
  --set existingSecret.enabled=true \
  --set existingSecret.name=my-anthropic-secret
```

### Configuration

See all available options:

```bash
helm show values ./charts/claude-code-bridge
```

#### Common Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `anthropicApiKey` | Your Anthropic API key | `""` |
| `existingSecret.enabled` | Use existing secret | `false` |
| `existingSecret.name` | Name of existing secret | `""` |
| `image.repository` | Docker image repository | `ghcr.io/appsome/claude-code-bridge` |
| `image.tag` | Docker image tag | `appVersion` |
| `config.maxSessions` | Maximum concurrent sessions | `100` |
| `config.sessionTimeoutMs` | Session timeout in ms | `3600000` |
| `config.logLevel` | Log level | `info` |
| `persistence.data.size` | Data volume size | `1Gi` |
| `persistence.projects.size` | Projects volume size | `10Gi` |
| `ingress.enabled` | Enable ingress | `false` |

### Upgrade

```bash
helm upgrade claude-code-bridge ./charts/claude-code-bridge \
  --namespace claude-code \
  --reuse-values
```

### Uninstall

```bash
helm uninstall claude-code-bridge --namespace claude-code
```

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
