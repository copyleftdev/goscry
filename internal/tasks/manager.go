package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/copyleftdev/goscry/internal/config"
	"github.com/copyleftdev/goscry/internal/taskstypes"
)

const twoFAWaitTimeout = 5 * time.Minute // Max time to wait for 2FA code

// Define a stub for MCP Client until the real implementation is available
type mcpClient struct {
	endpoint string
	apiKey   string
}

func newMCPClient(endpoint, apiKey string) *mcpClient {
	return &mcpClient{
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

// TwoFactorAuthRequest is a stub for the MCP two-factor auth request
type twoFactorAuthRequest struct {
	TaskID      string `json:"task_id"`
	Provider    string `json:"provider"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Email       string `json:"email,omitempty"`
}

type Manager struct {
	cfg            *config.Config
	browserExecutor BrowserExecutor
	logger         *log.Logger
	tasks          map[uuid.UUID]*taskstypes.Task
	mu             sync.RWMutex
	mcpConn        *mcpClient // Changed to our stub type
}

// NewManager creates a new task manager with the provided browser manager and logger.
func NewManager(cfg *config.Config, browserExecutor BrowserExecutor, logger *log.Logger) *Manager {
	// Create a simple manager without MCP connection for now
	mgr := &Manager{
		cfg:            cfg,
		browserExecutor: browserExecutor,
		logger:         logger,
		tasks:          make(map[uuid.UUID]*taskstypes.Task),
	}
	
	// Add stub MCP client if Config has the fields, otherwise use a default
	mcpEndpoint := "http://localhost:8080"
	mcpApiKey := "default-key"
	
	// Check if cfg.MCPConfig exists through reflection to avoid compile errors
	if cfg != nil {
		// This is just a placeholder - in real code we'd check if cfg.MCPConfig exists
		mgr.logger.Println("Using default MCP configuration")
	}
	
	mgr.mcpConn = newMCPClient(mcpEndpoint, mcpApiKey)
	return mgr
}

// SubmitTask adds a task to the manager's queue and starts executing it.
func (m *Manager) SubmitTask(task *taskstypes.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Store the task in the manager
	m.tasks[task.ID] = task
	
	// Start task execution in a goroutine
	go m.executeTask(task)
	
	return nil
}

// GetTaskStatus returns a copy of a task with its current status.
func (m *Manager) GetTaskStatus(id uuid.UUID) (*taskstypes.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	task, exists := m.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}
	
	// Return a copy to avoid race conditions
	taskCopy := *task
	return &taskCopy, nil
}

// Provide2FACode sends a 2FA code to a task waiting for one.
func (m *Manager) Provide2FACode(id uuid.UUID, code string) error {
	m.mu.RLock()
	task, exists := m.tasks[id]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}
	
	// Check if the task is waiting for 2FA
	if task.Status != taskstypes.StatusWaitingFor2FA {
		return fmt.Errorf("task is not waiting for 2FA code (status: %s)", task.Status)
	}
	
	// Send the code to the task's channel
	select {
	case task.TfaCodeChan <- code:
		m.logger.Printf("2FA code provided for task %s", id)
		return nil
	default:
		// This should never happen if the task is really waiting for 2FA
		return fmt.Errorf("failed to provide 2FA code, channel not ready")
	}
}

// executeTask handles the execution of a task, moving through execution phases.
func (m *Manager) executeTask(task *taskstypes.Task) {
	// Update initial status to running
	m.updateTaskStatus(task, taskstypes.StatusRunning)
	
	// Start browser execution
	result, err := m.browserExecutor.ExecuteTask(task)
	
	// Update task with final status based on execution result
	if err != nil {
		m.logger.Printf("Error executing task %s: %v", task.ID, err)
		task.Result = &taskstypes.TaskResult{
			Error: err.Error(),
		}
		m.updateTaskStatus(task, taskstypes.StatusFailed)
	} else {
		task.Result = result
		m.updateTaskStatus(task, taskstypes.StatusCompleted)
	}
	
	// Send callback notification if configured
	if task.CallbackURL != "" {
		go m.notifyCallback(task)
	}
}

// updateTaskStatus handles updating task status with proper locking
func (m *Manager) updateTaskStatus(task *taskstypes.Task, status taskstypes.TaskStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task.Status = status
	task.UpdatedAt = time.Now()
}

// notifyCallback sends a notification to the callback URL if specified
func (m *Manager) notifyCallback(task *taskstypes.Task) {
	if task.CallbackURL == "" {
		return
	}
	
	m.logger.Printf("Sending callback notification for task %s to %s", task.ID, task.CallbackURL)
	
	// Helper function to marshal task for callback - add to taskstypes package later
	marshalForCallback := func(task *taskstypes.Task) ([]byte, error) {
		// Create a simplified version with only the fields needed for callback
		callbackTask := struct {
			ID            string                       `json:"id"`
			Status        string                       `json:"status"`
			Result        *taskstypes.TaskResult       `json:"result,omitempty"`
			CurrentAction int                          `json:"current_action"`
			Actions       []taskstypes.Action          `json:"actions"`
			TwoFactorAuth taskstypes.TwoFactorAuthInfo `json:"two_factor_auth,omitempty"`
			CreatedAt     time.Time                    `json:"created_at"`
			UpdatedAt     time.Time                    `json:"updated_at"`
		}{
			ID:            task.ID.String(),
			Status:        string(task.Status),
			Result:        task.Result,
			CurrentAction: task.CurrentAction,
			Actions:       task.Actions,
			TwoFactorAuth: task.TwoFactorAuth,
			CreatedAt:     task.CreatedAt,
			UpdatedAt:     task.UpdatedAt,
		}
		
		return json.Marshal(callbackTask)
	}
	
	// Marshal task data for the callback
	taskData, err := marshalForCallback(task)
	if err != nil {
		m.logger.Printf("Error marshaling task data for callback: %v", err)
		return
	}
	
	// Create the request
	req, err := http.NewRequest("POST", task.CallbackURL, bytes.NewBuffer(taskData))
	if err != nil {
		m.logger.Printf("Error creating callback request: %v", err)
		return
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	// Add authentication if needed - using stub values for now
	callbackUsername := "callback-user"
	callbackPassword := "callback-password"
	
	// Check for callback auth configuration - stub implementation
	if m.cfg != nil {
		// In real code, we would check if m.cfg.CallbackAuth exists
		m.logger.Println("Using default callback authentication")
		
		// Set basic auth if needed
		if callbackUsername != "" && callbackPassword != "" {
			req.SetBasicAuth(callbackUsername, callbackPassword)
		}
	}
	
	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Printf("Error sending callback: %v", err)
		return
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("Callback notification sent successfully (status: %s)", resp.Status)
	} else {
		m.logger.Printf("Callback notification failed (status: %s)", resp.Status)
	}
}

// Shutdown gracefully cleans up any resources used by the manager.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Cancel any running tasks (in a real implementation)
	for id, task := range m.tasks {
		if task.Status == taskstypes.StatusRunning || task.Status == taskstypes.StatusWaitingFor2FA {
			m.logger.Printf("Cancelling task %s during shutdown", id)
			task.Status = taskstypes.StatusCancelled
		}
	}
	
	m.logger.Println("Task manager shut down")
	return nil
}
