// Package agentic provides examples for GPU connector usage.
package agentic

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleGPUDetection demonstrates basic GPU detection and information gathering.
func ExampleGPUDetection() {
	ctx := context.Background()

	// Create GPU client with default configuration
	gpuConfig := GPUConfig{
		NvidiaSMIPath:    "nvidia-smi",
		DockerRuntime:    "nvidia",
		EnableMonitoring: true,
		MaxGPUsPerTask:   4,
	}

	gpuClient := NewGPUClient(gpuConfig)

	// List available GPUs
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action": "list",
		},
	}

	result, err := gpuClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error listing GPUs: %v", err)
		return
	}

	fmt.Printf("GPU Detection Result: %+v\n", result.Output)
}

// ExampleGPUInfo demonstrates detailed GPU information retrieval.
func ExampleGPUInfo() {
	ctx := context.Background()

	gpuClient := NewGPUClient(GPUConfig{})

	// Get detailed information for specific GPUs
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action":  "info",
			"gpu_ids": []int{0, 1}, // Target specific GPUs
		},
	}

	result, err := gpuClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error getting GPU info: %v", err)
		return
	}

	fmt.Printf("GPU Info Result: %+v\n", result.Output)
}

// ExampleGPUMonitoring demonstrates GPU monitoring over time.
func ExampleGPUMonitoring() {
	ctx := context.Background()

	gpuClient := NewGPUClient(GPUConfig{})

	// Monitor GPUs for 30 seconds with 5-second intervals
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action": "monitor",
			"monitoring": &GPUMonitoringConfig{
				Interval:     5 * time.Second,
				Duration:     30 * time.Second,
				Metrics:      []string{"utilization", "memory", "temperature"},
				OutputFormat: "json",
			},
		},
	}

	result, err := gpuClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error monitoring GPUs: %v", err)
		return
	}

	fmt.Printf("GPU Monitoring Result: %+v\n", result.Output)
}

// ExampleCUDAInfo demonstrates CUDA installation detection.
func ExampleCUDAInfo() {
	ctx := context.Background()

	gpuClient := NewGPUClient(GPUConfig{})

	// Get CUDA information
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action": "cuda-info",
		},
	}

	result, err := gpuClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error getting CUDA info: %v", err)
		return
	}

	fmt.Printf("CUDA Info Result: %+v\n", result.Output)
}

// ExampleDockerGPUIntegration demonstrates Docker container with GPU support.
func ExampleDockerGPUIntegration() {
	ctx := context.Background()

	// Configure Docker with GPU support
	dockerConfig := DockerConfig{
		Engine: "docker",
	}

	dockerClient := NewDockerClient(dockerConfig)

	// Run a container with GPU access
	task := &Task{
		Type: "container",
		Payload: map[string]interface{}{
			"action": "run",
			"image":  "nvidia/cuda:11.8-runtime-ubuntu20.04",
			"name":   "gpu-test-container",
			"detach": true,
			"gpus": &DockerGPUConfig{
				Enabled:      true,
				Runtime:      "nvidia",
				All:          true,
				Capabilities: []string{"compute", "utility"},
			},
			"command": []string{"nvidia-smi"},
		},
	}

	result, err := dockerClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error running GPU container: %v", err)
		return
	}

	fmt.Printf("Docker GPU Container Result: %+v\n", result.Output)
}

// ExampleKubernetesGPUWorkload demonstrates Kubernetes GPU resource allocation.
func ExampleKubernetesGPUWorkload() {
	ctx := context.Background()

	k8sClient := NewKubernetesClient(KubernetesConfig{})

	// Create a pod with GPU resources
	gpuManifest := `
apiVersion: v1
kind: Pod
metadata:
  name: gpu-workload
  namespace: default
spec:
  containers:
  - name: gpu-container
    image: nvidia/cuda:11.8-runtime-ubuntu20.04
    command: ["nvidia-smi"]
    resources:
      limits:
        nvidia.com/gpu: 1
      requests:
        nvidia.com/gpu: 1
  nodeSelector:
    accelerator: nvidia-tesla-gpu
  tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
`

	task := &Task{
		Type: "kubernetes",
		Payload: map[string]interface{}{
			"action":    "apply",
			"manifest":  gpuManifest,
			"namespace": "default",
			"gpus": &K8sGPUConfig{
				Enabled:      true,
				ResourceName: "nvidia.com/gpu",
				Limit:        1,
				Request:      1,
				NodeSelector: map[string]string{
					"accelerator": "nvidia-tesla-gpu",
				},
			},
		},
	}

	result, err := k8sClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error creating GPU workload: %v", err)
		return
	}

	fmt.Printf("Kubernetes GPU Workload Result: %+v\n", result.Output)
}

// ExampleProxmoxGPUPassthrough demonstrates Proxmox GPU passthrough configuration.
func ExampleProxmoxGPUPassthrough() {
	ctx := context.Background()

	proxmoxConfig := ProxmoxConfig{
		URL:      "https://pve.example.com:8006",
		Username: "root@pam",
		Node:     "pve-node1",
	}

	proxmoxClient := NewProxmoxClient(proxmoxConfig)

	// Create a VM with GPU passthrough
	task := &Task{
		Type: "vm",
		Payload: map[string]interface{}{
			"action":  "create",
			"vm_type": "qemu",
			"vmid":    100,
			"node":    "pve-node1",
			"config": map[string]interface{}{
				"cores":  4,
				"memory": 8192,
				"scsihw": "virtio-scsi-pci",
				"net0":   "virtio,bridge=vmbr0",
				"vga":    "none",
			},
			"gpus": &ProxmoxGPUConfig{
				Enabled:    true,
				PCIDevices: []string{"01:00.0", "01:00.1"}, // GPU and audio device
				VFIO:       true,
				Primary:    true,
				ROMBAR:     true,
				X_VGA:      true,
			},
		},
	}

	result, err := proxmoxClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error creating GPU VM: %v", err)
		return
	}

	fmt.Printf("Proxmox GPU VM Result: %+v\n", result.Output)
}

