// Package agentic provides an OpenCode agent connector.
package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

// OpenCodeConfig holds OpenCode API config.
type OpenCodeConfig struct {
	APIKey   string
	Endpoint string // REST endpoint, e.g. https://api.opencode.ai/v1/agent/invoke
	WSURL    string // WebSocket endpoint, optional
}

// OpenCodeClient implements the Agent interface for OpenCode.
type OpenCodeClient struct {
	cfg OpenCodeConfig
}

func NewOpenCodeClient(cfg OpenCodeConfig) *OpenCodeClient {
	return &OpenCodeClient{cfg: cfg}
}

func (o *OpenCodeClient) Name() string { return "opencode" }

func (o *OpenCodeClient) Supports(taskType string) bool {
	return taskType == "opencode" || taskType == "llm" || taskType == "plugin"
}

func (o *OpenCodeClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	endpoint := o.cfg.Endpoint
	if endpoint == "" {
		return nil, errors.New("OpenCode endpoint not configured")
	}
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		log.Printf("[opencode] marshal error: %v", err)
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		log.Printf("[opencode] request error: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[opencode] http error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("[opencode] API error: %s", string(body))
		return &TaskResult{Error: errors.New(string(body))}, nil
	}
	var out interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return &TaskResult{Output: string(body)}, nil
	}
	return &TaskResult{Output: out}, nil
}
