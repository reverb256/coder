# GPU & Hardware Acceleration Support

This document describes the GPU and hardware acceleration capabilities added to the agentic orchestration system as part of Step 4 implementation.

## Overview

The GPU connector provides comprehensive NVIDIA GPU detection, management, allocation, and monitoring capabilities for agentic workloads. It integrates seamlessly with existing infrastructure connectors (Docker, Kubernetes, Proxmox) to enable GPU-accelerated containers, VMs, and ML/AI workloads.

## Features

### 1. NVIDIA GPU Detection and Management
- **GPU Discovery**: Automatic detection of available NVIDIA GPUs using `nvidia-smi`
- **GPU Information**: Detailed GPU specifications including memory, utilization, temperature, power draw
- **CUDA Support**: CUDA version detection and compatibility checking
- **GPU Status Monitoring**: Real-time monitoring of GPU utilization and health

### 2. GPU Resource Allocation
- **Dynamic Allocation**: Assign GPUs to specific containers or Kubernetes pods
- **Exclusive Access**: Support for exclusive GPU allocation to prevent resource conflicts
- **GPU Sharing**: Multiple workloads can share GPUs when appropriate
- **Memory Management**: GPU memory monitoring and limitation support

### 3. Infrastructure Integration
- **Docker Integration**: Full support for `--gpus` flag and nvidia runtime
- **Kubernetes Integration**: GPU resource requests and limits via device plugins
- **Proxmox Integration**: GPU passthrough configuration for VMs
- **Automatic Runtime Detection**: Detects and configures appropriate GPU runtimes

### 4. Agentic GPU Tools
- **ML Environment Setup**: Automated setup of GPU-enabled ML/AI environments
- **GPU Selection**: Intelligent GPU selection based on workload requirements
- **Monitoring Tools**: Comprehensive GPU monitoring and performance insights
- **CUDA Management**: CUDA toolkit detection and version management

## Quick Start

### Basic GPU Detection

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/coder/coder/agentic"
)

func main() {
    ctx := context.Background()
    
    // Create GPU client
    gpuClient := agentic.NewGPUClient(agentic.GPUConfig{})
    
    // List available GPUs
    task := &agentic.Task{
        Type: "gpu",
        Payload: map[string]interface{}{
            "action": "list",
        },
    }
    
    result, err := gpuClient.Execute(ctx, task)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Available GPUs: %+v\n", result.Output)
}
```

### Docker with GPU Support

```go
// Run a CUDA container with GPU access
task := &agentic.Task{
    Type: "container",
    Payload: map[string]interface{}{
        "action": "run",
        "image":  "nvidia/cuda:11.8-runtime-ubuntu20.04",
        "name":   "gpu-workload",
        "detach": true,
        "gpus": &agentic.DockerGPUConfig{
            Enabled:      true,
            Runtime:      "nvidia",
            All:          true,
            Capabilities: []string{"compute", "utility"},
        },
        "command": []string{"nvidia-smi"},
    },
}
```

### Kubernetes GPU Workload

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-workload
spec:
  containers:
  - name: gpu-container
    image: nvidia/cuda:11.8-runtime-ubuntu20.04
    resources:
      limits:
        nvidia.com/gpu: 1
      requests:
        nvidia.com/gpu: 1
  nodeSelector:
    accelerator: nvidia-tesla-gpu
```

### ML Environment Setup

```go
// Setup ML environment with GPU tools
gpuTools := agentic.NewGPUToolsClient(gpuClient)

mlConfig := agentic.MLEnvironmentConfig{
    GPURequirements: agentic.GPURequirements{
        MinMemoryGB:  8.0,
        RequiredGPUs: 1,
        WorkloadType: "ml",
    },
    Framework:      "pytorch",
    ContainerImage: "pytorch/pytorch:1.13.1-cuda11.6-cudnn8-runtime",
}

result, err := gpuTools.SetupMLEnvironment(ctx, mlConfig)
```

## Configuration

### GPU Connector Configuration

```go
type GPUConfig struct {
    NvidiaSMIPath    string // Path to nvidia-smi binary (default: "nvidia-smi")
    DockerRuntime    string // Docker runtime for GPU support (default: "nvidia")
    CUDAVersion      string // Required CUDA version
    EnableMonitoring bool   // Enable GPU monitoring
    MaxGPUsPerTask   int    // Maximum GPUs per task (default: 8)
}
```

### Environment Variables

