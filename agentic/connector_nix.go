// Package agentic provides a Nix/NixOS connector for reproducible infrastructure management.
package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

// NixClient is an agent for Nix/NixOS reproducible infrastructure tasks.
type NixClient struct {
	cfg NixConfig
}

// NixTask represents a Nix/NixOS orchestration task.
type NixTask struct {
	Action      string                 `json:"action"`      // build, shell, install, develop, rebuild, deploy, flake
	Expression  string                 `json:"expression"`  // Nix expression or flake reference
	Derivation  string                 `json:"derivation"`  // .drv file path
	FlakeRef    string                 `json:"flake_ref"`   // Flake reference (e.g., github:owner/repo)
	Attribute   string                 `json:"attribute"`   // Attribute to build (e.g., packages.x86_64-linux.hello)
	System      string                 `json:"system"`      // Target system (x86_64-linux, aarch64-linux, etc.)
	Profile     string                 `json:"profile"`     // Nix profile path
	Generation  int                    `json:"generation"`  // Specific generation number
	Config      map[string]interface{} `json:"config"`      // Additional configuration
	Environment map[string]string      `json:"environment"` // Environment variables
	Args        []string               `json:"args"`        // Additional command arguments
	WorkDir     string                 `json:"workdir"`     // Working directory
	Remote      *NixRemoteConfig       `json:"remote"`      // Remote execution config
	Nixpkgs     string                 `json:"nixpkgs"`     // Nixpkgs channel or path
}

// NixRemoteConfig represents remote execution configuration.
type NixRemoteConfig struct {
	Host       string `json:"host"`
	User       string `json:"user"`
	SSHKey     string `json:"ssh_key"`
	SystemType string `json:"system_type"`
}

// NixBuildResult represents the result of a Nix build.
type NixBuildResult struct {
	StorePath  string            `json:"store_path"`
	DrvPath    string            `json:"drv_path"`
	Outputs    map[string]string `json:"outputs"`
	BuildTime  time.Duration     `json:"build_time"`
	CacheHit   bool              `json:"cache_hit"`
	SystemType string            `json:"system_type"`
}

// NixSystemInfo represents NixOS system information.
type NixSystemInfo struct {
	Generation    int               `json:"generation"`
	ConfigPath    string            `json:"config_path"`
	KernelVersion string            `json:"kernel_version"`
	SystemVersion string            `json:"system_version"`
	Packages      map[string]string `json:"packages"`
	Services      []string          `json:"services"`
}

// FlakeInfo represents Nix flake information.
type FlakeInfo struct {
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	Locked      bool                   `json:"locked"`
	Outputs     map[string]interface{} `json:"outputs"`
	Inputs      map[string]interface{} `json:"inputs"`
}

// NewNixClient creates a new Nix/NixOS client.
func NewNixClient(cfg NixConfig) *NixClient {
	// Set sensible defaults
	if !cfg.FlakesEnabled {
		// Check if flakes are already enabled
		cfg.FlakesEnabled = isFlakesEnabled()
	}

	if len(cfg.Substitutes) == 0 {
		cfg.Substitutes = []string{"https://cache.nixos.org/"}
	}

	return &NixClient{cfg: cfg}
}

func (n *NixClient) Name() string { return "nix" }

func (n *NixClient) Supports(taskType string) bool {
	return taskType == "nix" || taskType == "nixos" || taskType == "flake" || taskType == "reproducible"
}

