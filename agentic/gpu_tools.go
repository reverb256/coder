// Package agentic provides GPU tools for agentic workloads and ML/AI tasks.
package agentic

import (
	"context"
	"fmt"

	"golang.org/x/xerrors"
)

// GPUToolsClient provides high-level GPU tools for agentic workloads.
type GPUToolsClient struct {
	gpuClient *GPUClient
}

// NewGPUToolsClient creates a new GPU tools client.
func NewGPUToolsClient(gpuClient *GPUClient) *GPUToolsClient {
	return &GPUToolsClient{
		gpuClient: gpuClient,
	}
}

// CheckGPUAvailability checks if GPUs are available for agentic workloads.
func (g *GPUToolsClient) CheckGPUAvailability(ctx context.Context) (*GPUAvailabilityResult, error) {
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action": "list",
		},
	}

	result, err := g.gpuClient.Execute(ctx, task)
	if err != nil {
		return &GPUAvailabilityResult{
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	gpuData, ok := result.Output.(map[string]interface{})
	if !ok {
		return &GPUAvailabilityResult{
			Available: false,
			Error:     "invalid GPU data format",
		}, nil
	}

	gpuCount, _ := gpuData["gpu_count"].(int)
	gpus, _ := gpuData["gpus"].([]GPUInfo)

	availability := &GPUAvailabilityResult{
		Available:    gpuCount > 0,
		GPUCount:     gpuCount,
		GPUs:         gpus,
		Capabilities: g.detectCapabilities(gpus),
	}

	return availability, nil
}

// SelectOptimalGPUs selects the best GPUs for a specific workload.
func (g *GPUToolsClient) SelectOptimalGPUs(ctx context.Context, requirements GPURequirements) (*GPUSelectionResult, error) {
	// Get available GPUs
	availability, err := g.CheckGPUAvailability(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to check GPU availability: %w", err)
	}

	if !availability.Available {
		return &GPUSelectionResult{
			Success: false,
			Error:   "no GPUs available",
		}, nil
	}

	// Filter and rank GPUs based on requirements
	candidates := g.filterGPUsByRequirements(availability.GPUs, requirements)
	if len(candidates) == 0 {
		return &GPUSelectionResult{
			Success: false,
			Error:   "no GPUs meet the requirements",
		}, nil
	}

	// Select the best GPUs
	selected := g.selectBestGPUs(candidates, requirements)

	return &GPUSelectionResult{
		Success:      true,
		SelectedGPUs: selected,
		TotalMemory:  g.calculateTotalMemory(selected),
		Reasoning:    g.generateSelectionReasoning(selected, requirements),
	}, nil
}

// SetupMLEnvironment sets up a GPU-enabled ML/AI environment.
func (g *GPUToolsClient) SetupMLEnvironment(ctx context.Context, config MLEnvironmentConfig) (*MLEnvironmentResult, error) {
	// Check GPU requirements
	gpuSelection, err := g.SelectOptimalGPUs(ctx, config.GPURequirements)
	if err != nil {
		return nil, xerrors.Errorf("failed to select GPUs: %w", err)
	}

	if !gpuSelection.Success {
		return &MLEnvironmentResult{
			Success: false,
			Error:   fmt.Sprintf("GPU selection failed: %s", gpuSelection.Error),
		}, nil
	}

	// Get CUDA info
	cudaTask := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action": "cuda-info",
		},
	}

	cudaResult, err := g.gpuClient.Execute(ctx, cudaTask)
	if err != nil {
		return &MLEnvironmentResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get CUDA info: %v", err),
		}, nil
	}

	cudaData, _ := cudaResult.Output.(map[string]interface{})
	cudaInfo, _ := cudaData["cuda_info"].(CUDAInfo)

	// Validate CUDA version compatibility
	if config.RequiredCUDAVersion != "" && !g.isCUDAVersionCompatible(cudaInfo.Version, config.RequiredCUDAVersion) {
		return &MLEnvironmentResult{
			Success: false,
			Error:   fmt.Sprintf("CUDA version %s not compatible with required %s", cudaInfo.Version, config.RequiredCUDAVersion),
		}, nil
	}

	return &MLEnvironmentResult{
		Success:         true,
		SelectedGPUs:    gpuSelection.SelectedGPUs,
		CUDAInfo:        cudaInfo,
		Environment:     g.generateMLEnvironment(gpuSelection.SelectedGPUs, cudaInfo, config),
		DockerRunConfig: g.generateDockerConfig(gpuSelection.SelectedGPUs, config),
		K8sManifest:     g.generateK8sManifest(gpuSelection.SelectedGPUs, config),
	}, nil
}

