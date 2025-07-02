// Package agentic provides a HuggingFace connector agent.
package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

// HFConfig holds HuggingFace API config.
type HFConfig struct {
	APIKey   string
	Model    string
	Endpoint string // Optional, defaults to https://api-inference.huggingface.co/models/
}

// HFClient is an agent for HuggingFace LLM/embedding tasks.
type HFClient struct {
	cfg HFConfig
}

func NewHFClient(cfg HFConfig) *HFClient {
	return &HFClient{cfg: cfg}
}

func (h *HFClient) Name() string { return "huggingface" }

func (h *HFClient) Supports(taskType string) bool {
	return taskType == "llm" || taskType == "embedding"
}

func (h *HFClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	endpoint := h.cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api-inference.huggingface.co/models/" + h.cfg.Model
	}
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
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
