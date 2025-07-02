// Package agentic provides Nix/NixOS infrastructure examples.
package agentic

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/xerrors"
)

// NixExampleConfig demonstrates Nix/NixOS workflow configuration.
type NixExampleConfig struct {
	FlakeRef    string `json:"flake_ref"`
	RemoteHost  string `json:"remote_host"`
	RemoteUser  string `json:"remote_user"`
	SSHKeyPath  string `json:"ssh_key_path"`
	Environment string `json:"environment"` // development, staging, production
}

// RunNixExamples demonstrates various Nix/NixOS operations.
func RunNixExamples(ctx context.Context, orchestrator *Orchestrator) error {
	log.Println("=== Nix/NixOS Infrastructure Examples ===")

	// Example 1: Build a simple package
	log.Println("\n1. Building a Nix package...")
	buildTask := &Task{
		Type: "nix",
		Payload: map[string]interface{}{
			"action":    "build",
			"flake_ref": "nixpkgs#hello",
			"system":    "x86_64-linux",
		},
	}

	result, err := orchestrator.ExecuteTask(ctx, buildTask)
	if err != nil {
		log.Printf("Build failed: %v", err)
	} else {
		log.Printf("Build result: %+v", result.Output)
	}

	// Example 2: Create a development environment
	log.Println("\n2. Creating a development environment...")
	devTask := &Task{
		Type: "nix",
		Payload: map[string]interface{}{
			"action":    "develop",
			"flake_ref": "github:nixos/templates#python",
			"config": map[string]interface{}{
				"command": "python --version",
			},
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, devTask)
	if err != nil {
		log.Printf("Development environment failed: %v", err)
	} else {
		log.Printf("Development environment result: %+v", result.Output)
	}

	// Example 3: NixOS system rebuild
	log.Println("\n3. NixOS system rebuild...")
	rebuildTask := &Task{
		Type: "nixos",
		Payload: map[string]interface{}{
			"action":    "rebuild",
			"flake_ref": "/etc/nixos",
			"config": map[string]interface{}{
				"rebuild_action": "dry-run", // Safe dry run
			},
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, rebuildTask)
	if err != nil {
		log.Printf("NixOS rebuild failed: %v", err)
	} else {
		log.Printf("NixOS rebuild result: %+v", result.Output)
	}

	// Example 4: Search packages
	log.Println("\n4. Searching for packages...")
	searchTask := &Task{
		Type: "nix",
		Payload: map[string]interface{}{
			"action":     "search",
			"expression": "python3",
			"nixpkgs":    "nixpkgs",
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, searchTask)
	if err != nil {
		log.Printf("Package search failed: %v", err)
	} else {
		log.Printf("Search result: %+v", result.Output)
	}

	// Example 5: Flake operations
	log.Println("\n5. Flake metadata...")
	flakeTask := &Task{
		Type: "nix",
		Payload: map[string]interface{}{
			"action":    "flake",
			"flake_ref": "nixpkgs",
			"config": map[string]interface{}{
				"flake_action": "metadata",
			},
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, flakeTask)
	if err != nil {
		log.Printf("Flake operation failed: %v", err)
	} else {
		log.Printf("Flake metadata: %+v", result.Output)
	}

	// Example 6: Store operations
	log.Println("\n6. Store information...")
	storeTask := &Task{
		Type: "nix",
		Payload: map[string]interface{}{
			"action": "store",
			"config": map[string]interface{}{
				"store_action": "info",
			},
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, storeTask)
	if err != nil {
		log.Printf("Store operation failed: %v", err)
	} else {
		log.Printf("Store info: %+v", result.Output)
	}

	// Example 7: System information
	log.Println("\n7. System information...")
	infoTask := &Task{
		Type: "nixos",
		Payload: map[string]interface{}{
			"action": "info",
		},
	}

	result, err = orchestrator.ExecuteTask(ctx, infoTask)
	if err != nil {
		log.Printf("System info failed: %v", err)
	} else {
		log.Printf("System info: %+v", result.Output)
	}

	return nil
}

// CreateNixWorkflow creates a comprehensive Nix-based infrastructure workflow.
func CreateNixWorkflow(config NixExampleConfig) *InfrastructureWorkflow {
	workflow := &InfrastructureWorkflow{
		ID:          fmt.Sprintf("nix-deploy-%s", config.Environment),
		Name:        fmt.Sprintf("Nix Infrastructure Deployment - %s", config.Environment),
		Description: "Complete Nix-based infrastructure deployment with reproducible builds",
		Steps:       []WorkflowStep{},
	}

	// Step 1: Build the system configuration
	workflow.Steps = append(workflow.Steps, WorkflowStep{
		ID:       "build-system",
		Name:     "Build NixOS Configuration",
		TaskType: "nix",
		Parameters: map[string]interface{}{
			"action":    "build",
			"flake_ref": fmt.Sprintf("%s#nixosConfigurations.%s", config.FlakeRef, config.Environment),
			"system":    "x86_64-linux",
		},
	})

	// Step 2: Deploy to remote host if specified
	if config.RemoteHost != "" {
		workflow.Steps = append(workflow.Steps, WorkflowStep{
			ID:        "deploy-remote",
			Name:      "Deploy to Remote Host",
			TaskType:  "nix",
			DependsOn: []string{"build-system"},
			Parameters: map[string]interface{}{
				"action":    "deploy",
				"flake_ref": fmt.Sprintf("%s#nixosConfigurations.%s", config.FlakeRef, config.Environment),
				"remote": map[string]interface{}{
					"host":    config.RemoteHost,
					"user":    config.RemoteUser,
					"ssh_key": config.SSHKeyPath,
				},
			},
		})
	}

	// Step 3: Build and deploy application containers
	workflow.Steps = append(workflow.Steps, WorkflowStep{
		ID:       "build-apps",
		Name:     "Build Application Containers",
		TaskType: "nix",
		DependsOn: func() []string {
			if config.RemoteHost != "" {
				return []string{"deploy-remote"}
			}
			return []string{"build-system"}
		}(),
		Parameters: map[string]interface{}{
			"action":    "build",
			"flake_ref": fmt.Sprintf("%s#packages.x86_64-linux", config.FlakeRef),
		},
	})

	// Step 4: Verify deployment
	workflow.Steps = append(workflow.Steps, WorkflowStep{
		ID:        "verify-deployment",
		Name:      "Verify Deployment",
		TaskType:  "nixos",
		DependsOn: []string{"build-apps"},
		Parameters: map[string]interface{}{
			"action": "info",
		},
	})

	// Step 5: Cleanup old generations
	workflow.Steps = append(workflow.Steps, WorkflowStep{
		ID:        "cleanup",
		Name:      "Cleanup Old Generations",
		TaskType:  "nix",
		DependsOn: []string{"verify-deployment"},
		Parameters: map[string]interface{}{
			"action": "gc",
			"config": map[string]interface{}{
				"max_age": "30d",
			},
		},
		ContinueOnError: true, // Don't fail workflow if cleanup fails
	})

	return workflow
}

// RunNixInfrastructureDemo demonstrates a complete Nix infrastructure workflow.
func RunNixInfrastructureDemo(ctx context.Context, orchestrator *Orchestrator) error {
	log.Println("=== Nix Infrastructure Workflow Demo ===")

	// Create a sample workflow configuration
	config := NixExampleConfig{
		FlakeRef:    "github:myorg/nixos-infrastructure",
		RemoteHost:  "staging.example.com",
		RemoteUser:  "deploy",
		SSHKeyPath:  "/path/to/deploy.key",
		Environment: "staging",
	}

	// Create the workflow
	workflow := CreateNixWorkflow(config)

	log.Printf("Created workflow: %s", workflow.Name)
	log.Printf("Workflow steps: %d", len(workflow.Steps))

	// Execute the workflow
	result, err := orchestrator.ExecuteInfrastructureWorkflow(ctx, workflow)
	if err != nil {
		return xerrors.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("Workflow completed with status: %s", result.Status)
	for i, step := range result.Steps {
		log.Printf("Step %d (%s): %s", i+1, step.Name, step.Status)
		if step.Error != "" {
			log.Printf("  Error: %s", step.Error)
		}
	}

	return nil
}

// ExampleNixFlakeDefinition provides an example flake.nix for infrastructure.
const ExampleNixFlakeDefinition = `{
  description = "Agentic Infrastructure with NixOS";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        # Development environment
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            nix
            nixos-rebuild
            docker
            kubectl
            terraform
          ];

          shellHook = ''
            echo "Agentic Infrastructure Development Environment"
            echo "Available tools: go, nix, docker, kubectl, terraform"
          '';
        };

        # Application packages
        packages = {
          agentic-server = pkgs.buildGoModule {
            pname = "agentic-server";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
          };

          docker-image = pkgs.dockerTools.buildImage {
            name = "agentic-server";
            tag = "latest";
            contents = [ self.packages.${system}.agentic-server ];
            config = {
              Cmd = [ "/bin/agentic-server" ];
              ExposedPorts = { "8080/tcp" = {}; };
            };
          };
        };
      }) // {
      # NixOS configurations
      nixosConfigurations = {
        staging = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ./nixos/staging.nix
            {
              services.agentic = {
                enable = true;
                package = self.packages.x86_64-linux.agentic-server;
              };
            }
          ];
        };

        production = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ./nixos/production.nix
            {
              services.agentic = {
                enable = true;
                package = self.packages.x86_64-linux.agentic-server;
              };
            }
          ];
        };
      };
    };
}`

// ExampleNixOSConfiguration provides an example NixOS system configuration.
const ExampleNixOSConfiguration = `{ config, pkgs, ... }:

{
  # Basic system configuration
  system.stateVersion = "23.11";
  
  # Enable flakes
  nix.settings.experimental-features = [ "nix-command" "flakes" ];
  
  # Networking
  networking = {
    hostName = "agentic-server";
    firewall = {
      enable = true;
      allowedTCPPorts = [ 22 80 443 8080 ];
    };
  };

  # Services
  services = {
    # SSH
    openssh = {
      enable = true;
      settings = {
        PasswordAuthentication = false;
        PermitRootLogin = "no";
      };
    };

    # Docker
    docker = {
      enable = true;
      enableOnBoot = true;
    };

    # Nginx reverse proxy
    nginx = {
      enable = true;
      recommendedGzipSettings = true;
      recommendedOptimisation = true;
      recommendedProxySettings = true;
      recommendedTlsSettings = true;

      virtualHosts."agentic.example.com" = {
        enableACME = true;
        forceSSL = true;
        locations."/" = {
          proxyPass = "http://127.0.0.1:8080";
          proxyWebsockets = true;
        };
      };
    };

    # ACME for SSL certificates
    acme = {
      acceptTerms = true;
      defaults.email = "admin@example.com";
    };
  };

  # System packages
  environment.systemPackages = with pkgs; [
    git
    curl
    htop
    docker-compose
    kubectl
    nix
  ];

  # Users
  users.users.deploy = {
    isNormalUser = true;
    extraGroups = [ "wheel" "docker" ];
    openssh.authorizedKeys.keys = [
      "ssh-rsa AAAAB3NzaC1yc2EAAAA... deploy@agentic"
    ];
  };

  # Security
  security.sudo.wheelNeedsPassword = false;
}`