// MonitorGPUWorkload monitors GPU usage for agentic workloads.
func (g *GPUToolsClient) MonitorGPUWorkload(ctx context.Context, config GPUMonitoringConfig) (*GPUMonitoringResult, error) {
	task := &Task{
		Type: "gpu",
		Payload: map[string]interface{}{
			"action":     "monitor",
			"monitoring": config,
		},
	}

	result, err := g.gpuClient.Execute(ctx, task)
	if err != nil {
		return nil, xerrors.Errorf("failed to monitor GPUs: %w", err)
	}

	monitoringData, ok := result.Output.(map[string]interface{})
	if !ok {
		return nil, xerrors.New("invalid monitoring data format")
	}

	return &GPUMonitoringResult{
		Success:     true,
		MonitorData: monitoringData,
		Insights:    g.generateMonitoringInsights(monitoringData),
	}, nil
}

// Helper types

// GPUAvailabilityResult represents GPU availability information.
type GPUAvailabilityResult struct {
	Available    bool      `json:"available"`
	GPUCount     int       `json:"gpu_count"`
	GPUs         []GPUInfo `json:"gpus"`
	Capabilities []string  `json:"capabilities"`
	Error        string    `json:"error,omitempty"`
}

// GPURequirements represents requirements for GPU selection.
type GPURequirements struct {
	MinMemoryGB     float64  `json:"min_memory_gb"`
	MinComputeUnits int      `json:"min_compute_units"`
	RequiredGPUs    int      `json:"required_gpus"`
	PreferredModels []string `json:"preferred_models"`
	MaxPowerDraw    float64  `json:"max_power_draw"`
	WorkloadType    string   `json:"workload_type"` // "ml", "ai", "compute", "graphics"
}

// GPUSelectionResult represents the result of GPU selection.
type GPUSelectionResult struct {
	Success      bool      `json:"success"`
	SelectedGPUs []GPUInfo `json:"selected_gpus"`
	TotalMemory  int64     `json:"total_memory_mb"`
	Reasoning    string    `json:"reasoning"`
	Error        string    `json:"error,omitempty"`
}

// MLEnvironmentConfig represents configuration for ML environment setup.
type MLEnvironmentConfig struct {
	GPURequirements     GPURequirements `json:"gpu_requirements"`
	RequiredCUDAVersion string          `json:"required_cuda_version"`
	Framework           string          `json:"framework"` // "pytorch", "tensorflow", "jax"
	PythonVersion       string          `json:"python_version"`
	AdditionalPackages  []string        `json:"additional_packages"`
	ContainerImage      string          `json:"container_image"`
	WorkspacePath       string          `json:"workspace_path"`
}

// MLEnvironmentResult represents the result of ML environment setup.
type MLEnvironmentResult struct {
	Success         bool                   `json:"success"`
	SelectedGPUs    []GPUInfo              `json:"selected_gpus"`
	CUDAInfo        CUDAInfo               `json:"cuda_info"`
	Environment     map[string]string      `json:"environment"`
	DockerRunConfig map[string]interface{} `json:"docker_run_config"`
	K8sManifest     string                 `json:"k8s_manifest"`
	Error           string                 `json:"error,omitempty"`
}

