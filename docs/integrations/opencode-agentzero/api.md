# OpenCode & Agent-Zero Integration: API Documentation

## Overview

This document provides a reference for all API endpoints related to OpenCode and Agent-Zero integration.

---

## Authentication

- **API Key:** Pass as `Authorization: Bearer <API_KEY>` header.
- **Session Token:** Use `Coder-Session-Token` header if applicable.

---

## Endpoints

### OpenCode Agent Endpoints

- **List Agents**
  - `GET /api/v2/agentic/opencode/agents`
  - **Response:**
    ```json
    {
      "agents": [
        {
          "id": "opencode",
          "name": "opencode",
          "status": "active",
          "config": {}
        }
      ]
    }
    ```

- **Invoke Agent**
  - `POST /api/v2/agentic/opencode/invoke`
  - **Request:**
    ```json
    {
      "agent_id": "opencode",
      "task": { "prompt": "Generate README.md" }
    }
    ```
  - **Response:**
    ```json
    {
      "result": { "output": "README.md content..." },
      "error": ""
    }
    ```

### Agent-Zero Workflow Endpoints

- **List Workflows**
  - `GET /api/v2/agentic/agent-zero/workflows`
  - **Response:**
    ```json
    {
      "workflows": [
        {
          "id": "wf-123",
          "name": "CI Pipeline",
          "description": "Build and test workflow",
          "steps": [ ... ]
        }
      ]
    }
    ```

- **Create Workflow**
  - `POST /api/v2/agentic/agent-zero/workflows`
  - **Request:**
    ```json
    {
      "name": "CI Pipeline",
      "steps": [ { "type": "build" }, { "type": "test" } ]
    }
    ```

- **Orchestrate Workflow**
  - `POST /api/v2/agentic/agent-zero/orchestrate`
  - **Request:**
    ```json
    {
      "workflow_id": "wf-123",
      "input": { "repo": "my-repo" }
    }
    ```
  - **Response:**
    ```json
    {
      "result": { "status": "success" },
      "error": ""
    }
    ```

---

## Error Codes

| Code | Meaning                        | Troubleshooting                        |
|------|--------------------------------|----------------------------------------|
| 400  | Bad Request                    | Check request payload and parameters   |
| 401  | Unauthorized                   | Verify API key or session token        |
| 404  | Not Found                      | Check endpoint path and resource IDs   |
| 500  | Internal Server Error           | Check logs and backend service health  |

---

## SDK Usage Example (Go)

```go
import "github.com/coder/coder/v2/codersdk/agentic"

client := agentic.NewOpenCodeClient(agentic.OpenCodeConfig{
    APIKey:   "your-opencode-key",
    Endpoint: "https://api.opencode.ai/v1/agent/invoke",
})

result, err := client.Execute(ctx, &agentic.Task{
    Type:    "opencode",
    Payload: map[string]interface{}{"prompt": "Generate README.md"},
})
if err != nil {
    // handle error
}
fmt.Println(result.Output)
```

---

For further details, see the [User Guide](user-guide.md) and [Administrator Guide](admin-guide.md).
