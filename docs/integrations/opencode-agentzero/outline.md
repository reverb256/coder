# OpenCode & Agent-Zero Integration Documentation Plan

## 1. Implementation Summary

- **Overview:** Summarizes all 5 phases, referencing key files:
  - [`agentic/connector_opencode.go`](agentic/connector_opencode.go)
  - [`agentic/connector_agentzero.go`](agentic/connector_agentzero.go)
  - [`coderd/coderd.go`](coderd/coderd.go)
  - [`codersdk/agentic.go`](codersdk/agentic.go)
  - [`examples/templates/opencode-agentzero/`](examples/templates/opencode-agentzero/)
- **Architecture:** System diagram (see below).
- **Integration Points:** List and describe all integration touchpoints.
- **Feature Matrix:** Table of delivered capabilities.
- **System Requirements:** OS, dependencies, network, and API keys.

## 2. Deployment Guide

- **Prerequisites:** Hardware, OS, network, API keys, and external service access.
- **Installation:** Step-by-step for backend, connectors, and workspace templates.
- **Configuration:** How to set up endpoints, secrets, and environment variables.
- **Validation:** How to verify deployment (health checks, test scripts).
- **Troubleshooting:** Common issues and solutions.

## 3. User Guide

- **Using OpenCode Agents:** How to invoke and interact with agents in workspaces.
- **Agent-Zero Workflows:** Creating, managing, and running workflows.
- **UI Walkthrough:** Screenshots and explanations of dashboard, workspace management, and agent interaction.
- **Best Practices:** Usage patterns, security, and performance tips.

## 4. Administrator Guide

- **System Configuration:** Managing deployment, scaling, and upgrades.
- **Security:** API key management, network security, and audit logging.
- **Monitoring:** Metrics, logs, and alerting.
- **Maintenance:** Backup, restore, and update procedures.
- **Performance Tuning:** Configuration options for optimization.

## 5. API Documentation

- **Endpoints:** Document `/api/v2/agentic/opencode` and `/api/v2/agentic/agent-zero` endpoints with request/response examples.
- **Authentication:** API key and session token usage.
- **Error Codes:** List and explain error responses.
- **SDK Usage:** Example code for Go SDK ([`codersdk/agentic.go`](codersdk/agentic.go)).

---

## Example Mermaid Architecture Diagram

```mermaid
flowchart TD
    UserUI[User Interface<br/>(Dashboard, Workspace Mgmt)]
    API[API Layer<br/>coderd/coderd.go]
    OpenCode[OpenCode Connector<br/>agentic/connector_opencode.go]
    AgentZero[Agent-Zero Connector<br/>agentic/connector_agentzero.go]
    Templates[Workspace Templates<br/>examples/templates/opencode-agentzero/]
    SDK[API Types/SDK<br/>codersdk/agentic.go]
    CI[Testing/CI Framework]
    UserUI -- REST/WS --> API
    API -- Agentic API --> OpenCode
    API -- Agentic API --> AgentZero
    API -- Uses --> Templates
    API -- Uses --> SDK
    CI -- Validates --> API
```

---

## Documentation File Organization

- `docs/integrations/opencode-agentzero/overview.md`
- `docs/integrations/opencode-agentzero/deployment.md`
- `docs/integrations/opencode-agentzero/user-guide.md`
- `docs/integrations/opencode-agentzero/admin-guide.md`
- `docs/integrations/opencode-agentzero/api.md`
- Images/screenshots in `docs/integrations/opencode-agentzero/images/`