// GPUMonitoringResult represents GPU monitoring results.
type GPUMonitoringResult struct {
	Success     bool                   `json:"success"`
	MonitorData map[string]interface{} `json:"monitor_data"`
	Insights    []string               `json:"insights"`
	Error       string                 `json:"error,omitempty"`
}

// Helper methods

// detectCapabilities detects GPU capabilities.
func (g *GPUToolsClient) detectCapabilities(gpus []GPUInfo) []string {
	capabilities := []string{}

	for _, gpu := range gpus {
		if gpu.CUDAVersion != "" {
			capabilities = append(capabilities, "cuda")
		}

		// Check for specific GPU models and their capabilities
		gpuName := gpu.Name
		if contains(gpuName, "RTX") || contains(gpuName, "GTX") {
			capabilities = append(capabilities, "gaming", "compute")
		}
		if contains(gpuName, "Tesla") || contains(gpuName, "V100") || contains(gpuName, "A100") {
			capabilities = append(capabilities, "datacenter", "ml", "ai")
		}
		if contains(gpuName, "Quadro") {
			capabilities = append(capabilities, "professional", "graphics")
		}
	}

	return uniqueStrings(capabilities)
}

// filterGPUsByRequirements filters GPUs based on requirements.
func (g *GPUToolsClient) filterGPUsByRequirements(gpus []GPUInfo, req GPURequirements) []GPUInfo {
	var candidates []GPUInfo

	for _, gpu := range gpus {
		// Check memory requirement
		if req.MinMemoryGB > 0 {
			memoryGB := float64(gpu.MemoryTotal) / 1024.0
			if memoryGB < req.MinMemoryGB {
				continue
			}
		}

		// Check power draw
		if req.MaxPowerDraw > 0 && gpu.PowerDraw > req.MaxPowerDraw {
			continue
		}

		// Check if GPU is available
		if gpu.Status != "available" {
			continue
		}

		// Check preferred models
		if len(req.PreferredModels) > 0 {
			found := false
			for _, model := range req.PreferredModels {
				if contains(gpu.Name, model) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		candidates = append(candidates, gpu)
	}

	return candidates
}

// selectBestGPUs selects the best GPUs from candidates.
func (g *GPUToolsClient) selectBestGPUs(candidates []GPUInfo, req GPURequirements) []GPUInfo {
	if len(candidates) == 0 {
		return []GPUInfo{}
	}

	// Sort by memory (descending) and utilization (ascending)
	// For simplicity, just take the first N GPUs
	requiredGPUs := req.RequiredGPUs
	if requiredGPUs == 0 {
		requiredGPUs = 1
	}

	if len(candidates) < requiredGPUs {
		return candidates
	}

	return candidates[:requiredGPUs]
}

// calculateTotalMemory calculates total memory of selected GPUs.
func (g *GPUToolsClient) calculateTotalMemory(gpus []GPUInfo) int64 {
	var total int64
	for _, gpu := range gpus {
		total += gpu.MemoryTotal
	}
	return total
}

// generateSelectionReasoning generates reasoning for GPU selection.
func (g *GPUToolsClient) generateSelectionReasoning(gpus []GPUInfo, req GPURequirements) string {
	if len(gpus) == 0 {
		return "No GPUs selected"
	}

	totalMemory := g.calculateTotalMemory(gpus)
	return fmt.Sprintf("Selected %d GPU(s) with total memory %d MB for %s workload",
		len(gpus), totalMemory, req.WorkloadType)
}

// isCUDAVersionCompatible checks CUDA version compatibility.
func (g *GPUToolsClient) isCUDAVersionCompatible(available, required string) bool {
	// Simple version comparison - in practice, this would be more sophisticated
	return available >= required
}

// generateMLEnvironment generates ML environment variables.
func (g *GPUToolsClient) generateMLEnvironment(gpus []GPUInfo, cuda CUDAInfo, config MLEnvironmentConfig) map[string]string {
	env := map[string]string{
		"CUDA_VISIBLE_DEVICES":   g.getGPUIDsString(gpus),
		"NVIDIA_VISIBLE_DEVICES": g.getGPUIDsString(gpus),
		"CUDA_VERSION":           cuda.Version,
	}

	if config.Framework == "pytorch" {
		env["PYTORCH_CUDA_ALLOC_CONF"] = "max_split_size_mb:128"
	} else if config.Framework == "tensorflow" {
		env["TF_FORCE_GPU_ALLOW_GROWTH"] = "true"
	}

	return env
}

// generateDockerConfig generates Docker run configuration for GPUs.
func (g *GPUToolsClient) generateDockerConfig(gpus []GPUInfo, config MLEnvironmentConfig) map[string]interface{} {
	gpuConfig := &DockerGPUConfig{
		Enabled:      true,
		Runtime:      "nvidia",
		GPUIDs:       g.getGPUIDs(gpus),
		Capabilities: []string{"compute", "utility"},
	}

	dockerConfig := map[string]interface{}{
		"image":   config.ContainerImage,
		"runtime": "nvidia",
		"gpus":    gpuConfig,
		"env":     g.generateMLEnvironment(gpus, CUDAInfo{}, config),
	}

	if config.WorkspacePath != "" {
		dockerConfig["volumes"] = []string{
			fmt.Sprintf("%s:/workspace", config.WorkspacePath),
		}
	}

	return dockerConfig
}

// generateK8sManifest generates Kubernetes manifest for GPU workloads.
func (g *GPUToolsClient) generateK8sManifest(gpus []GPUInfo, config MLEnvironmentConfig) string {
	gpuCount := len(gpus)

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: ml-workload
spec:
  containers:
  - name: ml-container
    image: %s
    resources:
      limits:
        nvidia.com/gpu: %d
      requests:
        nvidia.com/gpu: %d
    env:
    - name: CUDA_VISIBLE_DEVICES
      value: "all"
    - name: NVIDIA_VISIBLE_DEVICES
      value: "all"
  nodeSelector:
    accelerator: nvidia-tesla-gpu
  tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
`, config.ContainerImage, gpuCount, gpuCount)

	return manifest
}

// generateMonitoringInsights generates insights from monitoring data.
func (g *GPUToolsClient) generateMonitoringInsights(data map[string]interface{}) []string {
	insights := []string{}

	// Analyze samples if available
	if samples, ok := data["samples"].([]map[string]interface{}); ok && len(samples) > 0 {
		insights = append(insights, fmt.Sprintf("Collected %d monitoring samples", len(samples)))

		// Check for high utilization
		for _, sample := range samples {
			if gpus, ok := sample["gpus"].([]GPUInfo); ok {
				for _, gpu := range gpus {
					if gpu.UtilizationGPU > 90 {
						insights = append(insights, fmt.Sprintf("GPU %d shows high utilization (%d%%)", gpu.ID, gpu.UtilizationGPU))
					}
					if gpu.Temperature > 80 {
						insights = append(insights, fmt.Sprintf("GPU %d temperature is high (%d°C)", gpu.ID, gpu.Temperature))
					}
				}
			}
		}
	}

	return insights
}

// Utility functions

// getGPUIDs extracts GPU IDs from GPU info list.
func (g *GPUToolsClient) getGPUIDs(gpus []GPUInfo) []int {
	ids := make([]int, len(gpus))
	for i, gpu := range gpus {
		ids[i] = gpu.ID
	}
	return ids
}

// getGPUIDsString returns GPU IDs as comma-separated string.
func (g *GPUToolsClient) getGPUIDsString(gpus []GPUInfo) string {
	if len(gpus) == 0 {
		return "all"
	}

	ids := make([]string, len(gpus))
	for i, gpu := range gpus {
		ids[i] = fmt.Sprintf("%d", gpu.ID)
	}

	return fmt.Sprintf("%v", ids)
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			(len(s) > len(substr) && s[1:len(substr)+1] == substr))))
}

// uniqueStrings returns unique strings from a slice.
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range slice {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}
