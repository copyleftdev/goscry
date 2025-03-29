package tasks

import (
	"context"
	"github.com/copyleftdev/goscry/internal/taskstypes"
)

// BrowserExecutor defines the interface for executing browser tasks.
// This decouples the task manager from the specific browser implementation.
type BrowserExecutor interface {
	// ExecuteTask runs the browser actions defined within the task.
	// It should handle the entire lifecycle for the browser part of the task,
	// including potential 2FA waits.
	// Returns a result object and an error if the execution fails.
	ExecuteTask(task *taskstypes.Task) (*taskstypes.TaskResult, error)

	// Shutdown allows for graceful cleanup of browser resources if needed at this level.
	Shutdown(ctx context.Context) error
}
