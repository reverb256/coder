# OpenCode & Agent-Zero Integration: Deployment Guide

## Prerequisites

- **Operating System:** Linux (recommended), macOS, Windows (with Docker)
- **Dependencies:** Go 1.20+, Docker, Node.js (for UI), PostgreSQL
- **Network:** Outbound HTTPS to OpenCode and Agent-Zero endpoints
- **API Keys:** Obtain valid API keys for OpenCode and Agent-Zero
- **Permissions:** Admin access for deployment and configuration

## Installation Steps

1. **Clone the Repository**
   ```bash
   git clone https://github.com/your-org/your-repo.git
   cd your-repo
   ```

2. **Build Backend Services**
   ```bash
   make build
   ```

3. **Install Dependencies for UI**
   ```bash
   cd site
   npm install
   npm run build
   cd ..
   ```

4. **Configure Environment Variables**
   - Copy and edit `.env.example` in `agentic/` and backend directories.
   - Set OpenCode and Agent-Zero API keys and endpoints.

5. **Set Up Database**
   ```bash
   createdb coder
   # Apply migrations as needed
   ```

6. **Deploy Workspace Templates**
   - Copy files from [`examples/templates/opencode-agentzero/`](examples/templates/opencode-agentzero/) to your workspace templates directory.
   - Edit `workspace.yaml` as needed for your environment.

7. **Start Services**
   ```bash
   docker-compose up -d
   # or use your preferred orchestration method
   ```

## Configuration

- **OpenCode Connector:** Configure `agentic/connector_opencode.go` with your API key and endpoint.
- **Agent-Zero Connector:** Configure `agentic/connector_agentzero.go` with your API key and endpoint.
- **Workspace Templates:** Edit `workspace.yaml`, `opencode-service.sh`, and `agentzero-service.sh` for your environment.
- **Networking:** See `networking.md` in the template directory for security and connectivity best practices.

## Environment Setup & Validation

- **Health Checks:** Access `/api/v2/healthz` to verify backend health.
- **Integration Tests:** Run provided test scripts or CI pipeline to validate integration.
- **UI Validation:** Log in to the dashboard and verify agent and workflow management features.

## Troubleshooting

| Issue                          | Solution                                                      |
|---------------------------------|---------------------------------------------------------------|
| API connection errors           | Check API keys, endpoints, and network connectivity           |
| Workspace template not loading  | Verify `workspace.yaml` syntax and paths                      |
| Agent not responding            | Check logs for connector errors, validate API credentials     |
| UI not showing agents/workflows | Ensure backend is running and API endpoints are reachable     |
| Database errors                 | Confirm PostgreSQL is running and migrations are applied      |

---

For advanced configuration and maintenance, see the [Administrator Guide](admin-guide.md).
