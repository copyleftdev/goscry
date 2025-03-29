package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/copyleftdev/goscry/internal/config"
	"github.com/copyleftdev/goscry/internal/dom"
	"github.com/copyleftdev/goscry/internal/tasks"
	"github.com/copyleftdev/goscry/internal/taskstypes"
	"golang.org/x/sync/semaphore"
)

// Compile-time check to ensure Manager implements the interface
var _ tasks.BrowserExecutor = (*Manager)(nil)

type Manager struct {
	allocatorCtx    context.Context
	allocatorCancel context.CancelFunc
	cfg             *config.BrowserConfig
	logger          *log.Logger
	sem             *semaphore.Weighted
	activeCtxWg     sync.WaitGroup
}

func NewManager(cfg *config.BrowserConfig, logger *log.Logger) (*Manager, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("mute-audio", true),
		chromedp.IgnoreCertErrors,
	)

	if cfg.ExecutablePath != "" {
		opts = append(opts, chromedp.ExecPath(cfg.ExecutablePath))
	}
	if cfg.UserDataDir != "" {
		opts = append(opts, chromedp.UserDataDir(cfg.UserDataDir))
	} else {
		opts = append(opts, chromedp.Flag("guest", true))
	}

	// Store context and its cancel func
	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	return &Manager{
		allocatorCtx:    allocatorCtx,
		allocatorCancel: cancel,
		cfg:             cfg,
		logger:          logger,
		sem:             semaphore.NewWeighted(int64(cfg.MaxSessions)),
	}, nil
}

// ExecuteTask implements the tasks.BrowserExecutor interface.
func (m *Manager) ExecuteTask(task *taskstypes.Task) (*taskstypes.TaskResult, error) {
	// Create a context with timeout for this task execution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Default timeout
	defer cancel()

	// Acquire a browser slot from our semaphore
	if err := m.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("failed to acquire browser slot: %w", err)
	}
	defer m.sem.Release(1)

	// Track this active browser context for graceful shutdown
	m.activeCtxWg.Add(1)
	defer m.activeCtxWg.Done()

	// Create a new browser context for this task
	browserCtx, browserCancel := chromedp.NewContext(
		m.allocatorCtx,
		chromedp.WithLogf(m.logger.Printf),
	)
	defer browserCancel()

	// Store the task's browser context ID for future reference if needed
	if chromeTarget := chromedp.FromContext(browserCtx); chromeTarget != nil && chromeTarget.Target != nil {
		task.BrowserContextID = chromeTarget.Target.TargetID.String()
	} else {
		m.logger.Printf("Warning: Could not get Target ID, browser context might not be fully initialized")
		// Set a placeholder value instead of nil
		task.BrowserContextID = "unknown"
	}

	// Initialize the result
	result := &taskstypes.TaskResult{
		Success: true,
		Message: "Task completed successfully",
	}

	// Execute each action in sequence until done or error
	for i, action := range task.Actions {
		// Update current action index
		task.CurrentAction = i

		// Generate the chromedp action from task action
		chromedpAction, err := GenerateActionSequence(action, task.Credentials, "")
		if err != nil {
			result.Success = false
			result.Message = "Failed to generate action"
			result.Error = err.Error()
			return result, err
		}

		// We might need to handle 2FA during execution
		if action.Type == taskstypes.ActionNavigate || action.Type == taskstypes.ActionClick {
			// Execute with potential 2FA checks
			err = m.executeWithPotential2FA(browserCtx, chromedpAction, task)
		} else {
			// Normal execution for other action types
			err = chromedp.Run(browserCtx, chromedpAction)
		}

		// Handle action execution failure
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Failed on action %d: %s", i, action.Type)
			result.Error = err.Error()
			return result, err
		}
	}

	// All actions completed successfully
	return result, nil
}

