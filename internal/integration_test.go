package internal

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/copyleftdev/goscry/internal/config"
	"github.com/copyleftdev/goscry/internal/tasks"
	"github.com/copyleftdev/goscry/internal/tasks/mocks"
	"github.com/copyleftdev/goscry/internal/taskstypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This is an integration test that simulates the entire GoScry workflow
// using the mock browser executor to avoid actual browser dependencies

func TestGoScryIntegrationWorkflow(t *testing.T) {
	// Create a mock browser executor
	mockBrowser := mocks.NewMockBrowserExecutor()

	// Configure the mock to simulate 2FA when needed
	mockBrowser.SimulateTwoFactorAuth(true)

	// Create a test logger
	testLogger := log.New(os.Stderr, "TEST: ", log.LstdFlags)

	// Create a minimal config
	cfg := &config.Config{
		Browser: config.BrowserConfig{
			MaxSessions: 5,
			Headless:    true,
		},
	}

	// Create a task manager with the mock browser
	manager := tasks.NewManager(cfg, mockBrowser, testLogger)

	// Test without 2FA first - this should complete successfully
	simpleActions := []taskstypes.Action{
		{
			Type:  taskstypes.ActionNavigate,
			Value: "https://example.com",
		},
	}

	simpleTask := &taskstypes.Task{
		ID:            uuid.New(),
		Status:        taskstypes.StatusPending,
		Actions:       simpleActions,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CurrentAction: 0,
	}

	// Submit the simple task
	err := manager.SubmitTask(simpleTask)
	require.NoError(t, err)

	// Wait for completion
	waitForTaskCompletion(t, manager, simpleTask.ID, 2*time.Second)

	// Verify it completed
	task, err := manager.GetTaskStatus(simpleTask.ID)
	require.NoError(t, err)
	assert.Equal(t, taskstypes.StatusCompleted, task.Status)

	// Clean up
	err = manager.Shutdown(context.Background())
	require.NoError(t, err)
}

// Helper function to wait for task completion
func waitForTaskCompletion(t *testing.T, manager *tasks.Manager, taskID uuid.UUID, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := manager.GetTaskStatus(taskID)
		if err == nil && task.Status == taskstypes.StatusCompleted {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	task, err := manager.GetTaskStatus(taskID)
	status := "unknown"
	if err == nil {
		status = string(task.Status)
	}
	t.Fatalf("Task did not complete within timeout, current status: %s", status)
}
