// Package agentic provides task/job definitions and a background scheduler.
package agentic

import (
	"context"
	"sync"
)

// Task represents a unit of work for an agent.
type Task struct {
	Type    string
	Payload map[string]interface{}
	Status  string
	Result  *TaskResult
}

// TaskResult holds the result of a completed task.
type TaskResult struct {
	Output interface{}
	Error  error
}

// Scheduler manages task queueing and background execution.
type Scheduler struct {
	queue    chan *Task
	registry *Registry
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewScheduler creates a new Scheduler.
func NewScheduler(registry *Registry, queueSize int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		queue:    make(chan *Task, queueSize),
		registry: registry,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Schedule adds a task to the queue.
func (s *Scheduler) Schedule(task *Task) {
	s.queue <- task
}

// Run starts background workers.
func (s *Scheduler) Run(workers int) {
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

// Stop signals all workers to stop and waits for them.
func (s *Scheduler) Stop() {
	s.cancel()
	close(s.queue)
	s.wg.Wait()
}

func (s *Scheduler) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case task, ok := <-s.queue:
			if !ok {
				return
			}
			s.handleTask(task)
		}
	}
}

func (s *Scheduler) handleTask(task *Task) {
	agent, err := s.registry.Select(task.Type)
	if err != nil {
		task.Status = "failed"
		task.Result = &TaskResult{Error: err}
		return
	}
	task.Status = "running"
	res, err := agent.Execute(s.ctx, task)
	task.Status = "done"
	task.Result = res
	if err != nil {
		task.Result.Error = err
	}
}