// executeWithPotential2FA runs an action and checks for 2FA prompts
func (m *Manager) executeWithPotential2FA(ctx context.Context, action chromedp.Action, task *taskstypes.Task) error {
	// Run the action first
	if err := chromedp.Run(ctx, action); err != nil {
		return err
	}

	// After navigation or click, check if we now have a 2FA prompt
	if is2FA, promptType, err := m.detect2FAPrompt(ctx); err != nil {
		m.logger.Printf("Error checking for 2FA: %v", err)
	} else if is2FA {
		m.logger.Printf("Detected 2FA prompt type: %s", promptType)

		// Update task status to waiting for 2FA
		task.Status = taskstypes.StatusWaitingFor2FA

		// Wait for 2FA code to be provided
		code, err := task.WaitForTFACode(ctx)
		if err != nil {
			return fmt.Errorf("2FA code wait error: %w", err)
		}

		// We have a code, let's try to input it
		// Find the 2FA input field
		var selector string
		switch promptType {
		case "input":
			selector = "input[type='text'], input[type='number'], input[type='tel']"
		case "button":
			// Some 2FA might just need a button click, no input
			selector = "button.confirm, button.verify"
		default:
			// Fall back to common selectors
			selector = "input[name='code'], input[placeholder*='code'], input[aria-label*='code']"
		}

		// Enter the code
		if selector != "" {
			if err := chromedp.Run(ctx, chromedp.Tasks{
				chromedp.WaitVisible(selector),
				chromedp.Clear(selector),
				chromedp.SendKeys(selector, code),
				chromedp.Submit(selector),
			}); err != nil {
				return fmt.Errorf("failed to input 2FA code: %w", err)
			}
		}

		// Update task status back to running
		task.Status = taskstypes.StatusRunning
	}

	return nil
}

func (m *Manager) detect2FAPrompt(ctx context.Context) (bool, string, error) {
	tfaSelectors := []string{
		"input[name='otp']", "input[name='security_code']", "input[autocomplete='one-time-code']",
		"#verification_code", "input[id*='2fa']", "input[id*='mfa']",
	}
	tfaTextPatterns := []string{
		"enter verification code", "two-factor authentication", "security code", "enter the code",
	}

	var isPresent bool
	var details string = "Unknown 2FA prompt"

	// Check selectors first
	for _, selector := range tfaSelectors {
		checkAction := dom.IsElementPresentAction(selector, &isPresent)
		if err := chromedp.Run(ctx, checkAction); err == nil && isPresent {
			details = fmt.Sprintf("Detected via selector: %s", selector)
			return true, details, nil
		} else if err != nil {
			m.logger.Printf("Error checking 2FA selector %s: %v", selector, err) // Log non-critical error
		}
	}

	// Check text content if no selector matched
	var pageText string
	getTextAction := dom.GetTextContentAction(&pageText)
	if err := chromedp.Run(ctx, getTextAction); err == nil {
		pageTextLower := strings.ToLower(pageText)
		for _, pattern := range tfaTextPatterns {
			if strings.Contains(pageTextLower, pattern) {
				details = fmt.Sprintf("Detected via text: %s", pattern)
				return true, details, nil
			}
		}
	} else {
		m.logger.Printf("Error getting page text for 2FA check: %v", err) // Log non-critical error
	}

	return false, "", nil // No prompt detected
}

// Shutdown implements the tasks.BrowserExecutor interface.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Println("Shutting down browser manager...")

	// Signal allocator context to cancel
	if m.allocatorCancel != nil {
		m.allocatorCancel()
	}

	// Wait for active ExecuteTask calls to finish or timeout
	shutdownComplete := make(chan struct{})
	go func() {
		m.activeCtxWg.Wait() // Wait for all ExecuteTask goroutines to release semaphore/finish
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		m.logger.Println("All active browser sessions have finished.")
	case <-ctx.Done():
		m.logger.Println("Shutdown timeout reached while waiting for active browser sessions.")
		return ctx.Err()
	}

	// Allocator shutdown is handled by cancelling its context.
	m.logger.Println("Browser manager shutdown complete.")
	return nil
}

// --- Helper Actions (Potentially moved to dom package or kept here) ---
// These were previously defined but might not be needed directly by Manager anymore,
// as action execution is now encapsulated. Keep Getters if status checks need them.

func (m *Manager) GetCurrentURLAction(url *string) chromedp.Action {
	return chromedp.Location(url)
}

func (m *Manager) GetPageTitleAction(title *string) chromedp.Action {
	return chromedp.Title(title)
}

// --- Cookie/Storage Helpers (Can be exposed via Manager if needed by API directly) ---

func (m *Manager) GetCookiesAction(cookies *[]*network.Cookie) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		c, err := network.GetCookies().Do(ctx)
		if err != nil {
			return err
		}
		*cookies = c
		return nil
	})
}

func (m *Manager) SetCookiesAction(cookies []*network.CookieParam) chromedp.Action {
	return network.SetCookies(cookies)
}

func (m *Manager) ClearCookiesAction() chromedp.Action {
	return network.ClearBrowserCookies()
}

// Other storage actions (Get/Set Local/Session Storage) would follow similar patterns using chromedp.Evaluate
