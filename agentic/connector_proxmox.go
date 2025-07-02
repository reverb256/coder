// Package agentic provides a Proxmox VE connector for VM/container orchestration.
package agentic

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

// ProxmoxConfig holds Proxmox VE API configuration.
type ProxmoxConfig struct {
	URL      string // Proxmox VE API URL (e.g., https://pve.example.com:8006)
	Username string // Username (e.g., root@pam)
	Password string // Password or API token secret
	Token    string // API token (alternative to password)
	Node     string // Default node name
	Insecure bool   // Skip TLS verification (for testing)
}

// ProxmoxClient is an agent for Proxmox VE VM/container orchestration tasks.
type ProxmoxClient struct {
	cfg        ProxmoxConfig
	httpClient *http.Client
	ticket     string    // Authentication ticket
	csrfToken  string    // CSRF prevention token
	ticketExp  time.Time // Ticket expiration
}

// ProxmoxTask represents a Proxmox orchestration task.
type ProxmoxTask struct {
	Action            string                 `json:"action"`   // create, start, stop, delete, list
	VMType            string                 `json:"vm_type"`  // qemu, lxc
	VMID              int                    `json:"vmid"`     // Virtual machine ID
	Node              string                 `json:"node"`     // Proxmox node name
	Template          string                 `json:"template"` // Template name/ID
	Config            map[string]interface{} `json:"config"`   // VM/container configuration
	WaitForCompletion bool                   `json:"wait"`     // Wait for task completion
	GPUs              *ProxmoxGPUConfig      `json:"gpus"`     // GPU passthrough configuration
}

// ProxmoxVM represents a Proxmox virtual machine or container.
type ProxmoxVM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"` // qemu, lxc
	Node   string `json:"node"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"`
	Disk   string `json:"disk"`
	Uptime int    `json:"uptime"`
}

