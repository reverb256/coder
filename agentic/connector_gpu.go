// Package agentic provides NVIDIA GPU connector for hardware acceleration management.
package agentic

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

// GPUConfig holds GPU configuration.
type GPUConfig struct {
	NvidiaSMIPath    string `json:"nvidia_smi_path"`   // Path to nvidia-smi binary
	DockerRuntime    string `json:"docker_runtime"`    // Docker runtime for GPU support ("nvidia", "runc")
	CUDAVersion      string `json:"cuda_version"`      // Required CUDA version
	EnableMonitoring bool   `json:"enable_monitoring"` // Enable GPU monitoring
	MaxGPUsPerTask   int    `json:"max_gpus_per_task"` // Maximum GPUs per task
}

// GPUClient is an agent for NVIDIA GPU management tasks.
type GPUClient struct {
	cfg GPUConfig
}

// GPUTask represents a GPU management task.
type GPUTask struct {
	Action      string                 `json:"action"`       // detect, list, allocate, monitor, cuda-info, memory-info
	GPUIDs      []int                  `json:"gpu_ids"`      // Specific GPU IDs to target
	ContainerID string                 `json:"container_id"` // Container ID for GPU allocation
	PodName     string                 `json:"pod_name"`     // Kubernetes pod name
	Namespace   string                 `json:"namespace"`    // Kubernetes namespace
	Exclusive   bool                   `json:"exclusive"`    // Exclusive GPU allocation
	Memory      int64                  `json:"memory"`       // GPU memory requirement in MB
	Config      map[string]interface{} `json:"config"`       // Additional configuration
	Monitoring  *GPUMonitoringConfig   `json:"monitoring"`   // Monitoring configuration
}

// GPUMonitoringConfig represents GPU monitoring configuration.
type GPUMonitoringConfig struct {
	Interval     time.Duration `json:"interval"`      // Monitoring interval
	Duration     time.Duration `json:"duration"`      // Total monitoring duration
	Metrics      []string      `json:"metrics"`       // Metrics to collect
	OutputFormat string        `json:"output_format"` // json, csv, xml
}

// GPUInfo represents information about a GPU.
type GPUInfo struct {
	ID                int          `json:"id"`
	Name              string       `json:"name"`
	UUID              string       `json:"uuid"`
	PCIBusID          string       `json:"pci_bus_id"`
	DriverVersion     string       `json:"driver_version"`
	CUDAVersion       string       `json:"cuda_version"`
	MemoryTotal       int64        `json:"memory_total"`       // MB
	MemoryUsed        int64        `json:"memory_used"`        // MB
	MemoryFree        int64        `json:"memory_free"`        // MB
	UtilizationGPU    int          `json:"utilization_gpu"`    // Percentage
	UtilizationMemory int          `json:"utilization_memory"` // Percentage
	Temperature       int          `json:"temperature"`        // Celsius
	PowerDraw         float64      `json:"power_draw"`         // Watts
	PowerLimit        float64      `json:"power_limit"`        // Watts
	ClockGraphics     int          `json:"clock_graphics"`     // MHz
	ClockMemory       int          `json:"clock_memory"`       // MHz
	Processes         []GPUProcess `json:"processes"`
	Status            string       `json:"status"` // available, allocated, error
}

// GPUProcess represents a process using GPU.
type GPUProcess struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	MemoryUsed  int64  `json:"memory_used"` // MB
	Type        string `json:"type"`        // C (Compute), G (Graphics), C+G
}

// GPUAllocation represents a GPU allocation.
type GPUAllocation struct {
	GPUIDs      []int     `json:"gpu_ids"`
	ContainerID string    `json:"container_id,omitempty"`
	PodName     string    `json:"pod_name,omitempty"`
	Namespace   string    `json:"namespace,omitempty"`
	Exclusive   bool      `json:"exclusive"`
	AllocatedAt time.Time `json:"allocated_at"`
	MemoryLimit int64     `json:"memory_limit,omitempty"` // MB
}

