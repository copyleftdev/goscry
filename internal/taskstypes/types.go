package taskstypes

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Task status constants
type TaskStatus string

const (
	StatusPending       TaskStatus = "pending"
	StatusRunning       TaskStatus = "running"
	StatusWaitingFor2FA TaskStatus = "waiting_for_2fa"
	StatusCompleted     TaskStatus = "completed"
	StatusFailed        TaskStatus = "failed"
	StatusCancelled     TaskStatus = "cancelled"
)

// Action type constants
type ActionType string

const (
	ActionNavigate    ActionType = "navigate"
	ActionWaitVisible ActionType = "wait_visible"
	ActionWaitHidden  ActionType = "wait_hidden"
	ActionWaitDelay   ActionType = "wait_delay"
	ActionClick       ActionType = "click"
	ActionInput       ActionType = "type"
	ActionSelect      ActionType = "select"
	ActionScroll      ActionType = "scroll"
	ActionScreenshot  ActionType = "screenshot"
	ActionGetDOM      ActionType = "get_dom"
	ActionRunScript   ActionType = "run_script"
	ActionLogin       ActionType = "login"
)

// TFA provider constants
type TFAProvider string

const (
	TFAProviderEmail TFAProvider = "email"
	TFAProviderSMS   TFAProvider = "sms"
	TFAProviderApp   TFAProvider = "app"
)

// Action represents a browser action to be performed
type Action struct {
	Type     ActionType    `json:"type"`
	Selector string        `json:"selector,omitempty"`
	Value    string        `json:"value,omitempty"`
	Format   string        `json:"format,omitempty"`
	Timeout  time.Duration `json:"-"`
}

// SelectorOrDefault returns the selector if set, otherwise returns the default selector
func (a *Action) SelectorOrDefault(defaultSelector string) string {
	if a.Selector == "" {
		return defaultSelector
	}
	return a.Selector
}

// Credentials for authentication actions
type Credentials struct {
	Username string `json:"-"`
	Password string `json:"-"`
}

// TwoFactorAuthInfo for 2FA configuration and state
type TwoFactorAuthInfo struct {
	Expected    bool        `json:"expected"`
	Handler     string      `json:"handler"`
	Provider    TFAProvider `json:"provider"`
	Email       string      `json:"email,omitempty"`
	PhoneNumber string      `json:"phone_number,omitempty"`
	Secret      string      `json:"-"`
	Code        string      `json:"-"`
}

// Task struct definition
type Task struct {
	ID               uuid.UUID         `json:"id"`
	Status           TaskStatus        `json:"status"`
	Actions          []Action          `json:"actions"`
	Credentials      *Credentials      `json:"-"`
	TwoFactorAuth    TwoFactorAuthInfo `json:"two_factor_auth"`
	CurrentAction    int               `json:"current_action"`
	Result           *TaskResult       `json:"result,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	BrowserContextID string            `json:"-"`
	CallbackURL      string            `json:"callback_url,omitempty"`
	TfaCodeChan      chan string       `json:"-"`
}

// WaitForTFACode waits for a 2FA code to be provided through the task's channel
func (t *Task) WaitForTFACode(ctx context.Context) (string, error) {
	if t.TfaCodeChan == nil {
		t.TfaCodeChan = make(chan string, 1)
	}

	// Create a timeout context if not already done
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Wait for either a code or a timeout
	select {
	case code := <-t.TfaCodeChan:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// TaskResult contains the execution result
type TaskResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message,omitempty"`
	Data       interface{}            `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
}

// UpdateStatus updates the task status and timestamp
func (t *Task) UpdateStatus(status TaskStatus) {
	t.Status = status
	t.UpdatedAt = time.Now()
}

// SetResult sets the task result
func (t *Task) SetResult(success bool, message string, data interface{}, customData map[string]interface{}, err error) {
	if t.Result == nil {
		t.Result = &TaskResult{}
	}

	t.Result.Success = success
	t.Result.Message = message
	t.Result.Data = data
	t.Result.CustomData = customData

	if err != nil {
		t.Result.Error = err.Error()
	}
}
