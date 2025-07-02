// Package agentic provides Kubernetes connector for cluster orchestration.
package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/xerrors"
)

// KubernetesConfig holds Kubernetes configuration.
type KubernetesConfig struct {
	Engine     string `json:"engine"`     // "kubectl", "microk8s", "k3s", "talosctl"
	Kubeconfig string `json:"kubeconfig"` // Path to kubeconfig file
	Context    string `json:"context"`    // Kubernetes context name
	Namespace  string `json:"namespace"`  // Default namespace
}

// KubernetesClient is an agent for Kubernetes cluster orchestration tasks.
type KubernetesClient struct {
	cfg KubernetesConfig
}

// KubernetesTask represents a Kubernetes orchestration task.
type KubernetesTask struct {
	Action    string                 `json:"action"`    // apply, delete, get, create, scale, logs, exec
	Resource  string                 `json:"resource"`  // pods, services, deployments, etc.
	Name      string                 `json:"name"`      // Resource name
	Namespace string                 `json:"namespace"` // Resource namespace
	Manifest  string                 `json:"manifest"`  // YAML/JSON manifest
	File      string                 `json:"file"`      // Manifest file path
	Labels    map[string]string      `json:"labels"`    // Label selectors
	Config    map[string]interface{} `json:"config"`    // Additional configuration
	Command   []string               `json:"command"`   // Command for exec
	Container string                 `json:"container"` // Container name for exec/logs
	Follow    bool                   `json:"follow"`    // Follow logs
	Replicas  int32                  `json:"replicas"`  // Number of replicas for scale
	GPUs      *K8sGPUConfig          `json:"gpus"`      // GPU configuration
}

// KubernetesResource represents a Kubernetes resource.
type KubernetesResource struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
	Status    string            `json:"status"`
	Age       string            `json:"age"`
	Ready     string            `json:"ready,omitempty"`
	Restarts  string            `json:"restarts,omitempty"`
	IP        string            `json:"ip,omitempty"`
}

// K8sGPUConfig represents GPU configuration for Kubernetes resources.
type K8sGPUConfig struct {
	Enabled      bool              `json:"enabled"`       // Enable GPU support
	ResourceName string            `json:"resource_name"` // GPU resource name (nvidia.com/gpu, amd.com/gpu)
	Limit        int               `json:"limit"`         // Number of GPUs to request
	Request      int               `json:"request"`       // Number of GPUs to request (minimum)
	NodeSelector map[string]string `json:"node_selector"` // Node selector for GPU nodes
	Tolerations  []K8sToleration   `json:"tolerations"`   // Tolerations for GPU nodes
	RuntimeClass string            `json:"runtime_class"` // Runtime class for GPU workloads
	DevicePlugin string            `json:"device_plugin"` // Device plugin type (nvidia, amd)
}

// K8sToleration represents a Kubernetes toleration.
type K8sToleration struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
}

// NewKubernetesClient creates a new Kubernetes client.
func NewKubernetesClient(cfg KubernetesConfig) *KubernetesClient {
	if cfg.Engine == "" {
		cfg.Engine = "kubectl" // Default to kubectl
	}

	return &KubernetesClient{cfg: cfg}
}

func (k *KubernetesClient) Name() string { return "kubernetes" }

func (k *KubernetesClient) Supports(taskType string) bool {
	return taskType == "kubernetes" || taskType == "k8s" || taskType == "cluster"
}