// ProxmoxTaskStatus represents a Proxmox task status.
type ProxmoxTaskStatus struct {
	UPID      string `json:"upid"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	ExitCode  string `json:"exitstatus,omitempty"`
	StartTime int    `json:"starttime"`
	EndTime   int    `json:"endtime,omitempty"`
}

// ProxmoxGPUConfig represents GPU passthrough configuration for Proxmox VMs.
type ProxmoxGPUConfig struct {
	Enabled    bool     `json:"enabled"`     // Enable GPU passthrough
	PCIDevices []string `json:"pci_devices"` // PCI device IDs (e.g., "01:00.0")
	GPUIDs     []int    `json:"gpu_ids"`     // GPU IDs to passthrough
	VFIO       bool     `json:"vfio"`        // Use VFIO for passthrough
	Primary    bool     `json:"primary"`     // Set as primary GPU
	ROMBAR     bool     `json:"rombar"`      // Enable ROM BAR
	X_VGA      bool     `json:"x_vga"`       // Enable VGA passthrough
	PrimaryGPU bool     `json:"primary_gpu"` // Configure as primary GPU
}

// NewProxmoxClient creates a new Proxmox VE client.
func NewProxmoxClient(cfg ProxmoxConfig) *ProxmoxClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	return &ProxmoxClient{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (p *ProxmoxClient) Name() string { return "proxmox" }

func (p *ProxmoxClient) Supports(taskType string) bool {
	return taskType == "vm" || taskType == "container" || taskType == "infrastructure"
}

func (p *ProxmoxClient) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Parse task payload
	var proxmoxTask ProxmoxTask
	if err := mapToStruct(task.Payload, &proxmoxTask); err != nil {
		return &TaskResult{Error: xerrors.Errorf("invalid task payload: %w", err)}, nil
	}

	// Ensure authentication
	if err := p.authenticate(ctx); err != nil {
		return &TaskResult{Error: xerrors.Errorf("authentication failed: %w", err)}, nil
	}

	// Execute based on action
	switch proxmoxTask.Action {
	case "list":
		result, err := p.listVMs(ctx, proxmoxTask.Node)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "create":
		result, err := p.createVM(ctx, &proxmoxTask)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "start":
		result, err := p.startVM(ctx, proxmoxTask.Node, proxmoxTask.VMID, proxmoxTask.VMType)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "stop":
		result, err := p.stopVM(ctx, proxmoxTask.Node, proxmoxTask.VMID, proxmoxTask.VMType)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "delete":
		result, err := p.deleteVM(ctx, proxmoxTask.Node, proxmoxTask.VMID, proxmoxTask.VMType)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	case "status":
		result, err := p.getVMStatus(ctx, proxmoxTask.Node, proxmoxTask.VMID, proxmoxTask.VMType)
		if err != nil {
			return &TaskResult{Error: err}, nil
		}
		return &TaskResult{Output: result}, nil

	default:
		return &TaskResult{Error: xerrors.Errorf("unsupported action: %s", proxmoxTask.Action)}, nil
	}
}

// authenticate performs authentication with Proxmox VE API.
func (p *ProxmoxClient) authenticate(ctx context.Context) error {
	// Check if we have a valid ticket
	if time.Now().Before(p.ticketExp) && p.ticket != "" {
		return nil
	}

	// Use API token if provided
	if p.cfg.Token != "" {
		// API tokens don't need authentication tickets
		return nil
	}

	// Authenticate with username/password
	authURL := fmt.Sprintf("%s/api2/json/access/ticket", strings.TrimSuffix(p.cfg.URL, "/"))

	data := url.Values{}
	data.Set("username", p.cfg.Username)
	data.Set("password", p.cfg.Password)

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return xerrors.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return xerrors.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return xerrors.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var authResp struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return xerrors.Errorf("failed to decode auth response: %w", err)
	}

	p.ticket = authResp.Data.Ticket
	p.csrfToken = authResp.Data.CSRFPreventionToken
	p.ticketExp = time.Now().Add(2 * time.Hour) // Proxmox tickets are valid for 2 hours

	return nil
}

// makeRequest makes an authenticated request to the Proxmox API.
func (p *ProxmoxClient) makeRequest(ctx context.Context, method, path string, payload interface{}) (*http.Response, error) {
	apiURL := fmt.Sprintf("%s/api2/json%s", strings.TrimSuffix(p.cfg.URL, "/"), path)

	var reqBody io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, xerrors.Errorf("failed to marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, reqBody)
	if err != nil {
		return nil, xerrors.Errorf("failed to create request: %w", err)
	}

	// Set authentication headers
	if p.cfg.Token != "" {
		// Use API token
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", p.cfg.Token))
	} else {
		// Use authentication ticket
		req.Header.Set("Cookie", fmt.Sprintf("PVEAuthCookie=%s", p.ticket))
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", p.csrfToken)
		}
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return p.httpClient.Do(req)
}

// listVMs lists all VMs and containers on a node.
func (p *ProxmoxClient) listVMs(ctx context.Context, node string) ([]ProxmoxVM, error) {
	if node == "" {
		node = p.cfg.Node
	}

	path := fmt.Sprintf("/nodes/%s/qemu", node)
	resp, err := p.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to list VMs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, xerrors.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data []struct {
			VMID   int    `json:"vmid"`
			Name   string `json:"name"`
			Status string `json:"status"`
			CPU    int    `json:"cpu"`
			Memory int    `json:"mem"`
			Disk   int    `json:"disk"`
			Uptime int    `json:"uptime"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, xerrors.Errorf("failed to decode response: %w", err)
	}

	vms := make([]ProxmoxVM, len(apiResp.Data))
	for i, vm := range apiResp.Data {
		vms[i] = ProxmoxVM{
			VMID:   vm.VMID,
			Name:   vm.Name,
			Status: vm.Status,
			Type:   "qemu",
			Node:   node,
			CPU:    vm.CPU,
			Memory: vm.Memory,
			Disk:   strconv.Itoa(vm.Disk),
			Uptime: vm.Uptime,
		}
	}

	// Also list containers
	path = fmt.Sprintf("/nodes/%s/lxc", node)
	resp, err = p.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return vms, nil // Return VMs even if container listing fails
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var containerResp struct {
			Data []struct {
				VMID   int    `json:"vmid"`
				Name   string `json:"name"`
				Status string `json:"status"`
				CPU    int    `json:"cpu"`
				Memory int    `json:"mem"`
				Disk   int    `json:"disk"`
				Uptime int    `json:"uptime"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&containerResp); err == nil {
			for _, container := range containerResp.Data {
				vms = append(vms, ProxmoxVM{
					VMID:   container.VMID,
					Name:   container.Name,
					Status: container.Status,
					Type:   "lxc",
					Node:   node,
					CPU:    container.CPU,
					Memory: container.Memory,
					Disk:   strconv.Itoa(container.Disk),
					Uptime: container.Uptime,
				})
			}
		}
	}

	return vms, nil
}

// createVM creates a new VM or container.
func (p *ProxmoxClient) createVM(ctx context.Context, task *ProxmoxTask) (map[string]interface{}, error) {
	if task.Node == "" {
		task.Node = p.cfg.Node
	}

	vmType := task.VMType
	if vmType == "" {
		vmType = "qemu" // Default to QEMU VM
	}

	// Build the configuration
	config := make(map[string]interface{})
	if task.Config != nil {
		config = task.Config
	}

	// Set VMID if provided
	if task.VMID > 0 {
		config["vmid"] = task.VMID
	}

	path := fmt.Sprintf("/nodes/%s/%s", task.Node, vmType)
	resp, err := p.makeRequest(ctx, "POST", path, config)
	if err != nil {
		return nil, xerrors.Errorf("failed to create VM: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("VM creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"status": "created", "response": string(body)}, nil
	}

	return result, nil
}

// startVM starts a VM or container.
func (p *ProxmoxClient) startVM(ctx context.Context, node string, vmid int, vmType string) (map[string]interface{}, error) {
	if node == "" {
		node = p.cfg.Node
	}
	if vmType == "" {
		vmType = "qemu"
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d/status/start", node, vmType, vmid)
	resp, err := p.makeRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to start VM: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("VM start failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"status": "started", "vmid": vmid}, nil
	}

	return result, nil
}

// stopVM stops a VM or container.
func (p *ProxmoxClient) stopVM(ctx context.Context, node string, vmid int, vmType string) (map[string]interface{}, error) {
	if node == "" {
		node = p.cfg.Node
	}
	if vmType == "" {
		vmType = "qemu"
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d/status/stop", node, vmType, vmid)
	resp, err := p.makeRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to stop VM: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("VM stop failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"status": "stopped", "vmid": vmid}, nil
	}

	return result, nil
}

// deleteVM deletes a VM or container.
func (p *ProxmoxClient) deleteVM(ctx context.Context, node string, vmid int, vmType string) (map[string]interface{}, error) {
	if node == "" {
		node = p.cfg.Node
	}
	if vmType == "" {
		vmType = "qemu"
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d", node, vmType, vmid)
	resp, err := p.makeRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to delete VM: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("VM deletion failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"status": "deleted", "vmid": vmid}, nil
	}

	return result, nil
}

// getVMStatus gets the status of a VM or container.
func (p *ProxmoxClient) getVMStatus(ctx context.Context, node string, vmid int, vmType string) (map[string]interface{}, error) {
	if node == "" {
		node = p.cfg.Node
	}
	if vmType == "" {
		vmType = "qemu"
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d/status/current", node, vmType, vmid)
	resp, err := p.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to get VM status: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("VM status request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, xerrors.Errorf("failed to decode status response: %w", err)
	}

	return result, nil
}

// mapToStruct converts a map to a struct using JSON marshaling/unmarshaling.
func mapToStruct(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
