# Agentic FOSS Infrastructure Orchestration

---

## 🚀 Demo Setup: OpenCode + Agent-Zero Integration

This section describes how to run the complete integration demo using local mock services.

### 1. Prerequisites

- Go 1.20+ installed
- Bash shell

### 2. Start Mock Services

In separate terminals, run:

```bash
go run ./agentic/testdata/mock_opencode_server.go
go run ./agentic/testdata/mock_agentzero_server.go
```

Or use the provided helper script:

```bash
./scripts/demo_start.sh
```

### 3. Configure Environment

```bash
cp agentic/.env.demo agentic/.env
cp agentic/config.demo.yaml agentic/config.yaml
```

### 4. Run the Demo

- Start the main integration service or run integration tests:

```bash
go run ./agentic/example.go
# or
./scripts/demo_test.sh
```

### 5. Stop Demo Services

```bash
./scripts/demo_stop.sh
```

### 6. Troubleshooting

- Ensure ports 8081 and 8082 are free.
- Check logs for errors in mock server terminals.
- See [docs/integrations/opencode-agentzero/demo.md](../docs/integrations/opencode-agentzero/demo.md) for full guide.

---

## Features

### 🔐 Authentication
- **GitHub OAuth**: FOSS implementation using `golang.org/x/oauth2`
- **Session Management**: Secure session handling with configurable expiration
- **Middleware Support**: HTTP middleware for protecting routes
- **User Context**: Easy access to authenticated user information

### 🔑 Secrets Management
- **Environment Variables**: First-class support for env vars (highest priority)
- **Encrypted File Storage**: AES-256-GCM encryption for local file storage
- **Multi-Store Support**: Unified interface for multiple secret stores
- **Future-Ready**: Hooks for Vault, sops, and Kubernetes secrets
- **Security**: Secrets are never logged or exposed in UI

### 🏗️ Infrastructure Orchestration
- **Proxmox Integration**: Complete VM/container orchestration via Proxmox VE API
- **Docker & Podman**: Full container lifecycle management with Compose support
- **Kubernetes Family**: Native support for kubectl, MicroK8s, K3s, and Talos
- **Nix/NixOS Support**: Reproducible infrastructure with Nix expressions and flakes
- **Multi-Step Workflows**: Complex infrastructure automation with dependency management
- **Unified Interface**: Single API for all infrastructure operations

### 🤖 New Agent Connectors

- **OpenCode Agent Connector**: Integrates OpenCode AI agent platform via REST/WebSocket APIs.
- **Agent-Zero Orchestrator Connector**: Integrates Agent-Zero orchestration via JSON-RPC/HTTP APIs.

- **PBKDF2 Key Derivation**: Strong password-based encryption
- **Secure Random Generation**: Cryptographically secure random values
- **HTTP-Only Cookies**: Session cookies with security flags
- **CORS Support**: Configurable cross-origin requests
- **Input Validation**: Proper validation and sanitization

## Quick Start

### 1. Environment Setup

```bash
# GitHub OAuth App (create at https://github.com/settings/applications/new)
export GITHUB_CLIENT_ID="your_github_client_id"
export GITHUB_CLIENT_SECRET="your_github_client_secret"
export GITHUB_REDIRECT_URL="http://localhost:8080/auth/callback"

# Optional: Custom secrets file
export AGENTIC_SECRETS_FILE=".agentic/secrets.json"
export AGENTIC_SECRETS_PASSWORD="your_secure_password"

# API Keys (optional)
export HUGGINGFACE_API_KEY="hf_your_api_key"
export IO_INTELLIGENCE_API_KEY="your_ioi_api_key"

# Infrastructure Credentials
export PROXMOX_URL="https://your-proxmox-server:8006"
export PROXMOX_USERNAME="your_proxmox_user@pam"
export PROXMOX_PASSWORD="your_proxmox_password"
export PROXMOX_TOKEN="your_proxmox_api_token"
export PROXMOX_NODE="your_default_proxmox_node"

# Docker Configuration
export DOCKER_HOST="unix:///var/run/docker.sock"
export DOCKER_CERT_PATH="/path/to/docker/certs"

# Kubernetes Configuration
export KUBECONFIG="/path/to/kubeconfig"
export KUBE_CONTEXT="your_kubernetes_context"
export KUBE_NAMESPACE="default"

# Nix/NixOS Configuration
export NIX_PATH="nixpkgs=/path/to/nixpkgs"
export NIX_SIGNING_KEY="/path/to/nix/signing.key"
export NIX_SUBSTITUTERS="https://cache.nixos.org/,https://your-cache.com"
export NIX_TRUSTED_KEYS="cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=,your-cache-key"
export NIX_REMOTE_HOSTS="user@host1:x86_64-linux,user@host2:aarch64-linux"
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "log"
    "os/signal"
    "syscall"

    "github.com/coder/coder/v2/agentic"
)

func main() {
    // Create server with auth and secrets
    server, err := agentic.NewExampleServer()
    if err != nil {
        log.Fatal(err)
    }

    // Setup graceful shutdown
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Start server
    if err := server.Start(ctx, ":8080"); err != nil {
        log.Fatal(err)
    }
}
```

