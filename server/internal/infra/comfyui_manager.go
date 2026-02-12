package infra

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ComfyUI paths (from user)
const (
	// Conda environment path
	condaEnvPath = "D:\\conda\\envs\\comfyui"

	// Python interpreter path in conda environment
	pythonExePath = "D:\\conda\\envs\\comfyui\\python.exe"

	// ComfyUI root directory (where models are stored)
	comfyuiRootDir = "D:\\ComfyUI"
)

// ComfyUI configuration
const (
	comfyuiHost = "127.0.0.1"
	comfyuiPort = 8188
	maxRetries    = 3
	retryDelay    = 3 * time.Second
	startupTimeout = 30 * time.Second
)

// ComfyUI status
type ComfyUIStatus string

const (
	ComfyUIStatusStopped ComfyUIStatus = "stopped"
	ComfyUIStatusStarting ComfyUIStatus = "starting"
	ComfyUIStatusRunning  ComfyUIStatus = "running"
	ComfyUIStatusError   ComfyUIStatus = "error"
)

// ComfyUIManager manages local ComfyUI instance
type ComfyUIManager struct {
	status      ComfyUIStatus
	process     *os.Process
	statusMutex sync.RWMutex
	config      *ComfyUIManagerConfig
}

// ComfyUIManagerConfig holds ComfyUI manager configuration
type ComfyUIManagerConfig struct {
	Host     string
	Port     int
	ModelsDir string
	UseGPU   bool
}

// NewComfyUIManager creates a new ComfyUI manager
func NewComfyUIManager(config *ComfyUIManagerConfig) *ComfyUIManager {
	return &ComfyUIManager{
		status:     ComfyUIStatusStopped,
		process:     nil,
		statusMutex: sync.RWMutex{},
		config:      config,
	}
}

// Start starts ComfyUI using conda environment
func (m *ComfyUIManager) Start(ctx context.Context) error {
	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()

	if m.status == ComfyUIStatusRunning {
		return nil
	}

	// Update status
	m.status = ComfyUIStatusStarting

	// Check if conda environment exists
	if _, err := os.Stat(condaEnvPath); os.IsNotExist(err) {
		return fmt.Errorf("conda environment not found at: %s", condaEnvPath)
	}

	// Check if Python interpreter exists
	if _, err := os.Stat(pythonExePath); os.IsNotExist(err) {
		return fmt.Errorf("Python interpreter not found at: %s", pythonExePath)
	}

	// Check if ComfyUI root directory exists
	if _, err := os.Stat(comfyuiRootDir); os.IsNotExist(err) {
		return fmt.Errorf("ComfyUI root directory not found at: %s", comfyuiRootDir)
	}

	// Prepare command
	// Directly execute main.py with conda python interpreter
	cmd := exec.Command(pythonExePath, "main.py", "--listen", comfyuiHost, "--port", fmt.Sprintf("%d", comfyuiPort))
	// Set working directory to ComfyUI root to ensure model paths are correct
	cmd.Dir = comfyuiRootDir

	// Capture output for debugging
	log.Printf("Starting ComfyUI: %s %s (dir: %s)", pythonExePath, cmd.Args, comfyuiRootDir)

	m.status = ComfyUIStatusStarting

	// Start the process
	if err := cmd.Start(); err != nil {
		m.status = ComfyUIStatusError
		return fmt.Errorf("failed to start ComfyUI: %w", err)
	}

	log.Printf("ComfyUI process started with PID: %d", cmd.Process.Pid)

	m.process = cmd.Process

	// Wait for startup with timeout
	go m.waitForStartup(ctx)

	return nil
}

// Stop stops ComfyUI
func (m *ComfyUIManager) Stop(ctx context.Context) error {
	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()

	if m.status == ComfyUIStatusStopped {
		return nil
	}

	m.status = ComfyUIStatusStopped

	if m.process != nil {
			if err := m.process.Kill(); err != nil {
			// Log error but don't fail
		return fmt.Errorf("failed to kill ComfyUI: %w", err)
		}
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
			_, err := m.process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			m.status = ComfyUIStatusError
		return err
		}
		m.status = ComfyUIStatusStopped
		return nil
	case <-ctx.Done():
		m.process.Kill()
		m.status = ComfyUIStatusStopped
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Force kill if timeout
		m.process.Kill()
		m.status = ComfyUIStatusStopped
		return fmt.Errorf("stop timeout")
	}
}

// waitForStartup waits for ComfyUI to be ready
func (m *ComfyUIManager) waitForStartup(ctx context.Context) {
	log.Printf("ComfyUI waitForStartup started")

	// Wait a bit for the process to start
	time.Sleep(3 * time.Second)

	// Mark as running - actual HTTP readiness will be checked when making requests
	m.statusMutex.Lock()
	if m.status == ComfyUIStatusStarting {
		m.status = ComfyUIStatusRunning
		log.Printf("ComfyUI marked as running (PID: %d)", m.process.Pid)
	}
	m.statusMutex.Unlock()
}

// GetStatus returns current status
func (m *ComfyUIManager) GetStatus() ComfyUIStatus {
	m.statusMutex.RLock()
	defer m.statusMutex.RUnlock()
	return m.status
}

// IsReady checks if ComfyUI is ready to accept requests
func (m *ComfyUIManager) IsReady() bool {
	return m.status == ComfyUIStatusRunning
}

// GetURL returns the ComfyUI API URL
func (m *ComfyUIManager) GetURL() string {
	return fmt.Sprintf("http://%s:%d", m.config.Host, m.config.Port)
}

// Restart restarts ComfyUI
func (m *ComfyUIManager) Restart(ctx context.Context) error {
	if err := m.Stop(ctx); err != nil {
		return err
	}

	return m.Start(ctx)
}
