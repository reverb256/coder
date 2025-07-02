// Package agentic provides an Agent-Zero orchestrator connector.
package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// AgentZeroConfig holds Agent-Zero API config.
type AgentZeroConfig struct {
	Endpoint string // JSON-RPC/HTTP endpoint, e.g. https://agentzero.local/api/jsonrpc
	APIKey   string
}

// AgentZeroClient implements the Agent interface for Agent-Zero orchestration.
type AgentZeroClient struct {
	cfg    AgentZeroConfig
	memory sync.Map // simple in-memory state for demonstration
}

func NewAgentZeroClient(cfg AgentZeroConfig) *AgentZeroClient {
	return &AgentZeroClient{cfg: cfg}
}

func (a *AgentZeroClient) Name() string { return "agent-zero" }

func (a *AgentZeroClient) Supports(taskType string) bool {
	return taskType == "agent-zero" || taskType == "orchestration" || taskType == "workflow"
}

func (a *AgentZeroClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	endpoint := a.cfg.Endpoint
	if endpoint == "" {
		return nil, errors.New("Agent-Zero endpoint not configured")
	}
	// JSON-RPC 2.0 request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "orchestrate",
		"params":  task.Payload,
		"id":      time.Now().UnixNano(),
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[agent-zero] marshal error: %v", err)
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		log.Printf("[agent-zero] request error: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[agent-zero] http error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("[agent-zero] API error: %s", string(body))
		return &TaskResult{Error: errors.New(string(body))}, nil
	}
	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  interface{} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return &TaskResult{Output: string(body)}, nil
	}
	if rpcResp.Error != nil {
		return &TaskResult{Error: errors.New("Agent-Zero error")}, nil
	}
	return &TaskResult{Output: rpcResp.Result}, nil
}
