// Package agentic provides Docker and Podman connector for container orchestration.
package agentic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

// DockerConfig holds Docker/Podman configuration.
type DockerConfig struct {
	Engine   string `json:"engine"`   // "docker" or "podman"
	Host     string `json:"host"`     // Docker host (optional)
	TLS      bool   `json:"tls"`      // Use TLS
	CertPath string `json:"certpath"` // Certificate path for TLS
}

// DockerClient is an agent for Docker/Podman container orchestration tasks.
type DockerClient struct {
	cfg DockerConfig
}

// DockerTask represents a Docker orchestration task.
type DockerTask struct {
	Action        string                 `json:"action"`     // run, build, stop, rm, ps, logs, pull, push
	Image         string                 `json:"image"`      // Container image
	Name          string                 `json:"name"`       // Container name
	Command       []string               `json:"command"`    // Command to run
	Ports         []string               `json:"ports"`      // Port mappings (e.g., "8080:80")
	Volumes       []string               `json:"volumes"`    // Volume mounts (e.g., "/host:/container")
	Env           map[string]string      `json:"env"`        // Environment variables
	Network       string                 `json:"network"`    // Network name
	Labels        map[string]string      `json:"labels"`     // Container labels
	Remove        bool                   `json:"remove"`     // Remove container when it stops
	Detach        bool                   `json:"detach"`     // Run in background
	Config        map[string]interface{} `json:"config"`     // Additional configuration
	BuildPath     string                 `json:"buildpath"`  // Build context path
	DockerfileCmd string                 `json:"dockerfile"` // Dockerfile path
	Compose       *DockerComposeTask     `json:"compose"`    // Docker Compose task
	GPUs          *DockerGPUConfig       `json:"gpus"`       // GPU configuration
}

// DockerComposeTask represents a Docker Compose task.
type DockerComposeTask struct {
	File     string            `json:"file"`     // Compose file path
	Project  string            `json:"project"`  // Project name
	Services []string          `json:"services"` // Specific services
	EnvVars  map[string]string `json:"env"`      // Environment variables
	Profiles []string          `json:"profiles"` // Compose profiles
	Override []string          `json:"override"` // Override files
}

// DockerGPUConfig represents GPU configuration for Docker containers.
type DockerGPUConfig struct {
	Enabled      bool     `json:"enabled"`      // Enable GPU support
	Runtime      string   `json:"runtime"`      // GPU runtime (nvidia, runc)
	GPUIDs       []int    `json:"gpu_ids"`      // Specific GPU IDs
	All          bool     `json:"all"`          // Use all available GPUs
	Capabilities []string `json:"capabilities"` // GPU capabilities (compute, utility, graphics)
	Memory       string   `json:"memory"`       // GPU memory limit
}

// ContainerInfo represents container information.
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Status  string            `json:"status"`
	Ports   []string          `json:"ports"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels"`
}