// CUDAInfo represents CUDA installation information.
type CUDAInfo struct {
	Version           string   `json:"version"`
	DriverVersion     string   `json:"driver_version"`
	RuntimeVersion    string   `json:"runtime_version"`
	SupportedArchs    []string `json:"supported_architectures"`
	CUDNNVersion      string   `json:"cudnn_version,omitempty"`
	TensorRTVersion   string   `json:"tensorrt_version,omitempty"`
	InstallationPaths []string `json:"installation_paths"`
}

// NewGPUClient creates a new GPU client.
func NewGPUClient(cfg GPUConfig) *GPUClient {
	// Set sensible defaults
	if cfg.NvidiaSMIPath == "" {
		cfg.NvidiaSMIPath = "nvidia-smi"
	}
	if cfg.DockerRuntime == "" {
		cfg.DockerRuntime = "nvidia"
	}
	if cfg.MaxGPUsPerTask == 0 {
		cfg.MaxGPUsPerTask = 8
	}

	return &GPUClient{cfg: cfg}
}

func (g *GPUClient) Name() string { return "gpu" }

func (g *GPUClient) Supports(taskType string) bool {
	return taskType == "gpu" || taskType == "nvidia" || taskType == "cuda" || taskType == "hardware"
}

func (g *GPUClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Parse task payload
	var gpuTask GPUTask
	if err := mapToStruct(task.Payload, &gpuTask); err != nil {
		return &TaskResult{Error: xerrors.Errorf("invalid task payload: %w", err)}, nil
	}

	// Execute based on action
	switch gpuTask.Action {
	case "detect", "list":
		result, err := g.listGPUs(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "info":
		result, err := g.getGPUInfo(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "monitor":
		result, err := g.monitorGPUs(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "allocate":
		result, err := g.allocateGPUs(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "deallocate":
		result, err := g.deallocateGPUs(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "cuda-info":
		result, err := g.getCUDAInfo(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "memory-info":
		result, err := g.getMemoryInfo(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "processes":
		result, err := g.getGPUProcesses(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "topology":
		result, err := g.getGPUTopology(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "reset":
		result, err := g.resetGPUs(ctx, &gpuTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	default:
		return &TaskResult{Error: xerrors.Errorf("unsupported action: %s", gpuTask.Action)}, nil
	}
}

// listGPUs lists all available GPUs.
func (g *GPUClient) listGPUs(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	// Check if nvidia-smi is available
	if err := g.checkNvidiaSMI(ctx); err != nil {
		return nil, xerrors.Errorf("nvidia-smi not available: %w", err)
	}

	args := []string{
		"--query-gpu=index,name,uuid,pci.bus_id,driver_version,cuda_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw,power.limit,clocks.current.graphics,clocks.current.memory",
		"--format=csv,noheader,nounits",
	}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to list GPUs: %w", err)
	}

	gpus, err := g.parseGPUList(output)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse GPU list: %w", err)
	}

	result := map[string]interface{}{
		"action":    "list",
		"gpu_count": len(gpus),
		"gpus":      gpus,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// getGPUInfo gets detailed information about specific GPUs.
func (g *GPUClient) getGPUInfo(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	var args []string

	if len(task.GPUIDs) > 0 {
		// Query specific GPUs
		gpuIDs := make([]string, len(task.GPUIDs))
		for i, id := range task.GPUIDs {
			gpuIDs[i] = strconv.Itoa(id)
		}
		args = []string{
			fmt.Sprintf("--id=%s", strings.Join(gpuIDs, ",")),
		}
	}

	args = append(args, []string{
		"--query-gpu=index,name,uuid,pci.bus_id,driver_version,cuda_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw,power.limit,clocks.current.graphics,clocks.current.memory,compute_mode,persistence_mode",
		"--format=csv,noheader,nounits",
	}...)

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get GPU info: %w", err)
	}

	gpus, err := g.parseDetailedGPUInfo(output)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse GPU info: %w", err)
	}

	// Get processes for each GPU
	for i := range gpus {
		processes, err := g.getProcessesForGPU(ctx, gpus[i].ID)
		if err == nil {
			gpus[i].Processes = processes
		}
	}

	result := map[string]interface{}{
		"action":    "info",
		"gpu_count": len(gpus),
		"gpus":      gpus,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// monitorGPUs monitors GPU metrics over time.
func (g *GPUClient) monitorGPUs(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	if task.Monitoring == nil {
		task.Monitoring = &GPUMonitoringConfig{
			Interval:     5 * time.Second,
			Duration:     60 * time.Second,
			Metrics:      []string{"utilization", "memory", "temperature", "power"},
			OutputFormat: "json",
		}
	}

	var samples []map[string]interface{}
	startTime := time.Now()
	ticker := time.NewTicker(task.Monitoring.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Since(startTime) >= task.Monitoring.Duration {
				goto done
			}

			// Collect sample
			sample, err := g.collectMonitoringSample(ctx, task)
			if err != nil {
				continue // Skip failed samples
			}
			samples = append(samples, sample)

		}
	}

done:
	result := map[string]interface{}{
		"action":       "monitor",
		"start_time":   startTime.UTC(),
		"end_time":     time.Now().UTC(),
		"duration":     time.Since(startTime).String(),
		"interval":     task.Monitoring.Interval.String(),
		"sample_count": len(samples),
		"samples":      samples,
	}

	return result, nil
}

// allocateGPUs allocates GPUs for a container or pod.
func (g *GPUClient) allocateGPUs(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	// Get available GPUs
	gpuListTask := &GPUTask{Action: "list"}
	gpuListResult, err := g.listGPUs(ctx, gpuListTask)
	if err != nil {
		return nil, xerrors.Errorf("failed to get GPU list: %w", err)
	}

	gpus, ok := gpuListResult["gpus"].([]GPUInfo)
	if !ok {
		return nil, xerrors.New("failed to parse GPU list")
	}

	// Select GPUs for allocation
	var selectedGPUs []int
	if len(task.GPUIDs) > 0 {
		// Use specified GPUs
		selectedGPUs = task.GPUIDs
	} else {
		// Auto-select available GPUs
		for _, gpu := range gpus {
			if gpu.Status == "available" && len(selectedGPUs) < g.cfg.MaxGPUsPerTask {
				selectedGPUs = append(selectedGPUs, gpu.ID)
			}
		}
	}

	if len(selectedGPUs) == 0 {
		return nil, xerrors.New("no available GPUs for allocation")
	}

	allocation := GPUAllocation{
		GPUIDs:      selectedGPUs,
		ContainerID: task.ContainerID,
		PodName:     task.PodName,
		Namespace:   task.Namespace,
		Exclusive:   task.Exclusive,
		AllocatedAt: time.Now(),
		MemoryLimit: task.Memory,
	}

	result := map[string]interface{}{
		"action":     "allocate",
		"allocation": allocation,
		"gpu_count":  len(selectedGPUs),
		"gpu_ids":    selectedGPUs,
		"timestamp":  time.Now().UTC(),
	}

	return result, nil
}

// deallocateGPUs deallocates GPUs from a container or pod.
func (g *GPUClient) deallocateGPUs(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	var deallocatedGPUs []int

	if len(task.GPUIDs) > 0 {
		deallocatedGPUs = task.GPUIDs
	} else {
		// Find GPUs allocated to the specified container/pod
		// This would typically query an allocation database
		// For now, we'll simulate this
		deallocatedGPUs = []int{} // Would be populated from allocation tracking
	}

	result := map[string]interface{}{
		"action":           "deallocate",
		"deallocated_gpus": deallocatedGPUs,
		"container_id":     task.ContainerID,
		"pod_name":         task.PodName,
		"namespace":        task.Namespace,
		"timestamp":        time.Now().UTC(),
	}

	return result, nil
}

// getCUDAInfo gets CUDA installation information.
func (g *GPUClient) getCUDAInfo(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	cudaInfo := CUDAInfo{}

	// Get CUDA version from nvidia-smi
	args := []string{"--query-gpu=cuda_version", "--format=csv,noheader,nounits"}
	output, err := g.execNvidiaSMI(ctx, args...)
	if err == nil && output != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			cudaInfo.Version = strings.TrimSpace(lines[0])
		}
	}

	// Get driver version
	args = []string{"--query-gpu=driver_version", "--format=csv,noheader,nounits"}
	output, err = g.execNvidiaSMI(ctx, args...)
	if err == nil && output != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			cudaInfo.DriverVersion = strings.TrimSpace(lines[0])
		}
	}

	// Try to get nvcc version
	if nvccOutput, err := g.execCommand(ctx, "nvcc", "--version"); err == nil {
		if re := regexp.MustCompile(`release (\d+\.\d+)`); re != nil {
			if matches := re.FindStringSubmatch(nvccOutput); len(matches) > 1 {
				cudaInfo.RuntimeVersion = matches[1]
			}
		}
	}

	// Check for common CUDA installation paths
	commonPaths := []string{
		"/usr/local/cuda",
		"/opt/cuda",
		"/usr/cuda",
	}

	for _, path := range commonPaths {
		if output, err := g.execCommand(ctx, "ls", "-d", path); err == nil && strings.TrimSpace(output) != "" {
			cudaInfo.InstallationPaths = append(cudaInfo.InstallationPaths, path)
		}
	}

	result := map[string]interface{}{
		"action":    "cuda-info",
		"cuda_info": cudaInfo,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// getMemoryInfo gets detailed GPU memory information.
func (g *GPUClient) getMemoryInfo(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	args := []string{
		"--query-gpu=index,memory.total,memory.used,memory.free",
		"--format=csv,noheader,nounits",
	}

	if len(task.GPUIDs) > 0 {
		gpuIDs := make([]string, len(task.GPUIDs))
		for i, id := range task.GPUIDs {
			gpuIDs[i] = strconv.Itoa(id)
		}
		args = append([]string{fmt.Sprintf("--id=%s", strings.Join(gpuIDs, ","))}, args...)
	}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get memory info: %w", err)
	}

	memoryInfo := g.parseMemoryInfo(output)

	result := map[string]interface{}{
		"action":      "memory-info",
		"memory_info": memoryInfo,
		"timestamp":   time.Now().UTC(),
	}

	return result, nil
}

// getGPUProcesses gets processes running on GPUs.
func (g *GPUClient) getGPUProcesses(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	args := []string{
		"--query-compute-apps=gpu_uuid,pid,process_name,used_memory",
		"--format=csv,noheader,nounits",
	}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get GPU processes: %w", err)
	}

	processes := g.parseProcessInfo(output)

	result := map[string]interface{}{
		"action":    "processes",
		"processes": processes,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// getGPUTopology gets GPU topology information.
func (g *GPUClient) getGPUTopology(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	args := []string{"topo", "-m"}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get GPU topology: %w", err)
	}

	result := map[string]interface{}{
		"action":    "topology",
		"topology":  output,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// resetGPUs resets GPU state.
func (g *GPUClient) resetGPUs(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	var args []string

	if len(task.GPUIDs) > 0 {
		for _, id := range task.GPUIDs {
			args = append(args, "--gpu-reset", "-i", strconv.Itoa(id))
		}
	} else {
		args = []string{"--gpu-reset"}
	}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to reset GPUs: %w", err)
	}

	result := map[string]interface{}{
		"action":    "reset",
		"output":    output,
		"gpu_ids":   task.GPUIDs,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// Helper methods

// checkNvidiaSMI checks if nvidia-smi is available.
func (g *GPUClient) checkNvidiaSMI(ctx context.Context) error {
	_, err := g.execNvidiaSMI(ctx, "--version")
	return err
}

// execNvidiaSMI executes nvidia-smi command.
func (g *GPUClient) execNvidiaSMI(ctx context.Context, args ...string) (string, error) {
	return g.execCommand(ctx, g.cfg.NvidiaSMIPath, args...)
}

// execCommand executes a command.
func (g *GPUClient) execCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", xerrors.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// parseGPUList parses nvidia-smi CSV output into GPUInfo structs.
func (g *GPUClient) parseGPUList(output string) ([]GPUInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var gpus []GPUInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 16 {
			continue
		}

		gpu := GPUInfo{}

		if id, err := strconv.Atoi(strings.TrimSpace(fields[0])); err == nil {
			gpu.ID = id
		}

		gpu.Name = strings.TrimSpace(fields[1])
		gpu.UUID = strings.TrimSpace(fields[2])
		gpu.PCIBusID = strings.TrimSpace(fields[3])
		gpu.DriverVersion = strings.TrimSpace(fields[4])
		gpu.CUDAVersion = strings.TrimSpace(fields[5])

		if memTotal, err := strconv.ParseInt(strings.TrimSpace(fields[6]), 10, 64); err == nil {
			gpu.MemoryTotal = memTotal
		}

		if memUsed, err := strconv.ParseInt(strings.TrimSpace(fields[7]), 10, 64); err == nil {
			gpu.MemoryUsed = memUsed
		}

		if memFree, err := strconv.ParseInt(strings.TrimSpace(fields[8]), 10, 64); err == nil {
			gpu.MemoryFree = memFree
		}

		if utilGPU, err := strconv.Atoi(strings.TrimSpace(fields[9])); err == nil {
			gpu.UtilizationGPU = utilGPU
		}

		if utilMem, err := strconv.Atoi(strings.TrimSpace(fields[10])); err == nil {
			gpu.UtilizationMemory = utilMem
		}

		if temp, err := strconv.Atoi(strings.TrimSpace(fields[11])); err == nil {
			gpu.Temperature = temp
		}

		if power, err := strconv.ParseFloat(strings.TrimSpace(fields[12]), 64); err == nil {
			gpu.PowerDraw = power
		}

		if powerLimit, err := strconv.ParseFloat(strings.TrimSpace(fields[13]), 64); err == nil {
			gpu.PowerLimit = powerLimit
		}

		if clockGfx, err := strconv.Atoi(strings.TrimSpace(fields[14])); err == nil {
			gpu.ClockGraphics = clockGfx
		}

		if clockMem, err := strconv.Atoi(strings.TrimSpace(fields[15])); err == nil {
			gpu.ClockMemory = clockMem
		}

		// Determine status based on utilization
		if gpu.UtilizationGPU > 0 || len(gpu.Processes) > 0 {
			gpu.Status = "allocated"
		} else {
			gpu.Status = "available"
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// parseDetailedGPUInfo parses detailed GPU information.
func (g *GPUClient) parseDetailedGPUInfo(output string) ([]GPUInfo, error) {
	// Similar to parseGPUList but with additional fields
	return g.parseGPUList(output)
}

// getProcessesForGPU gets processes for a specific GPU.
func (g *GPUClient) getProcessesForGPU(ctx context.Context, gpuID int) ([]GPUProcess, error) {
	args := []string{
		fmt.Sprintf("--id=%d", gpuID),
		"--query-compute-apps=pid,process_name,used_memory",
		"--format=csv,noheader,nounits",
	}

	output, err := g.execNvidiaSMI(ctx, args...)
	if err != nil {
		return nil, err
	}

	return g.parseProcessInfo(output), nil
}

// parseProcessInfo parses process information from nvidia-smi output.
func (g *GPUClient) parseProcessInfo(output string) []GPUProcess {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var processes []GPUProcess

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 3 {
			continue
		}

		process := GPUProcess{}

		if pid, err := strconv.Atoi(strings.TrimSpace(fields[0])); err == nil {
			process.PID = pid
		}

		process.ProcessName = strings.TrimSpace(fields[1])

		if memUsed, err := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64); err == nil {
			process.MemoryUsed = memUsed
		}

		process.Type = "C" // Default to compute

		processes = append(processes, process)
	}

	return processes
}

// parseMemoryInfo parses memory information.
func (g *GPUClient) parseMemoryInfo(output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var memInfo []map[string]interface{}

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 4 {
			continue
		}

		info := map[string]interface{}{
			"gpu_id": strings.TrimSpace(fields[0]),
		}

		if memTotal, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64); err == nil {
			info["memory_total"] = memTotal
		}

		if memUsed, err := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64); err == nil {
			info["memory_used"] = memUsed
		}

		if memFree, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			info["memory_free"] = memFree
		}

		memInfo = append(memInfo, info)
	}

	return memInfo
}

// collectMonitoringSample collects a monitoring sample.
func (g *GPUClient) collectMonitoringSample(ctx context.Context, task *GPUTask) (map[string]interface{}, error) {
	gpuInfoTask := &GPUTask{Action: "info", GPUIDs: task.GPUIDs}
	gpuInfoResult, err := g.getGPUInfo(ctx, gpuInfoTask)
	if err != nil {
		return nil, err
	}

	sample := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"gpus":      gpuInfoResult["gpus"],
	}

	return sample, nil
}