func (k *KubernetesClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Parse task payload
	var k8sTask KubernetesTask
	if err := mapToStruct(task.Payload, &k8sTask); err != nil {
		return &TaskResult{Error: xerrors.Errorf("invalid task payload: %w", err)}, nil
	}

	// Execute based on action
	switch k8sTask.Action {
	case "apply":
		result, err := k.applyManifest(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "delete":
		result, err := k.deleteResource(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "get", "list":
		result, err := k.getResources(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "create":
		result, err := k.createResource(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "scale":
		result, err := k.scaleResource(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "logs":
		result, err := k.getLogs(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "exec":
		result, err := k.execCommand(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "describe":
		result, err := k.describeResource(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "port-forward":
		result, err := k.portForward(ctx, &k8sTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "cluster-info":
		result, err := k.getClusterInfo(ctx)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	default:
		return &TaskResult{Error: xerrors.Errorf("unsupported action: %s", k8sTask.Action)}, nil
	}
}

// applyManifest applies a Kubernetes manifest.
func (k *KubernetesClient) applyManifest(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	args := []string{"apply"}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Apply from manifest string or file
	if task.Manifest != "" {
		args = append(args, "-f", "-")
	} else if task.File != "" {
		args = append(args, "-f", task.File)
	} else {
		return nil, xerrors.New("either manifest or file must be specified")
	}

	var input string
	if task.Manifest != "" {
		input = task.Manifest
	}

	output, err := k.runCommand(ctx, input, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to apply manifest: %w", err)
	}

	result := map[string]interface{}{
		"action": "apply",
		"output": output,
	}

	return result, nil
}

// deleteResource deletes a Kubernetes resource.
func (k *KubernetesClient) deleteResource(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	args := []string{"delete"}

	if task.Resource == "" {
		return nil, xerrors.New("resource type is required")
	}

	args = append(args, task.Resource)

	if task.Name != "" {
		args = append(args, task.Name)
	}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Add label selectors
	if len(task.Labels) > 0 {
		var labelSelectors []string
		for key, value := range task.Labels {
			labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "-l", strings.Join(labelSelectors, ","))
	}

	// Delete from file if specified
	if task.File != "" {
		args = append(args, "-f", task.File)
	}

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to delete resource: %w", err)
	}

	result := map[string]interface{}{
		"action":   "delete",
		"resource": task.Resource,
		"name":     task.Name,
		"output":   output,
	}

	return result, nil
}

// getResources gets Kubernetes resources.
func (k *KubernetesClient) getResources(ctx context.Context, task *KubernetesTask) ([]KubernetesResource, error) {
	args := []string{"get"}

	if task.Resource == "" {
		task.Resource = "pods" // Default to pods
	}

	args = append(args, task.Resource)

	if task.Name != "" {
		args = append(args, task.Name)
	}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Add label selectors
	if len(task.Labels) > 0 {
		var labelSelectors []string
		for key, value := range task.Labels {
			labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "-l", strings.Join(labelSelectors, ","))
	}

	// Output as JSON for easier parsing
	args = append(args, "-o", "json")

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get resources: %w", err)
	}

	// Parse JSON output
	var response struct {
		Items []map[string]interface{} `json:"items"`
	}

	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, xerrors.Errorf("failed to parse kubectl output: %w", err)
	}

	var resources []KubernetesResource
	for _, item := range response.Items {
		resource := KubernetesResource{
			Kind:      getStringField(item, "kind"),
			Name:      getNestedStringField(item, "metadata", "name"),
			Namespace: getNestedStringField(item, "metadata", "namespace"),
		}

		// Extract labels
		if labels, ok := getNestedField(item, "metadata", "labels").(map[string]interface{}); ok {
			resource.Labels = make(map[string]string)
			for k, v := range labels {
				if str, ok := v.(string); ok {
					resource.Labels[k] = str
				}
			}
		}

		// Extract status based on resource type
		if status := getNestedStringField(item, "status", "phase"); status != "" {
			resource.Status = status
		} else if conditions := getNestedField(item, "status", "conditions"); conditions != nil {
			// For deployments and other resources with conditions
			resource.Status = "Unknown"
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// createResource creates a Kubernetes resource.
func (k *KubernetesClient) createResource(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	args := []string{"create"}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Create from manifest string or file
	if task.Manifest != "" {
		args = append(args, "-f", "-")
	} else if task.File != "" {
		args = append(args, "-f", task.File)
	} else {
		return nil, xerrors.New("either manifest or file must be specified")
	}

	var input string
	if task.Manifest != "" {
		input = task.Manifest
	}

	output, err := k.runCommand(ctx, input, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to create resource: %w", err)
	}

	result := map[string]interface{}{
		"action": "create",
		"output": output,
	}

	return result, nil
}

// scaleResource scales a Kubernetes resource.
func (k *KubernetesClient) scaleResource(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	if task.Resource == "" || task.Name == "" {
		return nil, xerrors.New("resource type and name are required")
	}

	args := []string{"scale", task.Resource, task.Name}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Add replicas
	args = append(args, fmt.Sprintf("--replicas=%d", task.Replicas))

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to scale resource: %w", err)
	}

	result := map[string]interface{}{
		"action":   "scale",
		"resource": task.Resource,
		"name":     task.Name,
		"replicas": task.Replicas,
		"output":   output,
	}

	return result, nil
}

// getLogs gets logs from a pod.
func (k *KubernetesClient) getLogs(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("pod name is required")
	}

	args := []string{"logs", task.Name}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Add container if specified
	if task.Container != "" {
		args = append(args, "-c", task.Container)
	}

	// Follow logs if requested
	if task.Follow {
		args = append(args, "-f")
	}

	// Add additional options from config
	if task.Config != nil {
		if tail, ok := task.Config["tail"].(string); ok && tail != "" {
			args = append(args, "--tail", tail)
		}
		if since, ok := task.Config["since"].(string); ok && since != "" {
			args = append(args, "--since", since)
		}
	}

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get logs: %w", err)
	}

	result := map[string]interface{}{
		"action":    "logs",
		"pod":       task.Name,
		"container": task.Container,
		"logs":      output,
	}

	return result, nil
}

// execCommand executes a command in a pod.
func (k *KubernetesClient) execCommand(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("pod name is required")
	}

	if len(task.Command) == 0 {
		return nil, xerrors.New("command is required")
	}

	args := []string{"exec", task.Name}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	// Add container if specified
	if task.Container != "" {
		args = append(args, "-c", task.Container)
	}

	// Add interactive and tty flags if requested
	if task.Config != nil {
		if interactive, ok := task.Config["interactive"].(bool); ok && interactive {
			args = append(args, "-i")
		}
		if tty, ok := task.Config["tty"].(bool); ok && tty {
			args = append(args, "-t")
		}
	}

	// Add -- separator and command
	args = append(args, "--")
	args = append(args, task.Command...)

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to exec command: %w", err)
	}

	result := map[string]interface{}{
		"action":    "exec",
		"pod":       task.Name,
		"container": task.Container,
		"command":   task.Command,
		"output":    output,
	}

	return result, nil
}

// describeResource describes a Kubernetes resource.
func (k *KubernetesClient) describeResource(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	if task.Resource == "" || task.Name == "" {
		return nil, xerrors.New("resource type and name are required")
	}

	args := []string{"describe", task.Resource, task.Name}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to describe resource: %w", err)
	}

	result := map[string]interface{}{
		"action":      "describe",
		"resource":    task.Resource,
		"name":        task.Name,
		"description": output,
	}

	return result, nil
}

// portForward forwards ports from a pod.
func (k *KubernetesClient) portForward(ctx context.Context, task *KubernetesTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("pod name is required")
	}

	if task.Config == nil || task.Config["ports"] == nil {
		return nil, xerrors.New("ports configuration is required")
	}

	ports, ok := task.Config["ports"].(string)
	if !ok {
		return nil, xerrors.New("ports must be a string (e.g., '8080:80')")
	}

	args := []string{"port-forward", task.Name, ports}

	// Add namespace if specified
	if task.Namespace != "" {
		args = append(args, "-n", task.Namespace)
	} else if k.cfg.Namespace != "" {
		args = append(args, "-n", k.cfg.Namespace)
	}

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to port-forward: %w", err)
	}

	result := map[string]interface{}{
		"action": "port-forward",
		"pod":    task.Name,
		"ports":  ports,
		"output": output,
	}

	return result, nil
}

// getClusterInfo gets cluster information.
func (k *KubernetesClient) getClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	args := []string{"cluster-info"}

	output, err := k.runCommand(ctx, "", args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get cluster info: %w", err)
	}

	result := map[string]interface{}{
		"action":       "cluster-info",
		"cluster_info": output,
	}

	return result, nil
}