// ImageInfo represents image information.
type ImageInfo struct {
	ID      string    `json:"id"`
	Tags    []string  `json:"tags"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
}

// NewDockerClient creates a new Docker/Podman client.
func NewDockerClient(cfg DockerConfig) *DockerClient {
	if cfg.Engine == "" {
		cfg.Engine = "docker" // Default to Docker
	}

	return &DockerClient{cfg: cfg}
}

func (d *DockerClient) Name() string { return "docker" }

func (d *DockerClient) Supports(taskType string) bool {
	return taskType == "container" || taskType == "docker" || taskType == "podman"
}

func (d *DockerClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Parse task payload
	var dockerTask DockerTask
	if err := mapToStruct(task.Payload, &dockerTask); err != nil {
		return &TaskResult{Error: xerrors.Errorf("invalid task payload: %w", err)}, nil
	}

	// Execute based on action
	switch dockerTask.Action {
	case "run":
		result, err := d.runContainer(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "build":
		result, err := d.buildImage(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "ps", "list":
		result, err := d.listContainers(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "images":
		result, err := d.listImages(ctx)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "stop":
		result, err := d.stopContainer(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "start":
		result, err := d.startContainer(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "rm", "remove":
		result, err := d.removeContainer(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "logs":
		result, err := d.getLogs(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "pull":
		result, err := d.pullImage(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "push":
		result, err := d.pushImage(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "compose":
		result, err := d.dockerCompose(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "inspect":
		result, err := d.inspectContainer(ctx, &dockerTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	default:
		return &TaskResult{Error: xerrors.Errorf("unsupported action: %s", dockerTask.Action)}, nil
	}
}

// runContainer runs a new container.
func (d *DockerClient) runContainer(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	args := []string{"run"}

	// Add options
	if task.Detach {
		args = append(args, "-d")
	}

	if task.Remove {
		args = append(args, "--rm")
	}

	if task.Name != "" {
		args = append(args, "--name", task.Name)
	}

	// Add port mappings
	for _, port := range task.Ports {
		args = append(args, "-p", port)
	}

	// Add volume mounts
	for _, volume := range task.Volumes {
		args = append(args, "-v", volume)
	}

	// Add environment variables
	for key, value := range task.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add labels
	for key, value := range task.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add network
	if task.Network != "" {
		args = append(args, "--network", task.Network)
	}

	// Add GPU support
	if task.GPUs != nil && task.GPUs.Enabled {
		if task.GPUs.All {
			args = append(args, "--gpus", "all")
		} else if len(task.GPUs.GPUIDs) > 0 {
			gpuDevices := make([]string, len(task.GPUs.GPUIDs))
			for i, id := range task.GPUs.GPUIDs {
				gpuDevices[i] = fmt.Sprintf("device=%d", id)
			}
			args = append(args, "--gpus", strings.Join(gpuDevices, ","))
		} else {
			args = append(args, "--gpus", "all")
		}

		// Add GPU runtime if specified
		if task.GPUs.Runtime != "" {
			args = append(args, "--runtime", task.GPUs.Runtime)
		}

		// Add GPU capabilities
		if len(task.GPUs.Capabilities) > 0 {
			caps := strings.Join(task.GPUs.Capabilities, ",")
			// Find and modify the --gpus argument to include capabilities
			for i, arg := range args {
				if arg == "--gpus" && i+1 < len(args) {
					args[i+1] = fmt.Sprintf("\"%s,capabilities=%s\"", args[i+1], caps)
					break
				}
			}
		}

		// Add GPU memory limit
		if task.GPUs.Memory != "" {
			// This would typically be handled via --gpus options or cgroup limits
			args = append(args, "--env", fmt.Sprintf("NVIDIA_MIG_CONFIG_DEVICES=%s", task.GPUs.Memory))
		}
	}

	// Add image
	args = append(args, task.Image)

	// Add command
	if len(task.Command) > 0 {
		args = append(args, task.Command...)
	}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to run container: %w", err)
	}

	result := map[string]interface{}{
		"action":       "run",
		"image":        task.Image,
		"name":         task.Name,
		"container_id": strings.TrimSpace(output),
	}

	return result, nil
}

// buildImage builds a Docker image.
func (d *DockerClient) buildImage(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	args := []string{"build"}

	if task.Name != "" {
		args = append(args, "-t", task.Name)
	}

	if task.DockerfileCmd != "" {
		args = append(args, "-f", task.DockerfileCmd)
	}

	// Add labels
	for key, value := range task.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add build context
	buildPath := task.BuildPath
	if buildPath == "" {
		buildPath = "."
	}
	args = append(args, buildPath)

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to build image: %w", err)
	}

	result := map[string]interface{}{
		"action": "build",
		"image":  task.Name,
		"output": output,
	}

	return result, nil
}

// listContainers lists containers.
func (d *DockerClient) listContainers(ctx context.Context, task *DockerTask) ([]ContainerInfo, error) {
	args := []string{"ps", "--format", "json"}

	// Show all containers if requested
	if task.Config != nil {
		if all, ok := task.Config["all"].(bool); ok && all {
			args = append(args, "-a")
		}
	}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to list containers: %w", err)
	}

	var containers []ContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var containerJSON map[string]interface{}
		if err := json.Unmarshal([]byte(line), &containerJSON); err != nil {
			continue // Skip malformed lines
		}

		container := ContainerInfo{
			ID:     getStringField(containerJSON, "ID"),
			Name:   getStringField(containerJSON, "Names"),
			Image:  getStringField(containerJSON, "Image"),
			Status: getStringField(containerJSON, "Status"),
		}

		if portsStr := getStringField(containerJSON, "Ports"); portsStr != "" {
			container.Ports = strings.Split(portsStr, ", ")
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// listImages lists Docker images.
func (d *DockerClient) listImages(ctx context.Context) ([]ImageInfo, error) {
	args := []string{"images", "--format", "json"}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to list images: %w", err)
	}

	var images []ImageInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var imageJSON map[string]interface{}
		if err := json.Unmarshal([]byte(line), &imageJSON); err != nil {
			continue // Skip malformed lines
		}

		image := ImageInfo{
			ID:   getStringField(imageJSON, "ID"),
			Tags: []string{getStringField(imageJSON, "Repository") + ":" + getStringField(imageJSON, "Tag")},
		}

		images = append(images, image)
	}

	return images, nil
}

// stopContainer stops a container.
func (d *DockerClient) stopContainer(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("container name is required")
	}

	args := []string{"stop", task.Name}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to stop container: %w", err)
	}

	result := map[string]interface{}{
		"action": "stop",
		"name":   task.Name,
		"output": strings.TrimSpace(output),
	}

	return result, nil
}

// startContainer starts a container.
func (d *DockerClient) startContainer(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("container name is required")
	}

	args := []string{"start", task.Name}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to start container: %w", err)
	}

	result := map[string]interface{}{
		"action": "start",
		"name":   task.Name,
		"output": strings.TrimSpace(output),
	}

	return result, nil
}

// removeContainer removes a container.
func (d *DockerClient) removeContainer(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("container name is required")
	}

	args := []string{"rm"}

	// Force removal if requested
	if task.Config != nil {
		if force, ok := task.Config["force"].(bool); ok && force {
			args = append(args, "-f")
		}
	}

	args = append(args, task.Name)

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to remove container: %w", err)
	}

	result := map[string]interface{}{
		"action": "remove",
		"name":   task.Name,
		"output": strings.TrimSpace(output),
	}

	return result, nil
}

// getLogs gets container logs.
func (d *DockerClient) getLogs(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("container name is required")
	}

	args := []string{"logs"}

	// Add options from config
	if task.Config != nil {
		if follow, ok := task.Config["follow"].(bool); ok && follow {
			args = append(args, "-f")
		}
		if tail, ok := task.Config["tail"].(string); ok && tail != "" {
			args = append(args, "--tail", tail)
		}
		if since, ok := task.Config["since"].(string); ok && since != "" {
			args = append(args, "--since", since)
		}
	}

	args = append(args, task.Name)

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to get logs: %w", err)
	}

	result := map[string]interface{}{
		"action": "logs",
		"name":   task.Name,
		"logs":   output,
	}

	return result, nil
}

// pullImage pulls a Docker image.
func (d *DockerClient) pullImage(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Image == "" {
		return nil, xerrors.New("image name is required")
	}

	args := []string{"pull", task.Image}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to pull image: %w", err)
	}

	result := map[string]interface{}{
		"action": "pull",
		"image":  task.Image,
		"output": output,
	}

	return result, nil
}

// pushImage pushes a Docker image.
func (d *DockerClient) pushImage(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Image == "" {
		return nil, xerrors.New("image name is required")
	}

	args := []string{"push", task.Image}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to push image: %w", err)
	}

	result := map[string]interface{}{
		"action": "push",
		"image":  task.Image,
		"output": output,
	}

	return result, nil
}

// dockerCompose executes Docker Compose commands.
func (d *DockerClient) dockerCompose(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Compose == nil {
		return nil, xerrors.New("compose configuration is required")
	}

	compose := task.Compose
	args := []string{"compose"}

	// Add compose file
	if compose.File != "" {
		args = append(args, "-f", compose.File)
	}

	// Add project name
	if compose.Project != "" {
		args = append(args, "-p", compose.Project)
	}

	// Add override files
	for _, override := range compose.Override {
		args = append(args, "-f", override)
	}

	// Add the compose action (from main task config)
	if task.Config != nil {
		if action, ok := task.Config["compose_action"].(string); ok {
			args = append(args, action)
		} else {
			args = append(args, "up", "-d") // Default action
		}
	} else {
		args = append(args, "up", "-d")
	}

	// Add specific services
	if len(compose.Services) > 0 {
		args = append(args, compose.Services...)
	}

	// Set environment variables
	env := make([]string, 0)
	for key, value := range compose.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	output, err := d.execCommandWithEnv(ctx, env, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute docker compose: %w", err)
	}

	result := map[string]interface{}{
		"action":  "compose",
		"project": compose.Project,
		"output":  output,
	}

	return result, nil
}

// inspectContainer inspects a container.
func (d *DockerClient) inspectContainer(ctx context.Context, task *DockerTask) (map[string]interface{}, error) {
	if task.Name == "" {
		return nil, xerrors.New("container name is required")
	}

	args := []string{"inspect", task.Name}

	output, err := d.execCommand(ctx, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to inspect container: %w", err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &inspectData); err != nil {
		return nil, xerrors.Errorf("failed to parse inspect output: %w", err)
	}

	result := map[string]interface{}{
		"action": "inspect",
		"name":   task.Name,
		"data":   inspectData,
	}

	return result, nil
}

// execCommand executes a Docker/Podman command.
func (d *DockerClient) execCommand(ctx context.Context, args ...string) (string, error) {
	return d.execCommandWithEnv(ctx, nil, args...)
}

// execCommandWithEnv executes a Docker/Podman command with environment variables.
func (d *DockerClient) execCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, d.cfg.Engine, args...)

	if len(env) > 0 {
		cmd.Env = append(cmd.Env, env...)
	}

	// Add Docker host if specified
	if d.cfg.Host != "" {
		cmd.Env = append(cmd.Env, "DOCKER_HOST="+d.cfg.Host)
	}

	// Add TLS configuration
	if d.cfg.TLS {
		cmd.Env = append(cmd.Env, "DOCKER_TLS_VERIFY=1")
		if d.cfg.CertPath != "" {
			cmd.Env = append(cmd.Env, "DOCKER_CERT_PATH="+d.cfg.CertPath)
		}
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", xerrors.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// getStringField safely gets a string field from a map.
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
