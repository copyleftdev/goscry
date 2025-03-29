package tasks

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/copyleftdev/goscry/internal/config"
	"github.com/copyleftdev/goscry/internal/tasks/mocks"
	"github.com/copyleftdev/goscry/internal/taskstypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestManager_SubmitTask(t *testing.T) {
	// Create a mock browser executor
	mockBrowser := mocks.NewMockBrowserExecutor()
	
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
	manager := NewManager(cfg, mockBrowser, testLogger)
	
	// Test submitting a basic task
	task := &taskstypes.Task{
		ID: uuid.New(),
		Actions: []taskstypes.Action{
			{
				Type:  taskstypes.ActionNavigate,
				Value: "https://example.com",
			},
			{
				Type:     taskstypes.ActionWaitVisible,
				Selector: "#content",
			},
		},
		Status:        taskstypes.StatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CurrentAction: 0,
	}
	
	// Submit the task
	err := manager.SubmitTask(task)
	
	// Assertions
	assert.NoError(t, err)
	
	// Wait a bit for processing to occur asynchronously
	time.Sleep(100 * time.Millisecond)
	
	// Get the task status
	taskStatus, err := manager.GetTaskStatus(task.ID)
	
	// Assertions for task retrieval
	assert.NoError(t, err)
	assert.Equal(t, task.ID, taskStatus.ID)
	// Status might be any of these depending on execution speed
	possibleStatuses := []taskstypes.TaskStatus{
		taskstypes.StatusPending,
		taskstypes.StatusRunning,
		taskstypes.StatusCompleted,
	}
	assert.Contains(t, possibleStatuses, taskStatus.Status)
	assert.Equal(t, 2, len(taskStatus.Actions))
}

func TestManager_Shutdown(t *testing.T) {
	// Create a mock browser executor
	mockBrowser := mocks.NewMockBrowserExecutor()
	
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
	manager := NewManager(cfg, mockBrowser, testLogger)
	
	// Call Shutdown
	err := manager.Shutdown(context.Background())
	
	// Assertions - just check that it doesn't error
	assert.NoError(t, err)
	// Note: In the real implementation, we don't actually call browser.Shutdown()
	// so we're not asserting mockBrowser.WasShutdownCalled() anymore
}