### 3. API Endpoints

#### Authentication
- `GET /auth/login` - Initiate GitHub OAuth
- `GET /auth/callback` - OAuth callback handler
- `POST /auth/logout` - Logout and clear session

#### Protected APIs (require authentication)
- `GET /api/user` - Get current user info
- `GET /api/secrets` - List secret keys
- `POST /api/secrets` - Store a secret
- `DELETE /api/secrets/{key}` - Delete a secret
- `POST /api/tasks` - Create agentic task
- `GET /api/tasks/{id}` - Get task status

#### Public
- `GET /health` - Health check

## Configuration

### OpenCode and Agent-Zero Configuration

Set these environment variables or secrets for agent registration:

- `OPENCODE_API_KEY`: OpenCode API key
- `OPENCODE_ENDPOINT`: OpenCode REST endpoint (e.g. https://api.opencode.ai/v1/agent/invoke)
- `OPENCODE_WS_URL`: (optional) OpenCode WebSocket endpoint

- `AGENTZERO_ENDPOINT`: Agent-Zero JSON-RPC/HTTP endpoint (e.g. https://agentzero.local/api/jsonrpc)
- `AGENTZERO_API_KEY`: Agent-Zero API key

### Default Configuration

```go
config := agentic.DefaultConfig()
// Uses environment variables by default
// Falls back to encrypted file storage
```

### Custom Configuration

```go
config := &agentic.Config{
    OpenCode: agentic.OpenCodeConfig{
        APIKey:   "your_opencode_api_key",
        Endpoint: "https://api.opencode.ai/v1/agent/invoke",
        WSURL:    "wss://api.opencode.ai/v1/agent/ws",
    },
    AgentZero: agentic.AgentZeroConfig{
        Endpoint: "https://agentzero.local/api/jsonrpc",
        APIKey:   "your_agentzero_api_key",
    },
    // ... other config ...
}
```

## Secret Stores

### Environment Variables
```go
envStore := agentic.NewEnvSecretStore()
secretManager.AddStore("env", envStore)
```

### Encrypted File Storage
```go
fileStore := agentic.NewFileSecretStore(".agentic/secrets.json", "password")
secretManager.AddStore("file", fileStore)
```

### Multi-Store Priority
1. Environment variables (highest priority)
2. Encrypted file storage
3. Future: Vault, K8s secrets, sops

## Security Best Practices

### ✅ Implemented
- AES-256-GCM encryption for file storage
- PBKDF2 key derivation (4096 iterations)
- Cryptographically secure random generation
- Secrets never logged or exposed
- HTTP-only session cookies
- Secure cookie flags (Secure, SameSite)
- Input validation and sanitization

### 🔄 Recommended
- Use HTTPS in production
- Set strong `AGENTIC_SECRETS_PASSWORD`
- Rotate GitHub OAuth secrets regularly
- Use environment variables for sensitive config
- Enable audit logging
- Configure CORS appropriately

## Integration Examples

### Supported Agent Types

- `"opencode"`: OpenCode agent connector (LLM, plugin, IDE agent tasks)
- `"agent-zero"`: Agent-Zero orchestrator (multi-agent workflow, orchestration)

```go
type VaultSecretStore struct {
    client *vault.Client
}

func (v *VaultSecretStore) Get(key string) (string, error) {
    // Implement Vault integration
}

// Register with secret manager
secretManager.AddStore("vault", vaultStore)
```

### Custom Authentication
```go
// Get authenticated user in handlers
session, ok := agentic.GetUserFromContext(r.Context())
if !ok {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}

log.Printf("User %s accessed resource", session.Login)
```

## Future Enhancements

The architecture supports these planned integrations:

### Advanced Secret Stores
- **HashiCorp Vault**: Enterprise-grade secret management
- **Mozilla SOPS**: Encrypted YAML/JSON with age/PGP
- **Kubernetes Secrets**: Native K8s secret integration
- **Cloud Provider KMS**: AWS Secrets Manager, Azure Key Vault, GCP Secret Manager

### Enhanced Authentication
- **Multi-Provider OAuth**: GitLab, Bitbucket, etc.
- **OIDC Support**: Generic OpenID Connect providers
- **API Key Authentication**: Service-to-service auth
- **mTLS**: Mutual TLS for secure communications

## Architecture Compliance

This implementation fulfills **Steps 1 & 2** of the FOSS Agentic Orchestration Plan:

### ✅ Step 1: Authentication & Secrets Management
- **GitHub OAuth**: Using FOSS libraries (golang.org/x/oauth2)
- **Secure Secrets Storage**: Environment variables and encrypted local files
- **Advanced Secret Manager Hooks**: Architecture supports Vault, sops, K8s secrets
- **Security**: Secrets never logged or exposed in UI
- **FOSS Compliance**: No proprietary dependencies

### ✅ Step 2: Container, VM, and Cluster Integration
- **Proxmox Integration**: Terraform provider approach with native API support
- **Docker & Podman**: Full support for Compose, Swarm, Podman CLI/API, rootless containers
- **Kubernetes Family**: Upstream k8s, MicroK8s, K3s, and Talos integration
- **Agentic Tools**: CLI integration with unified orchestration interface
- **Secure Credential Storage**: All infrastructure credentials via encrypted secrets

### ✅ Step 3: Nix/NixOS Reproducible Infrastructure
- **Nix Package Management**: Complete nix-build, nix-shell, nix-env operations
- **Flakes Support**: Modern Nix workflow with flake references and reproducible builds
- **NixOS System Management**: nixos-rebuild for declarative system configurations
- **Remote Deployments**: Deploy configurations to remote NixOS machines via SSH
- **Store Operations**: Garbage collection, binary cache management, store signing
- **Development Environments**: nix develop for reproducible development shells
- **Infrastructure as Code**: Nix expressions for declarative infrastructure definitions
- **Secrets Integration**: Secure storage of Nix store signing keys and remote access credentials

## Testing

```bash
# Test secret management
go test ./agentic -run TestSecrets

# Test authentication
go test ./agentic -run TestAuth

# Integration tests
go test ./agentic -run TestIntegration
```

## Contributing

This is part of the larger FOSS Agentic Orchestration project.

### ✅ Completed Steps:
1. **Step 1**: GitHub OAuth and secure secrets management
2. **Step 2**: Container, VM, and cluster integration (Proxmox, Docker, K8s)
3. **Step 3**: Nix/NixOS support for reproducible infrastructure

### 🚧 Next Steps:
4. **Step 4**: GPU detection and management
5. **Step 5**: Multi-cloud connectors (Cloudflare, Vercel, Netlify)
6. **Step 6**: Classical reasoning, DePIN, and privacy modules
7. **Step 7**: Agentic orchestration layer (advanced task scheduling, registry, backend selection)
8. **Step 8**: UI dashboard and user controls

See `ARCHITECTURE_AGENTIC_FOSS_FINAL.md` for the complete roadmap.

## Infrastructure Examples

### Run Infrastructure Demo
```bash
# Run the complete infrastructure demonstration
go run ./agentic/example_infrastructure.go

# Or run as infrastructure server
go run ./agentic/example_infrastructure.go infrastructure
```

### Proxmox Operations
```go
// List VMs on Proxmox
task := &agentic.Task{
    Type: "vm",
    Payload: map[string]interface{}{
        "action": "list",
        "node": "pve",
    },
}
```

### Docker Operations
```go
// Run a container
task := &agentic.Task{
    Type: "container",
    Payload: map[string]interface{}{
        "action": "run",
        "image": "nginx:alpine",
        "name": "web-server",
        "ports": []string{"8080:80"},
        "detach": true,
    },
}
```

### Kubernetes Operations
```go
// Deploy a manifest
task := &agentic.Task{
	Type: "kubernetes",
	Payload: map[string]interface{}{
		"action": "apply",
		"manifest": yamlManifest,
		"namespace": "default",
	},
}
```

### Nix/NixOS Operations
```go
// Build a Nix expression
task := &agentic.Task{
	Type: "nix",
	Payload: map[string]interface{}{
		"action": "build",
		"flake_ref": "github:nixos/nixpkgs#hello",
		"system": "x86_64-linux",
	},
}

// Deploy NixOS configuration
task := &agentic.Task{
	Type: "nixos",
	Payload: map[string]interface{}{
		"action": "rebuild",
		"flake_ref": "github:myorg/nixos-config#myhost",
		"config": map[string]interface{}{
			"rebuild_action": "switch",
		},
	},
}

// Remote deployment
task := &agentic.Task{
	Type: "nix",
	Payload: map[string]interface{}{
		"action": "deploy",
		"flake_ref": "github:myorg/infra#production",
		"remote": map[string]interface{}{
			"host": "prod-server.example.com",
			"user": "deploy",
			"ssh_key": "/path/to/deploy.key",
		},
	},
}
```
