package browser

import (
	"testing"

	"github.com/copyleftdev/goscry/internal/taskstypes"
	"github.com/stretchr/testify/assert"
)

// These tests check the browser action generation functionality

func TestGenerateActionSequence_Navigate(t *testing.T) {
	// Test navigate action
	action := taskstypes.Action{
		Type:  taskstypes.ActionNavigate,
		Value: "https://example.com",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

func TestGenerateActionSequence_WaitVisible(t *testing.T) {
	// Test wait visible action
	action := taskstypes.Action{
		Type:     taskstypes.ActionWaitVisible,
		Selector: "#content",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

func TestGenerateActionSequence_Click(t *testing.T) {
	// Test click action
	action := taskstypes.Action{
		Type:     taskstypes.ActionClick,
		Selector: "button.submit",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

func TestGenerateActionSequence_Type(t *testing.T) {
	// Test type action
	action := taskstypes.Action{
		Type:     taskstypes.ActionInput,
		Selector: "input[name='email']",
		Value:    "test@example.com",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

func TestGenerateActionSequence_WaitDelay(t *testing.T) {
	// Test wait delay action
	action := taskstypes.Action{
		Type:  taskstypes.ActionWaitDelay,
		Value: "5s",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

// Skip the screenshot test as it requires a running Chrome instance
// and causes a panic in the test environment
func TestGenerateActionSequence_Screenshot(t *testing.T) {
	t.Skip("Skipping screenshot test as it requires a running Chrome instance")
}

func TestGenerateActionSequence_GetDOM(t *testing.T) {
	// Test get DOM action
	action := taskstypes.Action{
		Type:     taskstypes.ActionGetDOM,
		Selector: "#main-content",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}

func TestGenerateActionSequence_InvalidAction(t *testing.T) {
	// Test with empty selector for click
	invalidAction := taskstypes.Action{
		Type:     taskstypes.ActionClick,
		Selector: "",
	}

	_, err := GenerateActionSequence(invalidAction, nil, "")
	assert.Error(t, err)
}

func TestGenerateActionSequence_2FACodeResolution(t *testing.T) {
	// Test 2FA code resolution
	action := taskstypes.Action{
		Type:     taskstypes.ActionInput,
		Selector: "input.tfa-code",
		Value:    "{{task.tfa_code}}",
	}

	cdpAction, err := GenerateActionSequence(action, nil, "123456")
	assert.NoError(t, err)
	assert.NotNil(t, cdpAction)
}
