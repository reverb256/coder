# OpenCode & Agent-Zero Integration: User Guide

## Introduction

This guide explains how to use OpenCode agents and Agent-Zero workflows within your Coder workspaces.

---

## Using OpenCode Agents

1. **Access Your Workspace**
   - Log in to the Coder dashboard.
   - Open a workspace configured with the OpenCode agent.

2. **Invoke OpenCode Agent**
   - Use the UI or API to send tasks to the OpenCode agent.
   - Example (API):
     ```bash
     curl -X POST https://your-coder-instance/api/v2/agentic/opencode/invoke \
       -H "Authorization: Bearer <API_KEY>" \
       -d '{"agent_id":"opencode","task":{"prompt":"Generate README.md"}}'
     ```
   - Results will be displayed in the UI or returned via API.

3. **Supported Tasks**
   - Code generation, LLM tasks, plugin execution, and more.

---

## Creating and Managing Agent-Zero Workflows

1. **Access Workflow Management**
   - Navigate to the Agent-Zero section in the dashboard.

2. **Create a Workflow**
   - Define workflow steps and parameters in the UI or via API.
   - Example (API):
     ```bash
     curl -X POST https://your-coder-instance/api/v2/agentic/agent-zero/workflows \
       -H "Authorization: Bearer <API_KEY>" \
       -d '{"name":"CI Pipeline","steps":[{"type":"build"},{"type":"test"}]}'
     ```

3. **Run a Workflow**
   - Select a workflow and provide input parameters.
   - Monitor execution status and results in the UI.

---

## UI Walkthrough

- **Dashboard:** View all workspaces, agents, and workflows.
- **Workspace Management:** Create, edit, and delete workspaces with integrated agents.
- **Agent Interaction:** Send tasks, view results, and manage agent settings.
- **Workflow Management:** Create, edit, run, and monitor workflows.

---

## Best Practices

- **Security:** Never share API keys. Use role-based access controls.
- **Performance:** Allocate sufficient resources for agent workloads.
- **Troubleshooting:** Use logs and health checks for diagnostics.
- **Template Customization:** Adapt workspace templates for your specific needs.

---

For advanced usage and administration, see the [Administrator Guide](admin-guide.md).