The following environment variables can be used to configure GPU support:

```bash
# NVIDIA GPU Configuration
NVIDIA_VISIBLE_DEVICES=all      # Control which GPUs are visible
CUDA_VISIBLE_DEVICES=0,1        # CUDA-specific GPU visibility
NVIDIA_DRIVER_CAPABILITIES=compute,utility  # Driver capabilities

# Docker GPU Runtime
DOCKER_RUNTIME=nvidia            # Default Docker runtime for GPUs

# Kubernetes GPU Resources
KUBE_GPU_RESOURCE=nvidia.com/gpu # GPU resource name in Kubernetes
```

## API Reference

### GPU Actions

The GPU connector supports the following actions:

- **`list`**: List all available GPUs
- **`info`**: Get detailed GPU information
- **`monitor`**: Monitor GPU usage over time
- **`allocate`**: Allocate GPUs to workloads
- **`deallocate`**: Release allocated GPUs
- **`cuda-info`**: Get CUDA installation information
- **`memory-info`**: Get GPU memory information
- **`processes`**: List processes using GPUs
- **`topology`**: Get GPU topology information
- **`reset`**: Reset GPU state

### Docker GPU Configuration

```go
type DockerGPUConfig struct {
    Enabled     bool     // Enable GPU support
    Runtime     string   // GPU runtime (nvidia, runc)
    GPUIDs      []int    // Specific GPU IDs
    All         bool     // Use all available GPUs
    Capabilities []string // GPU capabilities (compute, utility, graphics)
    Memory      string   // GPU memory limit
}
```

### Kubernetes GPU Configuration

```go
type K8sGPUConfig struct {
    Enabled       bool              // Enable GPU support
    ResourceName  string            // GPU resource name (nvidia.com/gpu)
    Limit         int               // Number of GPUs to request
    Request       int               // Number of GPUs to request (minimum)
    NodeSelector  map[string]string // Node selector for GPU nodes
    Tolerations   []K8sToleration   // Tolerations for GPU nodes
    RuntimeClass  string            // Runtime class for GPU workloads
}
```

### Proxmox GPU Configuration

```go
type ProxmoxGPUConfig struct {
    Enabled     bool     // Enable GPU passthrough
    PCIDevices  []string // PCI device IDs (e.g., "01:00.0")
    GPUIDs      []int    // GPU IDs to passthrough
    VFIO        bool     // Use VFIO for passthrough
    Primary     bool     // Set as primary GPU
    ROMBAR      bool     // Enable ROM BAR
    X_VGA       bool     // Enable VGA passthrough
}
```

## Requirements

### System Requirements

- **NVIDIA GPU**: Compatible NVIDIA graphics card
- **NVIDIA Driver**: Latest NVIDIA drivers installed
- **CUDA**: CUDA toolkit (optional, for development)
- **nvidia-smi**: NVIDIA System Management Interface
- **Docker**: Docker with nvidia runtime (for containers)
- **Kubernetes**: Device plugin for GPU support (for K8s)

### Prerequisites

1. **Install NVIDIA Drivers**:
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install nvidia-driver-470
   
   # Verify installation
   nvidia-smi
   ```

2. **Install Docker with NVIDIA Runtime**:
   ```bash
   # Install nvidia-docker2
   distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
   curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
   curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
   
   sudo apt update
   sudo apt install nvidia-docker2
   sudo systemctl restart docker
   ```

3. **Kubernetes GPU Device Plugin** (for K8s):
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.0/nvidia-device-plugin.yml
   ```

## Examples

### Complete ML Training Workflow

```go
func ExampleMLTrainingWorkflow() {
    ctx := context.Background()
    
    // Setup GPU client and tools
    gpuClient := agentic.NewGPUClient(agentic.GPUConfig{})
    gpuTools := agentic.NewGPUToolsClient(gpuClient)
    
    // Check GPU availability
    availability, err := gpuTools.CheckGPUAvailability(ctx)
    if err != nil || !availability.Available {
        log.Fatal("No GPUs available")
    }
    
    // Setup ML environment
    mlConfig := agentic.MLEnvironmentConfig{
        GPURequirements: agentic.GPURequirements{
            MinMemoryGB:     8.0,
            RequiredGPUs:    1,
            WorkloadType:    "ml",
            PreferredModels: []string{"RTX", "Tesla"},
        },
        Framework:           "pytorch",
        ContainerImage:      "pytorch/pytorch:latest",
        RequiredCUDAVersion: "11.0",
    }
    
    mlEnv, err := gpuTools.SetupMLEnvironment(ctx, mlConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("ML Environment ready with %d GPUs\n", len(mlEnv.SelectedGPUs))
    
    // Start monitoring
    monitoring := agentic.GPUMonitoringConfig{
        Interval: 10 * time.Second,
        Duration: 5 * time.Minute,
        Metrics:  []string{"utilization", "memory", "temperature"},
    }
    
    go func() {
        result, _ := gpuTools.MonitorGPUWorkload(ctx, monitoring)
        fmt.Printf("Monitoring results: %+v\n", result.Insights)
    }()
}
```