// ExampleMLEnvironmentSetup demonstrates ML environment setup with GPU tools.
func ExampleMLEnvironmentSetup() {
	ctx := context.Background()

	// Create GPU client and tools
	gpuClient := NewGPUClient(GPUConfig{})
	gpuTools := NewGPUToolsClient(gpuClient)

	// Define ML environment requirements
	mlConfig := MLEnvironmentConfig{
		GPURequirements: GPURequirements{
			MinMemoryGB:     8.0,
			RequiredGPUs:    1,
			WorkloadType:    "ml",
			PreferredModels: []string{"RTX", "Tesla", "A100"},
		},
		RequiredCUDAVersion: "11.0",
		Framework:           "pytorch",
		PythonVersion:       "3.9",
		ContainerImage:      "pytorch/pytorch:1.13.1-cuda11.6-cudnn8-runtime",
		WorkspacePath:       "/workspace",
		AdditionalPackages:  []string{"transformers", "datasets", "tensorboard"},
	}

	// Setup ML environment
	result, err := gpuTools.SetupMLEnvironment(ctx, mlConfig)
	if err != nil {
		log.Printf("Error setting up ML environment: %v", err)
		return
	}

	if !result.Success {
		log.Printf("ML environment setup failed: %s", result.Error)
		return
	}

	fmt.Printf("ML Environment Setup Result:\n")
	fmt.Printf("Selected GPUs: %+v\n", result.SelectedGPUs)
	fmt.Printf("CUDA Info: %+v\n", result.CUDAInfo)
	fmt.Printf("Environment Variables: %+v\n", result.Environment)
	fmt.Printf("Docker Config: %+v\n", result.DockerRunConfig)
	fmt.Printf("K8s Manifest:\n%s\n", result.K8sManifest)
}

// ExampleGPUAllocation demonstrates GPU allocation for agentic workloads.
func ExampleGPUAllocation() {
	ctx := context.Background()

	gpuClient := NewGPUClient(GPUConfig{})

	// Allocate GPUs for a container
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action":       "allocate",
			"container_id": "my-ml-container",
			"gpu_ids":      []int{0, 1}, // Allocate specific GPUs
			"exclusive":    true,
			"memory":       8192, // 8GB memory limit
		},
	}

	result, err := gpuClient.Execute(ctx, task)
	if err != nil {
		log.Printf("Error allocating GPUs: %v", err)
		return
	}

	fmt.Printf("GPU Allocation Result: %+v\n", result.Output)

	// Deallocate GPUs when done
	deallocateTask := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action":       "deallocate",
			"container_id": "my-ml-container",
			"gpu_ids":      []int{0, 1},
		},
	}

	deallocateResult, err := gpuClient.Execute(ctx, deallocateTask)
	if err != nil {
		log.Printf("Error deallocating GPUs: %v", err)
		return
	}

	fmt.Printf("GPU Deallocation Result: %+v\n", deallocateResult.Output)
}

// ExampleGPUWorkflowOrchestration demonstrates orchestrating a complete GPU workflow.
func ExampleGPUWorkflowOrchestration() {
	ctx := context.Background()

	// Create orchestrator with all connectors
	config := DefaultConfig()
	secretManager := NewSecretManager()

	// Add environment store for secrets
	secretManager.AddStore("env", NewEnvSecretStore())

	orchestrator, err := NewOrchestrator(config, secretManager)
	if err != nil {
		log.Printf("Error creating orchestrator: %v", err)
		return
	}

	// Create a workflow for GPU-enabled ML training
	workflow := &InfrastructureWorkflow{
		ID:          "gpu-ml-training",
		Name:        "GPU ML Training Workflow",
		Description: "Complete workflow for GPU-enabled ML model training",
		Steps: []WorkflowStep{
			{
				ID:       "detect-gpus",
				Name:     "Detect Available GPUs",
				TaskType: "gpu",
				Parameters: map[string]interface{}{
					"action": "list",
				},
			},
			{
				ID:        "setup-environment",
				Name:      "Setup ML Environment",
				TaskType:  "container",
				DependsOn: []string{"detect-gpus"},
				Parameters: map[string]interface{}{
					"action": "run",
					"image":  "pytorch/pytorch:latest",
					"name":   "ml-training",
					"detach": true,
					"gpus": map[string]interface{}{
						"enabled": true,
						"all":     true,
						"runtime": "nvidia",
					},
					"volumes": []string{"/data:/workspace/data"},
				},
			},
			{
				ID:        "monitor-training",
				Name:      "Monitor GPU Usage",
				TaskType:  "gpu",
				DependsOn: []string{"setup-environment"},
				Parameters: map[string]interface{}{
					"action": "monitor",
					"monitoring": map[string]interface{}{
						"interval": "10s",
						"duration": "5m",
						"metrics":  []string{"utilization", "memory", "temperature"},
					},
				},
			},
		},
	}

	// Execute the workflow
	result, err := orchestrator.ExecuteInfrastructureWorkflow(ctx, workflow)
	if err != nil {
		log.Printf("Error executing GPU workflow: %v", err)
		return
	}

	fmt.Printf("GPU Workflow Result:\n")
	fmt.Printf("Status: %s\n", result.Status)
	for _, step := range result.Steps {
		fmt.Printf("Step %s (%s): %s\n", step.StepID, step.Name, step.Status)
	}
}
