package codersdk

import (
	"encoding/json"
	"testing"
)

func TestOpenCodeAgent_MarshalUnmarshal(t *testing.T) {
	agent := OpenCodeAgent{ID: "1", Name: "test", Status: "active"}
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatal(err)
	}
	var out OpenCodeAgent
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != agent.ID || out.Name != agent.Name || out.Status != agent.Status {
		t.Error("marshal/unmarshal mismatch")
	}
}

func TestAgentZeroWorkflow_MarshalUnmarshal(t *testing.T) {
	wf := AgentZeroWorkflow{ID: "wf1", Name: "wf", Description: "desc"}
	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatal(err)
	}
	var out AgentZeroWorkflow
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != wf.ID || out.Name != wf.Name {
		t.Error("marshal/unmarshal mismatch")
	}
}

func TestInvokeOpenCodeAgentResponse_ErrorField(t *testing.T) {
	resp := InvokeOpenCodeAgentResponse{Result: nil, Error: "fail"}
	if resp.Error != "fail" {
		t.Error("error field not set")
	}
}

func TestOrchestrateAgentZeroResponse_ErrorField(t *testing.T) {
	resp := OrchestrateAgentZeroResponse{Result: nil, Error: "fail"}
	if resp.Error != "fail" {
		t.Error("error field not set")
	}
}
