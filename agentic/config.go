// Package agentic provides configuration for agentic connectors.
package agentic

import (
	"strings"
)

// Config holds API keys and model selection for all connectors.
type Config struct {
	HuggingFace    HFConfig
	IOIntelligence IOIConfig
	OpenCode       OpenCodeConfig
	AgentZero      AgentZeroConfig
	DefaultLLM     string // "huggingface" or "io_intelligence" or "opencode"
	DefaultEmbed   string // "huggingface" or "io_intelligence" or "opencode"

	// Authentication configuration
	Auth AuthConfig `json:"auth" yaml:"auth"`

	// Secrets management configuration
	Secrets SecretsConfig `json:"secrets" yaml:"secrets"`

	// Infrastructure configuration
	Infrastructure InfrastructureConfig `json:"infrastructure" yaml:"infrastructure"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	GitHub GitHubAuthConfig `json:"github" yaml:"github"`
}

// SecretsConfig holds secrets management configuration.
type SecretsConfig struct {
	// Primary secret store type: "env", "file", "vault", "k8s"
	Provider string `json:"provider" yaml:"provider"`

	// File-based secrets configuration
	File FileSecretsConfig `json:"file" yaml:"file"`

	// Vault configuration (for future implementation)
	Vault VaultConfig `json:"vault" yaml:"vault"`

	// Kubernetes secrets configuration (for future implementation)
	Kubernetes K8sSecretsConfig `json:"kubernetes" yaml:"kubernetes"`
}

// FileSecretsConfig holds file-based secrets configuration.
type FileSecretsConfig struct {
	Path     string `json:"path" yaml:"path"`
	Password string `json:"password" yaml:"password"` // Should be set via env var
}

// VaultConfig holds HashiCorp Vault configuration.
type VaultConfig struct {
	Address string `json:"address" yaml:"address"`
	Token   string `json:"token" yaml:"token"`
	Path    string `json:"path" yaml:"path"`
}

// K8sSecretsConfig holds Kubernetes secrets configuration.
type K8sSecretsConfig struct {
	Namespace string `json:"namespace" yaml:"namespace"`
	Name      string `json:"name" yaml:"name"`
}

// NixConfig holds Nix/NixOS configuration.
type NixConfig struct {
	NixPath       string            `json:"nix_path"`       // Custom NIX_PATH
	RemoteBuilds  bool              `json:"remote_builds"`  // Enable remote builds
	Substitutes   []string          `json:"substitutes"`    // Binary cache substitutes
	TrustedKeys   []string          `json:"trusted_keys"`   // Trusted public keys for cache
	FlakesEnabled bool              `json:"flakes_enabled"` // Enable flakes support
	RemoteHosts   []NixRemoteHost   `json:"remote_hosts"`   // Remote NixOS machines
	SigningKey    string            `json:"signing_key"`    // Store signing key path
	ExtraConfig   map[string]string `json:"extra_config"`   // Additional nix.conf settings
}

// NixRemoteHost represents a remote NixOS machine.
type NixRemoteHost struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	User       string `json:"user"`
	SSHKey     string `json:"ssh_key"`
	SystemType string `json:"system_type"` // x86_64-linux, aarch64-linux, etc.
	MaxJobs    int    `json:"max_jobs"`
}

// InfrastructureConfig holds infrastructure connector configurations.
type InfrastructureConfig struct {
	Proxmox    ProxmoxConfig    `json:"proxmox" yaml:"proxmox"`
	Docker     DockerConfig     `json:"docker" yaml:"docker"`
	Kubernetes KubernetesConfig `json:"kubernetes" yaml:"kubernetes"`
	Nix        NixConfig        `json:"nix" yaml:"nix"`
	GPU        GPUConfig        `json:"gpu" yaml:"gpu"`
}

// DefaultConfig returns a default configuration with secure defaults.
func DefaultConfig() *Config {
	return &Config{
		DefaultLLM:   "huggingface",
		DefaultEmbed: "huggingface",
		OpenCode:     OpenCodeConfig{},
		AgentZero:    AgentZeroConfig{},
		Auth: AuthConfig{
			GitHub: GitHubAuthConfig{
				Scopes: []string{"user:email", "read:user"},
			},
		},
		Secrets: SecretsConfig{
			Provider: "env", // Default to environment variables
			File: FileSecretsConfig{
				Path: ".agentic/secrets.json",
			},
		},
		Infrastructure: InfrastructureConfig{
			Docker: DockerConfig{
				Engine: "docker",
			},
			Kubernetes: KubernetesConfig{
				Engine: "kubectl",
			},
			GPU: GPUConfig{
				NvidiaSMIPath:    "nvidia-smi",
				DockerRuntime:    "nvidia",
				EnableMonitoring: false,
				MaxGPUsPerTask:   8,
			},
		},
	}
}

/*
 * LoadFromSecrets populates configuration values from the secret manager.
 * Also loads OpenCode and Agent-Zero config.
 */
func (c *Config) LoadFromSecrets(secretManager *SecretManager) error {
	// Load GitHub OAuth credentials
	if clientID, err := secretManager.Get("GITHUB_CLIENT_ID"); err == nil {
		c.Auth.GitHub.ClientID = clientID
	}

	if clientSecret, err := secretManager.Get("GITHUB_CLIENT_SECRET"); err == nil {
		c.Auth.GitHub.ClientSecret = clientSecret
	}

	if redirectURL, err := secretManager.Get("GITHUB_REDIRECT_URL"); err == nil {
		c.Auth.GitHub.RedirectURL = redirectURL
	}

	// Load HuggingFace API key
	if apiKey, err := secretManager.Get("HUGGINGFACE_API_KEY"); err == nil {
		c.HuggingFace.APIKey = apiKey
	}

	// Load IOI API key
	if apiKey, err := secretManager.Get("IO_INTELLIGENCE_API_KEY"); err == nil {
		c.IOIntelligence.APIKey = apiKey
	}

	// Load OpenCode config
	if apiKey, err := secretManager.Get("OPENCODE_API_KEY"); err == nil {
		c.OpenCode.APIKey = apiKey
	}
	if endpoint, err := secretManager.Get("OPENCODE_ENDPOINT"); err == nil {
		c.OpenCode.Endpoint = endpoint
	}
	if wsurl, err := secretManager.Get("OPENCODE_WS_URL"); err == nil {
		c.OpenCode.WSURL = wsurl
	}

	// Load Agent-Zero config
	if endpoint, err := secretManager.Get("AGENTZERO_ENDPOINT"); err == nil {
		c.AgentZero.Endpoint = endpoint
	}
	if apiKey, err := secretManager.Get("AGENTZERO_API_KEY"); err == nil {
		c.AgentZero.APIKey = apiKey
	}

	// Load infrastructure credentials
	if proxmoxURL, err := secretManager.Get("PROXMOX_URL"); err == nil {
		c.Infrastructure.Proxmox.URL = proxmoxURL
	}

	if proxmoxUsername, err := secretManager.Get("PROXMOX_USERNAME"); err == nil {
		c.Infrastructure.Proxmox.Username = proxmoxUsername
	}

	if proxmoxPassword, err := secretManager.Get("PROXMOX_PASSWORD"); err == nil {
		c.Infrastructure.Proxmox.Password = proxmoxPassword
	}

	if proxmoxToken, err := secretManager.Get("PROXMOX_TOKEN"); err == nil {
		c.Infrastructure.Proxmox.Token = proxmoxToken
	}

	if proxmoxNode, err := secretManager.Get("PROXMOX_NODE"); err == nil {
		c.Infrastructure.Proxmox.Node = proxmoxNode
	}

	// Load Docker configuration
	if dockerHost, err := secretManager.Get("DOCKER_HOST"); err == nil {
		c.Infrastructure.Docker.Host = dockerHost
	}

	if certPath, err := secretManager.Get("DOCKER_CERT_PATH"); err == nil {
		c.Infrastructure.Docker.CertPath = certPath
	}

	// Load Kubernetes configuration
	if kubeconfig, err := secretManager.Get("KUBECONFIG"); err == nil {
		c.Infrastructure.Kubernetes.Kubeconfig = kubeconfig
	}

	if k8sContext, err := secretManager.Get("KUBE_CONTEXT"); err == nil {
		c.Infrastructure.Kubernetes.Context = k8sContext
	}

	if k8sNamespace, err := secretManager.Get("KUBE_NAMESPACE"); err == nil {
		c.Infrastructure.Kubernetes.Namespace = k8sNamespace
	}

	// Load Nix configuration
	if nixPath, err := secretManager.Get("NIX_PATH"); err == nil {
		c.Infrastructure.Nix.NixPath = nixPath
	}

	if signingKey, err := secretManager.Get("NIX_SIGNING_KEY"); err == nil {
		c.Infrastructure.Nix.SigningKey = signingKey
	}

	if substitutes, err := secretManager.Get("NIX_SUBSTITUTES"); err == nil {
		c.Infrastructure.Nix.Substitutes = strings.Split(substitutes, ",")
	}

	if trustedKeys, err := secretManager.Get("NIX_TRUSTED_KEYS"); err == nil {
		c.Infrastructure.Nix.TrustedKeys = strings.Split(trustedKeys, ",")
	}

	return nil
}
