package tasks

import (
	"time"

	"github.com/google/uuid"
	"github.com/copyleftdev/goscry/internal/taskstypes"
)

type TaskStatus string

const (
	StatusPending       TaskStatus = "pending"
	StatusRunning       TaskStatus = "running"
	StatusWaitingFor2FA TaskStatus = "waiting_for_2fa"
	StatusCompleted     TaskStatus = "completed"
	StatusFailed        TaskStatus = "failed"
)

// Action type constants moved to taskstypes package - just alias to maintain compatibility
type ActionType = taskstypes.ActionType

// Constants moved to taskstypes
const (
	ActionNavigate    = taskstypes.ActionNavigate
	ActionWaitVisible = taskstypes.ActionWaitVisible
	ActionWaitHidden  = taskstypes.ActionWaitHidden
	ActionWaitDelay   = taskstypes.ActionWaitDelay
	ActionClick       = taskstypes.ActionClick
	ActionInput       = taskstypes.ActionInput
	ActionSelect      = taskstypes.ActionSelect
	ActionScroll      = taskstypes.ActionScroll
	ActionScreenshot  = taskstypes.ActionScreenshot
	ActionGetDOM      = taskstypes.ActionGetDOM
	ActionRunScript   = taskstypes.ActionRunScript
	ActionLogin       = taskstypes.ActionLogin
)

// Action type moved to taskstypes - alias for compatibility
type Action = taskstypes.Action

// Credentials moved to taskstypes - alias for compatibility
type Credentials = taskstypes.Credentials  

// TwoFactorAuthInfo moved to taskstypes - alias for compatibility
type TwoFactorAuthInfo = taskstypes.TwoFactorAuthInfo

type Task struct {
	ID               uuid.UUID         `json:"id"`
	Status           TaskStatus        `json:"status"`
	Actions          []Action          `json:"actions"`
	Credentials      *Credentials      `json:"-"`
	TwoFactorAuth    TwoFactorAuthInfo `json:"two_factor_auth"`
	CurrentAction    int               `json:"current_action"` // Index of the action being processed by executor
	Result           *TaskResult       `json:"result,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	BrowserContextID string            `json:"-"` // Internal tracking within browser executor if needed
	CallbackURL      string            `json:"callback_url,omitempty"`
	// Internal channel, not serialized. Used by Manager to signal executor about 2FA code.
	TfaCodeChan chan string `json:"-"`
}

type TaskResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message,omitempty"`
	Data       interface{}            `json:"data,omitempty"` // Raw result data (e.g., string, bytes, map)
	Error      string                 `json:"error,omitempty"`
	CustomData map[string]interface{} `json:"custom_data,omitempty"` // Added field for storing metadata like mimeType
}

func NewTask(actions []Action, creds *Credentials, tfa TwoFactorAuthInfo, callback string) *Task {
	return &Task{
		ID:            uuid.New(),
		Status:        StatusPending,
		Actions:       actions,
		Credentials:   creds,
		TwoFactorAuth: tfa,
		CurrentAction: 0,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		CallbackURL:   callback,
		// tfaCodeChan is initialized by Manager when needed
	}
}

// UpdateStatus should ideally be called internally by the manager or executor
// while holding appropriate locks if needed.
func (t *Task) UpdateStatus(status TaskStatus) {
	t.Status = status
	t.UpdatedAt = time.Now().UTC()
}

// SetResult should ideally be called internally by the manager or executor.
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
	} else {
		t.Result.Error = "" // Clear previous error if setting success result
	}
	t.UpdatedAt = time.Now().UTC()
}
