// Package agentic provides an IO Intelligence connector agent.
package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

// IOIConfig holds IO Intelligence API config.
type IOIConfig struct {
	APIKey   string
	Model    string
	Endpoint string // Optional, defaults to https://api.intelligence.io/v1/llm/
}

// IOIClient is an agent for IO Intelligence LLM/embedding tasks.
type IOIClient struct {
	cfg IOIConfig
}

func NewIOIClient(cfg IOIConfig) *IOIClient {
	return &IOIClient{cfg: cfg}
}

func (i *IOIClient) Name() string { return "io_intelligence" }

func (i *IOIClient) Supports(taskType string) bool {
	return taskType == "llm" || taskType == "embedding"
}

func (i *IOIClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	endpoint := i.cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.intelligence.io/v1/llm/" + i.cfg.Model
	}
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+i.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return &TaskResult{Error: xerrors.New(string(body))}, nil
	}
	var out interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return &TaskResult{Output: string(body)}, nil
	}
	return &TaskResult{Output: out}, nil
}
