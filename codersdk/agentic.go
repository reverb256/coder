// Package codersdk provides API types for agentic endpoints.

package codersdk

// OpenCodeAgent represents an OpenCode agent.
type OpenCodeAgent struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Config interface{} `json:"config,omitempty"`
}

// ListOpenCodeAgentsResponse is the response for listing OpenCode agents.
type ListOpenCodeAgentsResponse struct {
	Agents []OpenCodeAgent `json:"agents"`
}

// InvokeOpenCodeAgentRequest is the request to invoke an OpenCode agent.
type InvokeOpenCodeAgentRequest struct {
	AgentID string      `json:"agent_id"`
	Task    interface{} `json:"task"`
}

// InvokeOpenCodeAgentResponse is the response from invoking an OpenCode agent.
type InvokeOpenCodeAgentResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// AgentZeroWorkflow represents an Agent-Zero workflow.
type AgentZeroWorkflow struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Steps       interface{} `json:"steps"`
}

// ListAgentZeroWorkflowsResponse is the response for listing Agent-Zero workflows.
type ListAgentZeroWorkflowsResponse struct {
	Workflows []AgentZeroWorkflow `json:"workflows"`
}

// OrchestrateAgentZeroRequest is the request to execute an Agent-Zero orchestrated task.
type OrchestrateAgentZeroRequest struct {
	WorkflowID string      `json:"workflow_id"`
	Input      interface{} `json:"input"`
}

// OrchestrateAgentZeroResponse is the response from executing an Agent-Zero orchestrated task.
type OrchestrateAgentZeroResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}
