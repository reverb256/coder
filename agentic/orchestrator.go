// Package agentic provides orchestration for multiple infrastructure connectors.
package agentic

import (
	"context"
	"fmt"

	"golang.org/x/xerrors"
)

// Orchestrator manages multiple infrastructure agents and provides unified task execution.
type Orchestrator struct {
	registry      *Registry
	secretManager *SecretManager
	config        *Config
}

// NewOrchestrator creates a new orchestrator with all available connectors.
func NewOrchestrator(config *Config, secretManager *SecretManager) (*Orchestrator, error) {
	registry := NewRegistry()

	// Register LLM/AI connectors
	if config.HuggingFace.APIKey != "" {
		hfClient := NewHFClient(config.HuggingFace)
		registry.Register(hfClient)
	}

	if config.IOIntelligence.APIKey != "" {
		ioiClient := NewIOIClient(config.IOIntelligence)
		registry.Register(ioiClient)
	}

	// Register infrastructure connectors

	// Proxmox connector
	if config.Infrastructure.Proxmox.URL != "" {
		proxmoxClient := NewProxmoxClient(config.Infrastructure.Proxmox)
		registry.Register(proxmoxClient)
	}

	// Docker connector
	dockerClient := NewDockerClient(config.Infrastructure.Docker)
	registry.Register(dockerClient)

	// Kubernetes connector
	kubernetesClient := NewKubernetesClient(config.Infrastructure.Kubernetes)
	registry.Register(kubernetesClient)

	// Nix connector
	nixClient := NewNixClient(config.Infrastructure.Nix)
	registry.Register(nixClient)

	// GPU connector
	gpuClient := NewGPUClient(config.Infrastructure.GPU)
	registry.Register(gpuClient)

	return &Orchestrator{
		registry:      registry,
		secretManager: secretManager,
		config:        config,
	}, nil
}

// ExecuteTask executes a task using the appropriate connector.
func (o *Orchestrator) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
	agent, err := o.registry.Select(task.Type)
	if err != nil {
		return nil, xerrors.Errorf("no agent available for task type '%s': %w", task.Type, err)
	}

	return agent.Execute(ctx, task)
}

// ListConnectors returns information about available connectors.
func (o *Orchestrator) ListConnectors() []ConnectorInfo {
	var connectors []ConnectorInfo

	// Check each registered agent
	for _, agent := range o.registry.agents {
		connector := ConnectorInfo{
			Name:           agent.Name(),
			Type:           getConnectorType(agent),
			Status:         "available",
			SupportedTasks: getSupportedTasks(agent),
		}
		connectors = append(connectors, connector)
	}

	return connectors
}