// runCommand runs a kubectl/k8s command.
func (k *KubernetesClient) runCommand(ctx context.Context, input string, args ...string) (string, error) {
	var cmdName string
	var cmdArgs []string

	switch k.cfg.Engine {
	case "microk8s":
		cmdName = "microk8s"
		cmdArgs = append([]string{"kubectl"}, args...)
	case "k3s":
		cmdName = "k3s"
		cmdArgs = append([]string{"kubectl"}, args...)
	case "talosctl":
		// For Talos, we need to use talosctl with different commands
		cmdName = "talosctl"
		cmdArgs = k.convertToTalosCommand(args)
	default:
		cmdName = "kubectl"
		cmdArgs = args
	}

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	// Set kubeconfig if specified
	if k.cfg.Kubeconfig != "" && k.cfg.Engine != "talosctl" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+k.cfg.Kubeconfig)
	}

	// Set kubectl context if specified
	if k.cfg.Context != "" && k.cfg.Engine != "talosctl" {
		cmd.Args = append(cmd.Args, "--context", k.cfg.Context)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set input if provided
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	err := cmd.Run()
	if err != nil {
		return "", xerrors.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// convertToTalosCommand converts kubectl commands to talosctl equivalents.
func (k *KubernetesClient) convertToTalosCommand(args []string) []string {
	if len(args) == 0 {
		return args
	}

	// For now, we'll pass through kubectl commands to talosctl's kubectl
	// In a full implementation, you'd convert to native talosctl commands
	return append([]string{"kubectl"}, args...)
}

// getNestedField safely gets a nested field from a map.
func getNestedField(m map[string]interface{}, keys ...string) interface{} {
	current := m
	for i, key := range keys {
		if val, ok := current[key]; ok {
			if i == len(keys)-1 {
				return val
			}
			if next, ok := val.(map[string]interface{}); ok {
				current = next
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return nil
}

// getNestedStringField safely gets a nested string field from a map.
func getNestedStringField(m map[string]interface{}, keys ...string) string {
	if val := getNestedField(m, keys...); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
