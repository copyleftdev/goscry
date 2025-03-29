package taskstypes

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTask_Creation(t *testing.T) {
	// Create a new task
	id := uuid.New()
	actions := []Action{
		{
			Type:  ActionNavigate,
			Value: "https://example.com",
		},
		{
			Type:     ActionWaitVisible,
			Selector: "#content",
		},
	}
	
	task := &Task{
		ID:            id,
		Status:        StatusPending,
		Actions:       actions,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CurrentAction: 0,
	}
	
	// Assertions
	assert.Equal(t, id, task.ID)
	assert.Equal(t, StatusPending, task.Status)
	assert.Equal(t, 2, len(task.Actions))
	assert.Equal(t, ActionNavigate, task.Actions[0].Type)
	assert.Equal(t, "https://example.com", task.Actions[0].Value)
}

func TestTask_TwoFactorAuth(t *testing.T) {
	// Create a task with 2FA info
	task := &Task{
		ID:      uuid.New(),
		Status:  StatusPending,
		Actions: []Action{},
		TwoFactorAuth: TwoFactorAuthInfo{
			Provider:    TFAProviderEmail,
			Email:       "user@example.com",
			Expected:    true,
			PhoneNumber: "",
		},
		TfaCodeChan: make(chan string, 1),
	}
	
	// Test 2FA code handling
	go func() {
		// Simulate providing a code after a short delay
		time.Sleep(100 * time.Millisecond)
		task.TfaCodeChan <- "123456"
	}()
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Try to get the code
	code, err := task.WaitForTFACode(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "123456", code)
}

func TestTaskResult_Creation(t *testing.T) {
	// Create a task result
	customData := map[string]interface{}{
		"dom_type":    "text_content",
		"dom_content": "Sample DOM content",
	}
	
	result := &TaskResult{
		Success:    true,
		Message:    "Task completed successfully",
		Data:       "Sample DOM content",
		CustomData: customData,
	}
	
	// Assertions
	assert.True(t, result.Success)
	assert.Equal(t, "Task completed successfully", result.Message)
	assert.Equal(t, "Sample DOM content", result.Data)
	assert.NotNil(t, result.CustomData)
	assert.Equal(t, "text_content", result.CustomData["dom_type"])
	assert.Equal(t, "Sample DOM content", result.CustomData["dom_content"])
}

func TestAction_Validation(t *testing.T) {
	testCases := []struct {
		name    string
		action  Action
		isValid bool
	}{
		{
			name: "Valid navigate action",
			action: Action{
				Type:  ActionNavigate,
				Value: "https://example.com",
			},
			isValid: true,
		},
		{
			name: "Invalid navigate action - empty URL",
			action: Action{
				Type:  ActionNavigate,
				Value: "",
			},
			isValid: false,
		},
		{
			name: "Valid click action",
			action: Action{
				Type:     ActionClick,
				Selector: "#submit-button",
			},
			isValid: true,
		},
		{
			name: "Invalid click action - empty selector",
			action: Action{
				Type:     ActionClick,
				Selector: "",
			},
			isValid: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For now, we just check basic conditions
			if tc.action.Type == ActionNavigate {
				assert.Equal(t, tc.isValid, tc.action.Value != "")
			} else if tc.action.Type == ActionClick {
				assert.Equal(t, tc.isValid, tc.action.Selector != "")
			}
		})
	}
}

func TestTaskStatuses(t *testing.T) {
	// Make sure we have all the expected task statuses
	statuses := []TaskStatus{
		StatusPending,
		StatusRunning,
		StatusWaitingFor2FA,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}
	
	// Test that each status has a unique string representation
	seen := make(map[string]bool)
	for _, status := range statuses {
		statusStr := string(status)
		assert.False(t, seen[statusStr], "Duplicate status: %s", statusStr)
		seen[statusStr] = true
	}
}