func (n *NixClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Parse task payload
	var nixTask NixTask
	if err := mapToStruct(task.Payload, &nixTask); err != nil {
		return &TaskResult{Error: xerrors.Errorf("invalid task payload: %w", err)}, nil
	}

	// Execute based on action
	switch nixTask.Action {
	case "build":
		result, err := n.buildExpression(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "shell":
		result, err := n.nixShell(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "install":
		result, err := n.installPackage(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "develop":
		result, err := n.nixDevelop(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "rebuild":
		result, err := n.nixosRebuild(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "deploy":
		result, err := n.deployToRemote(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "flake":
		result, err := n.flakeOperation(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "gc":
		result, err := n.garbageCollect(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "search":
		result, err := n.searchPackages(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "info":
		result, err := n.systemInfo(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "store":
		result, err := n.storeOperation(ctx, &nixTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	default:
		return &TaskResult{Error: xerrors.Errorf("unsupported action: %s", nixTask.Action)}, nil
	}
}

// buildExpression builds a Nix expression or flake.
func (n *NixClient) buildExpression(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	startTime := time.Now()

	var args []string
	if task.FlakeRef != "" {
		// Build flake reference
		args = []string{"build"}
		if n.cfg.FlakesEnabled {
			args = append(args, "--experimental-features", "nix-command flakes")
		}
		args = append(args, task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	} else if task.Expression != "" {
		// Build expression
		args = []string{"build", "-E", task.Expression}
	} else {
		return nil, xerrors.New("either expression or flake_ref must be specified")
	}

	// Add system specification
	if task.System != "" {
		args = append(args, "--system", task.System)
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to build expression: %w", err)
	}

	buildTime := time.Since(startTime)

	// Parse output paths
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var storePaths []string
	for _, line := range lines {
		if strings.HasPrefix(line, "/nix/store/") {
			storePaths = append(storePaths, line)
		}
	}

	result := map[string]interface{}{
		"action":      "build",
		"store_paths": storePaths,
		"build_time":  buildTime.String(),
		"system":      task.System,
		"output":      output,
	}

	return result, nil
}

// nixShell creates a Nix shell environment.
func (n *NixClient) nixShell(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	var args []string

	if task.FlakeRef != "" {
		// Use flake for shell
		args = []string{"shell"}
		if n.cfg.FlakesEnabled {
			args = append(args, "--experimental-features", "nix-command flakes")
		}
		args = append(args, task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	} else if task.Expression != "" {
		// Use expression for shell
		args = []string{"shell", "-E", task.Expression}
	} else {
		return nil, xerrors.New("either expression or flake_ref must be specified")
	}

	// Add command to run in shell
	if task.Config != nil {
		if command, ok := task.Config["command"].(string); ok && command != "" {
			args = append(args, "--command", command)
		}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to create nix shell: %w", err)
	}

	result := map[string]interface{}{
		"action": "shell",
		"output": output,
	}

	return result, nil
}

// installPackage installs a package using nix-env or nix profile.
func (n *NixClient) installPackage(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	var args []string

	if task.FlakeRef != "" && n.cfg.FlakesEnabled {
		// Use nix profile for flakes
		args = []string{"profile", "install"}
		args = append(args, "--experimental-features", "nix-command flakes")
		args = append(args, task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	} else {
		// Use nix-env for traditional packages
		args = []string{"env", "-i"}
		if task.Expression != "" {
			args = append(args, "-E", task.Expression)
		} else if task.Attribute != "" {
			args = append(args, "-A", task.Attribute)
		} else {
			return nil, xerrors.New("package name, expression, or flake reference required")
		}
	}

	// Add profile if specified
	if task.Profile != "" {
		args = append(args, "--profile", task.Profile)
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to install package: %w", err)
	}

	result := map[string]interface{}{
		"action":  "install",
		"package": task.Attribute,
		"output":  output,
	}

	return result, nil
}

// nixDevelop creates a development environment.
func (n *NixClient) nixDevelop(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	args := []string{"develop"}

	if n.cfg.FlakesEnabled {
		args = append(args, "--experimental-features", "nix-command flakes")
	}

	if task.FlakeRef != "" {
		args = append(args, task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	}

	// Add command to run
	if task.Config != nil {
		if command, ok := task.Config["command"].(string); ok && command != "" {
			args = append(args, "--command", command)
		}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to create development environment: %w", err)
	}

	result := map[string]interface{}{
		"action": "develop",
		"output": output,
	}

	return result, nil
}

// nixosRebuild rebuilds NixOS system configuration.
func (n *NixClient) nixosRebuild(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	// Default to 'switch' if no specific action
	action := "switch"
	if task.Config != nil {
		if rebuildAction, ok := task.Config["rebuild_action"].(string); ok {
			action = rebuildAction
		}
	}

	args := []string{"rebuild", action}

	// Add flake reference if specified
	if task.FlakeRef != "" {
		args = append(args, "--flake", task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	}

	// Add target host for remote rebuilds
	if task.Remote != nil {
		args = append(args, "--target-host", fmt.Sprintf("%s@%s", task.Remote.User, task.Remote.Host))
		if task.Remote.SSHKey != "" {
			// Set SSH key via environment or config
			task.Environment = mergeEnvMaps(task.Environment, map[string]string{
				"NIX_SSHOPTS": fmt.Sprintf("-i %s", task.Remote.SSHKey),
			})
		}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixOSCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to rebuild NixOS: %w", err)
	}

	result := map[string]interface{}{
		"action":         "rebuild",
		"rebuild_action": action,
		"output":         output,
	}

	return result, nil
}

// deployToRemote deploys configuration to remote NixOS machine.
func (n *NixClient) deployToRemote(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	if task.Remote == nil {
		return nil, xerrors.New("remote configuration is required for deployment")
	}

	// Use nixos-rebuild with target-host
	args := []string{"rebuild", "switch"}
	args = append(args, "--target-host", fmt.Sprintf("%s@%s", task.Remote.User, task.Remote.Host))

	if task.FlakeRef != "" {
		args = append(args, "--flake", task.FlakeRef)
		if task.Attribute != "" {
			args[len(args)-1] = fmt.Sprintf("%s#%s", task.FlakeRef, task.Attribute)
		}
	}

	// Set SSH options
	if task.Remote.SSHKey != "" {
		task.Environment = mergeEnvMaps(task.Environment, map[string]string{
			"NIX_SSHOPTS": fmt.Sprintf("-i %s -o StrictHostKeyChecking=no", task.Remote.SSHKey),
		})
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixOSCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to deploy to remote: %w", err)
	}

	result := map[string]interface{}{
		"action": "deploy",
		"target": fmt.Sprintf("%s@%s", task.Remote.User, task.Remote.Host),
		"output": output,
	}

	return result, nil
}

// flakeOperation performs flake-specific operations.
func (n *NixClient) flakeOperation(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	if !n.cfg.FlakesEnabled {
		return nil, xerrors.New("flakes are not enabled")
	}

	flakeAction := "show"
	if task.Config != nil {
		if action, ok := task.Config["flake_action"].(string); ok {
			flakeAction = action
		}
	}

	args := []string{"flake", flakeAction}
	args = append(args, "--experimental-features", "nix-command flakes")

	if task.FlakeRef != "" {
		args = append(args, task.FlakeRef)
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute flake operation: %w", err)
	}

	result := map[string]interface{}{
		"action":       "flake",
		"flake_action": flakeAction,
		"flake_ref":    task.FlakeRef,
		"output":       output,
	}

	// Try to parse as JSON for certain operations
	if flakeAction == "show" || flakeAction == "metadata" {
		var flakeInfo map[string]interface{}
		if err := json.Unmarshal([]byte(output), &flakeInfo); err == nil {
			result["parsed_output"] = flakeInfo
		}
	}

	return result, nil
}

// garbageCollect performs Nix store garbage collection.
func (n *NixClient) garbageCollect(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	args := []string{"store", "gc"}

	// Add deletion options
	if task.Config != nil {
		if maxAge, ok := task.Config["max_age"].(string); ok && maxAge != "" {
			args = append(args, "--max-age", maxAge)
		}
		if dryRun, ok := task.Config["dry_run"].(bool); ok && dryRun {
			args = append(args, "--dry-run")
		}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to garbage collect: %w", err)
	}

	result := map[string]interface{}{
		"action": "gc",
		"output": output,
	}

	return result, nil
}

// searchPackages searches for packages in nixpkgs.
func (n *NixClient) searchPackages(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	if task.Expression == "" {
		return nil, xerrors.New("search query is required in expression field")
	}

	args := []string{"search"}

	if n.cfg.FlakesEnabled {
		args = append(args, "--experimental-features", "nix-command flakes")
		// Search in nixpkgs flake by default
		nixpkgs := task.Nixpkgs
		if nixpkgs == "" {
			nixpkgs = "nixpkgs"
		}
		args = append(args, nixpkgs, task.Expression)
	} else {
		// Use nix-env for search
		args = []string{"env", "-qa", fmt.Sprintf(".*%s.*", task.Expression)}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to search packages: %w", err)
	}

	// Parse search results
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var packages []string
	for _, line := range lines {
		if line != "" {
			packages = append(packages, strings.TrimSpace(line))
		}
	}

	result := map[string]interface{}{
		"action":   "search",
		"query":    task.Expression,
		"packages": packages,
		"output":   output,
	}

	return result, nil
}

// systemInfo gets NixOS system information.
func (n *NixClient) systemInfo(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get current generation
	output, err := n.execCommand(ctx, task, "nix-env", "--list-generations", "--profile", "/nix/var/nix/profiles/system")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			// Parse current generation from last line
			lastLine := lines[len(lines)-1]
			if strings.Contains(lastLine, "(current)") {
				parts := strings.Fields(lastLine)
				if len(parts) > 0 {
					result["current_generation"] = parts[0]
				}
			}
		}
		result["generations"] = lines
	}

	// Get nixos-version
	if output, err := n.execCommand(ctx, task, "nixos-version"); err == nil {
		result["nixos_version"] = strings.TrimSpace(output)
	}

	// Get kernel version
	if output, err := n.execCommand(ctx, task, "uname", "-r"); err == nil {
		result["kernel_version"] = strings.TrimSpace(output)
	}

	// Get store info
	if output, err := n.execNixCommand(ctx, task, "store", "info"); err == nil {
		result["store_info"] = strings.TrimSpace(output)
	}

	result["action"] = "info"
	return result, nil
}

// storeOperation performs Nix store operations.
func (n *NixClient) storeOperation(ctx context.Context, task *NixTask) (map[string]interface{}, error) {
	storeAction := "info"
	if task.Config != nil {
		if action, ok := task.Config["store_action"].(string); ok {
			storeAction = action
		}
	}

	args := []string{"store", storeAction}

	// Add store path if specified
	if task.Config != nil {
		if storePath, ok := task.Config["store_path"].(string); ok && storePath != "" {
			args = append(args, storePath)
		}
	}

	// Add extra arguments
	args = append(args, task.Args...)

	output, err := n.execNixCommand(ctx, task, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute store operation: %w", err)
	}

	result := map[string]interface{}{
		"action":       "store",
		"store_action": storeAction,
		"output":       output,
	}

	return result, nil
}

// execNixCommand executes a nix command.
func (n *NixClient) execNixCommand(ctx context.Context, task *NixTask, args ...string) (string, error) {
	return n.execCommand(ctx, task, "nix", args...)
}

// execNixOSCommand executes a nixos-rebuild command.
func (n *NixClient) execNixOSCommand(ctx context.Context, task *NixTask, args ...string) (string, error) {
	return n.execCommand(ctx, task, "nixos-rebuild", args...)
}

// execCommand executes a command with proper environment setup.
func (n *NixClient) execCommand(ctx context.Context, task *NixTask, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory
	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	// Setup environment
	env := os.Environ()

	// Add NIX_PATH if configured
	if n.cfg.NixPath != "" {
		env = append(env, fmt.Sprintf("NIX_PATH=%s", n.cfg.NixPath))
	}

	// Add custom environment variables
	for key, value := range task.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add signing key if configured
	if n.cfg.SigningKey != "" {
		env = append(env, fmt.Sprintf("NIX_SECRET_KEY_FILE=%s", n.cfg.SigningKey))
	}

	cmd.Env = env

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", xerrors.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// isFlakesEnabled checks if Nix flakes are enabled.
func isFlakesEnabled() bool {
	// Check nix.conf for experimental-features
	nixConf := "/etc/nix/nix.conf"
	if _, err := os.Stat(nixConf); err == nil {
		content, err := os.ReadFile(nixConf)
		if err == nil {
			return strings.Contains(string(content), "experimental-features") &&
				strings.Contains(string(content), "flakes")
		}
	}

	// Check user config
	home, err := os.UserHomeDir()
	if err == nil {
		userNixConf := filepath.Join(home, ".config", "nix", "nix.conf")
		if content, err := os.ReadFile(userNixConf); err == nil {
			return strings.Contains(string(content), "experimental-features") &&
				strings.Contains(string(content), "flakes")
		}
	}

	return false
}

// mergeEnvMaps merges two environment variable maps.
func mergeEnvMaps(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy base map
	for k, v := range base {
		result[k] = v
	}

	// Override with new values
	for k, v := range override {
		result[k] = v
	}

	return result
}
