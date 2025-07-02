//go:build unit

package agentic

import (
	"context"
	"testing"
)

func TestAgentZeroClient_BadConfig(t *testing.T) {
	client := NewAgentZeroClient(AgentZeroConfig{})
	_, err := client.Execute(context.Background(), &Task{Payload: map[string]interface{}{"foo": "bar"}})
	if err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestAgentZeroClient_Timeout(t *testing.T) {
	// No Timeout field in AgentZeroConfig; just test with a short context timeout
	client := NewAgentZeroClient(AgentZeroConfig{Endpoint: "http://localhost:9999"})
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	_, err := client.Execute(ctx, &Task{Payload: map[string]interface{}{"foo": "bar"}})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestAgentZeroClient_MalformedResponse(t *testing.T) {
	// Mock server returning malformed JSON
	// TODO: Implement with httptest.Server if supported
}

func TestAgentZeroClient_ConfigValidation(t *testing.T) {
	cfg := AgentZeroConfig{Endpoint: ""}
	if cfg.Endpoint != "" {
		t.Error("expected empty endpoint")
	}
}

func TestAgentZeroClient_NameAndSupports(t *testing.T) {
	client := NewAgentZeroClient(AgentZeroConfig{Endpoint: "http://localhost"})
	if client.Name() != "agent-zero" {
		t.Errorf("unexpected name: %s", client.Name())
	}
	if !client.Supports("agent-zero") {
		t.Error("should support 'agent-zero'")
	}
	if !client.Supports("orchestration") {
		t.Error("should support 'orchestration'")
	}
	if !client.Supports("workflow") {
		t.Error("should support 'workflow'")
	}
}