// ConnectorInfo provides information about a connector.
type ConnectorInfo struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`   // "llm", "infrastructure", "cloud"
	Status         string   `json:"status"` // "available", "configured", "error"
	SupportedTasks []string `json:"supported_tasks"`
}

// getConnectorType determines the type of a connector.
func getConnectorType(agent Agent) string {
	switch agent.Name() {
	case "huggingface", "io_intelligence":
		return "llm"
	case "proxmox", "docker", "kubernetes", "nix", "gpu":
		return "infrastructure"
	default:
		return "unknown"
	}
}

// getSupportedTasks returns the task types supported by an agent.
func getSupportedTasks(agent Agent) []string {
	var tasks []string

	// Test common task types
	testTasks := []string{
		"llm", "embedding",
		"vm", "container", "infrastructure",
		"kubernetes", "k8s", "cluster",
		"docker", "podman",
		"nix", "nixos", "flake", "reproducible",
		"gpu", "nvidia", "cuda", "hardware",
	}

	for _, taskType := range testTasks {
		if agent.Supports(taskType) {
			tasks = append(tasks, taskType)
		}
	}

	return tasks
}

// ExecuteInfrastructureWorkflow executes a complex infrastructure workflow.
func (o *Orchestrator) ExecuteInfrastructureWorkflow(ctx context.Context, workflow *InfrastructureWorkflow) (*WorkflowResult, error) {
	result := &WorkflowResult{
		WorkflowID: workflow.ID,
		Steps:      make([]StepResult, 0, len(workflow.Steps)),
	}

	for i, step := range workflow.Steps {
		stepResult := StepResult{
			StepID: step.ID,
			Name:   step.Name,
			Order:  i + 1,
		}

		// Execute the step
		task := &Task{
			Type:    step.TaskType,
			Payload: step.Parameters,
		}

		taskResult, err := o.ExecuteTask(ctx, task)
		if err != nil {
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
			result.Steps = append(result.Steps, stepResult)
			result.Status = "failed"
			return result, xerrors.Errorf("workflow step %d failed: %w", i+1, err)
		}

		stepResult.Status = "completed"
		stepResult.Output = taskResult.Output
		result.Steps = append(result.Steps, stepResult)

		// Check if we should continue on error
		if taskResult.Error != nil && !step.ContinueOnError {
			result.Status = "failed"
			return result, xerrors.Errorf("workflow step %d had error: %w", i+1, taskResult.Error)
		}
	}

	result.Status = "completed"
	return result, nil
}

// InfrastructureWorkflow represents a complex infrastructure automation workflow.
type InfrastructureWorkflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []WorkflowStep `json:"steps"`
}

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	TaskType        string                 `json:"task_type"` // vm, container, kubernetes, etc.
	Parameters      map[string]interface{} `json:"parameters"`
	DependsOn       []string               `json:"depends_on"` // Step IDs this step depends on
	ContinueOnError bool                   `json:"continue_on_error"`
}

// WorkflowResult represents the result of a workflow execution.
type WorkflowResult struct {
	WorkflowID string       `json:"workflow_id"`
	Status     string       `json:"status"` // "running", "completed", "failed"
	Steps      []StepResult `json:"steps"`
}

// StepResult represents the result of a single workflow step.
type StepResult struct {
	StepID string      `json:"step_id"`
	Name   string      `json:"name"`
	Order  int         `json:"order"`
	Status string      `json:"status"` // "pending", "running", "completed", "failed"
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// CreateVMWorkflow creates a workflow for provisioning a complete VM infrastructure.
func CreateVMWorkflow(vmConfig VMWorkflowConfig) *InfrastructureWorkflow {
	workflow := &InfrastructureWorkflow{
		ID:          fmt.Sprintf("vm-provision-%s", vmConfig.Name),
		Name:        fmt.Sprintf("Provision VM: %s", vmConfig.Name),
		Description: "Complete VM provisioning workflow with container setup",
		Steps:       []WorkflowStep{},
	}

	// Step 1: Create VM in Proxmox
	if vmConfig.UseProxmox {
		workflow.Steps = append(workflow.Steps, WorkflowStep{
			ID:       "create-vm",
			Name:     "Create Virtual Machine",
			TaskType: "vm",
			Parameters: map[string]interface{}{
				"action":  "create",
				"vm_type": "qemu",
				"node":    vmConfig.ProxmoxNode,
				"config": map[string]interface{}{
					"cores":  vmConfig.CPU,
					"memory": vmConfig.Memory,
					"scsihw": "virtio-scsi-pci",
					"net0":   "virtio,bridge=vmbr0",
				},
			},
		})
	}

	// Step 2: Start VM
	if vmConfig.UseProxmox {
		workflow.Steps = append(workflow.Steps, WorkflowStep{
			ID:        "start-vm",
			Name:      "Start Virtual Machine",
			TaskType:  "vm",
			DependsOn: []string{"create-vm"},
			Parameters: map[string]interface{}{
				"action":  "start",
				"vm_type": "qemu",
				"node":    vmConfig.ProxmoxNode,
				"vmid":    vmConfig.VMID,
			},
		})
	}

	// Step 3: Setup Docker containers
	if len(vmConfig.Containers) > 0 {
		for i, container := range vmConfig.Containers {
			stepID := fmt.Sprintf("deploy-container-%d", i+1)
			dependsOn := []string{}
			if vmConfig.UseProxmox {
				dependsOn = []string{"start-vm"}
			}

			workflow.Steps = append(workflow.Steps, WorkflowStep{
				ID:        stepID,
				Name:      fmt.Sprintf("Deploy Container: %s", container.Name),
				TaskType:  "container",
				DependsOn: dependsOn,
				Parameters: map[string]interface{}{
					"action":  "run",
					"image":   container.Image,
					"name":    container.Name,
					"ports":   container.Ports,
					"volumes": container.Volumes,
					"env":     container.Environment,
					"detach":  true,
				},
			})
		}
	}

	// Step 4: Deploy Kubernetes resources
	if len(vmConfig.KubernetesManifests) > 0 {
		for i, manifest := range vmConfig.KubernetesManifests {
			stepID := fmt.Sprintf("deploy-k8s-%d", i+1)

			workflow.Steps = append(workflow.Steps, WorkflowStep{
				ID:       stepID,
				Name:     fmt.Sprintf("Deploy K8s: %s", manifest.Name),
				TaskType: "kubernetes",
				Parameters: map[string]interface{}{
					"action":    "apply",
					"manifest":  manifest.YAML,
					"namespace": manifest.Namespace,
				},
			})
		}
	}

	return workflow
}

// VMWorkflowConfig represents configuration for a VM provisioning workflow.
type VMWorkflowConfig struct {
	Name        string `json:"name"`
	VMID        int    `json:"vmid"`
	CPU         int    `json:"cpu"`
	Memory      int    `json:"memory"`
	UseProxmox  bool   `json:"use_proxmox"`
	ProxmoxNode string `json:"proxmox_node"`

	// Container configuration
	Containers []ContainerConfig `json:"containers"`

	// Kubernetes configuration
	KubernetesManifests []K8sManifestConfig `json:"kubernetes_manifests"`
}

// ContainerConfig represents a container to deploy.
type ContainerConfig struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Ports       []string          `json:"ports"`
	Volumes     []string          `json:"volumes"`
	Environment map[string]string `json:"environment"`
}

// K8sManifestConfig represents a Kubernetes manifest to deploy.
type K8sManifestConfig struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	YAML      string `json:"yaml"`
}