### Multi-Node GPU Cluster

```go
func ExampleGPUCluster() {
    // Create GPU-enabled workflow
    workflow := &agentic.InfrastructureWorkflow{
        ID:   "gpu-cluster-setup",
        Name: "GPU Cluster Setup",
        Steps: []agentic.WorkflowStep{
            {
                ID:       "detect-gpus",
                Name:     "Detect Available GPUs",
                TaskType: "gpu",
                Parameters: map[string]interface{}{
                    "action": "list",
                },
            },
            {
                ID:        "setup-nodes",
                Name:      "Setup GPU Nodes",
                TaskType:  "kubernetes",
                DependsOn: []string{"detect-gpus"},
                Parameters: map[string]interface{}{
                    "action": "apply",
                    "manifest": generateGPUNodeManifest(),
                },
            },
            {
                ID:        "deploy-workloads",
                Name:      "Deploy GPU Workloads",
                TaskType:  "kubernetes",
                DependsOn: []string{"setup-nodes"},
                Parameters: map[string]interface{}{
                    "action":   "apply",
                    "manifest": generateGPUWorkloadManifest(),
                },
            },
        },
    }
}
```

## Troubleshooting

### Common Issues

1. **nvidia-smi not found**:
   - Ensure NVIDIA drivers are installed
   - Check PATH includes `/usr/bin`

2. **Docker GPU runtime not available**:
   - Install nvidia-docker2 package
   - Restart Docker daemon
   - Verify with: `docker run --gpus all nvidia/cuda:latest nvidia-smi`

3. **Kubernetes GPU resources not available**:
   - Install NVIDIA device plugin
   - Check node labels for GPU support
   - Verify resource requests/limits

4. **GPU allocation conflicts**:
   - Check for exclusive allocation settings
   - Monitor GPU utilization
   - Use GPU tools for optimal selection

### Debug Commands

```bash
# Check GPU status
nvidia-smi

# Test Docker GPU support
docker run --rm --gpus all nvidia/cuda:latest nvidia-smi

# Check Kubernetes GPU resources
kubectl describe nodes | grep -i gpu

# View GPU device plugin logs
kubectl logs -n kube-system -l name=nvidia-device-plugin-ds
```

## Performance Optimization

### GPU Selection Strategy

The GPU tools implement intelligent GPU selection based on:

1. **Memory Requirements**: Select GPUs with sufficient memory
2. **Utilization**: Prefer less utilized GPUs
3. **Model Preference**: Prioritize specific GPU models
4. **Power Efficiency**: Consider power draw limits
5. **Workload Type**: Match GPU capabilities to workload needs

### Monitoring Best Practices

1. **Regular Monitoring**: Use continuous monitoring for production workloads
2. **Temperature Tracking**: Monitor GPU temperatures to prevent throttling
3. **Memory Usage**: Track GPU memory to prevent out-of-memory errors
4. **Utilization Analysis**: Optimize workload distribution across GPUs

## Integration with Existing Systems

The GPU connector seamlessly integrates with:

- **Coder Workspaces**: GPU-enabled development environments
- **CI/CD Pipelines**: GPU-accelerated build and test processes  
- **ML Platforms**: Integration with popular ML frameworks
- **Container Orchestration**: Full Docker and Kubernetes support
- **VM Management**: Proxmox GPU passthrough for virtual machines

## Future Enhancements

Planned improvements include:

- **AMD GPU Support**: ROCm and AMD GPU integration
- **Multi-Instance GPU (MIG)**: NVIDIA MIG support for A100/H100
- **GPU Scheduling**: Advanced scheduling algorithms
- **Cost Optimization**: GPU usage cost tracking and optimization
- **Auto-scaling**: Dynamic GPU resource scaling based on demand
