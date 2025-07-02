// Package agentic provides a lightweight agent/task system for agentic tools and LLM connectors.
package agentic

import (
	"context"
	"sync"

	"golang.org/x/xerrors"
)

// Agent defines the interface for agentic tools (LLM, embedding, etc).
type Agent interface {
	Name() string
	Supports(taskType string) bool
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

// Registry manages agent registration and selection.
type Registry struct {
	mu     sync.RWMutex
	agents []Agent
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents = append(r.agents, agent)
}

func (r *Registry) Select(taskType string) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.agents {
		if a.Supports(taskType) {
			return a, nil
		}
	}
	return nil, xerrors.New("no agent supports task type: " + taskType)
}
