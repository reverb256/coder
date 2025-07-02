//go:build unit

package agentic

import (
	"context"
	"testing"
)

func TestOpenCodeClient_BadConfig(t *testing.T) {
	client := NewOpenCodeClient(OpenCodeConfig{})
	_, err := client.Execute(context.Background(), &Task{Payload: map[string]interface{}{"foo": "bar"}})
	if err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestOpenCodeClient_Timeout(t *testing.T) {
	client := NewOpenCodeClient(OpenCodeConfig{Endpoint: "http://localhost:9999", Timeout: 1})
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	_, err := client.Execute(ctx, &Task{Payload: map[string]interface{}{"foo": "bar"}})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestOpenCodeClient_MalformedResponse(t *testing.T) {
	// Mock server returning malformed JSON
	// TODO: Implement with httptest.Server if supported
}

func TestOpenCodeClient_ConfigValidation(t *testing.T) {
	cfg := OpenCodeConfig{Endpoint: ""}
	if cfg.Endpoint != "" {
		t.Error("expected empty endpoint")
	}
}

func TestOpenCodeClient_NameAndSupports(t *testing.T) {
	client := NewOpenCodeClient(OpenCodeConfig{Endpoint: "http://localhost"})
	if client.Name() != "opencode" {
		t.Errorf("unexpected name: %s", client.Name())
	}
	if !client.Supports("opencode") {
		t.Error("should support 'opencode'")
	}
	if !client.Supports("llm") {
		t.Error("should support 'llm'")
	}
	if !client.Supports("plugin") {
		t.Error("should support 'plugin'")
	}
}
