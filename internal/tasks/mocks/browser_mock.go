package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/copyleftdev/goscry/internal/taskstypes"
)

// MockBrowserExecutor implements the tasks.BrowserExecutor interface for testing
type MockBrowserExecutor struct {
	mu                sync.Mutex
	executedTasks     []*taskstypes.Task
	executionResults  map[string]*taskstypes.TaskResult
	executionErrors   map[string]error
	shutdownCalled    bool
	shutdownError     error
	simulateTwoFactor bool
}

// NewMockBrowserExecutor creates a new mock browser executor
func NewMockBrowserExecutor() *MockBrowserExecutor {
	return &MockBrowserExecutor{
		executedTasks:    make([]*taskstypes.Task, 0),
		executionResults: make(map[string]*taskstypes.TaskResult),
		executionErrors:  make(map[string]error),
	}
}

// ExecuteTask implements the BrowserExecutor interface
func (m *MockBrowserExecutor) ExecuteTask(task *taskstypes.Task) (*taskstypes.TaskResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executedTasks = append(m.executedTasks, task)

	// If we're simulating 2FA and task has 2FA info, return immediately with WaitingFor2FA status
	if m.simulateTwoFactor && task.TwoFactorAuth.Expected {
		// Only change status to waiting if we're not already past that point
		if task.Status != taskstypes.StatusWaitingFor2FA && task.Status != taskstypes.StatusCompleted {
			task.UpdateStatus(taskstypes.StatusWaitingFor2FA)
			return &taskstypes.TaskResult{
				Success: false,
				Message: "Task is waiting for 2FA code",
			}, nil
		}
	}

	// Use predefined result or error if available for this task ID
	taskID := task.ID.String()
	if result, ok := m.executionResults[taskID]; ok {
		task.Result = result
		task.UpdateStatus(taskstypes.StatusCompleted)
		return result, m.executionErrors[taskID]
	}
	
	// Default behavior is to simulate successful execution
	defaultResult := &taskstypes.TaskResult{
		Success: true,
		Message: "Task executed successfully by mock executor",
		Data:    fmt.Sprintf("Mock execution of task %s", task.ID),
	}
	
	// If we need to wait for 2FA, only proceed if the code has been provided
	if task.Status == taskstypes.StatusWaitingFor2FA {
		// If we have a code channel, use it to get the code
		if task.TfaCodeChan != nil {
			// Simulated wait for code
			select {
			case <-time.After(50 * time.Millisecond):
				// No code provided, keep waiting
				return &taskstypes.TaskResult{
					Success: false,
					Message: "Still waiting for 2FA code",
				}, nil
			case code := <-task.TfaCodeChan:
				// Code received, proceed with completion
				defaultResult.Message = fmt.Sprintf("Task completed with 2FA code: %s", code)
			}
		}
	}
	
	task.UpdateStatus(taskstypes.StatusCompleted)
	task.Result = defaultResult
	return defaultResult, nil
}

// Shutdown implements the BrowserExecutor interface
func (m *MockBrowserExecutor) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.shutdownCalled = true
	return m.shutdownError
}

// ExecutedTasks returns the tasks that were executed
func (m *MockBrowserExecutor) ExecutedTasks() []*taskstypes.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.executedTasks
}

// WasShutdownCalled returns whether Shutdown was called
func (m *MockBrowserExecutor) WasShutdownCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.shutdownCalled
}

// SetExecutionResult sets a predefined result for a task ID
func (m *MockBrowserExecutor) SetExecutionResult(taskID string, result *taskstypes.TaskResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executionResults[taskID] = result
	m.executionErrors[taskID] = err
}

// SetShutdownError sets the error to return from Shutdown
func (m *MockBrowserExecutor) SetShutdownError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.shutdownError = err
}

// SimulateTwoFactorAuth enables/disables 2FA simulation
func (m *MockBrowserExecutor) SimulateTwoFactorAuth(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.simulateTwoFactor = enable
}
