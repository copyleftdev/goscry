package browser

import (
	"fmt"
	"strconv"
	"time"

	// No internal task state access needed here
	"github.com/chromedp/chromedp"
	"github.com/copyleftdev/goscry/internal/dom"
	"github.com/copyleftdev/goscry/internal/taskstypes" // Use the shared types package instead
)

// GenerateActionSequence translates a task Action into a chromedp Action.
// It takes credentials and the current tfaCode separately to avoid importing the full task state logic.
func GenerateActionSequence(taskAction taskstypes.Action, taskCreds *taskstypes.Credentials, tfaCode string) (chromedp.Action, error) {

	// Helper to resolve values like {{task.tfa_code}}
	resolveValue := func(value string) string {
		if value == "{{task.tfa_code}}" && tfaCode != "" {
			return tfaCode
		}
		return value
	}

	switch taskAction.Type {
	case taskstypes.ActionNavigate:
		if taskAction.Value == "" {
			return nil, fmt.Errorf("navigate action requires a non-empty URL value")
		}
		return dom.NavigateAction(taskAction.Value), nil

	case taskstypes.ActionWaitVisible:
		if taskAction.Selector == "" {
			return nil, fmt.Errorf("wait_visible action requires a selector")
		}
		// We need to create a context action that adds timeout to the underlying action
		return chromedp.WaitVisible(taskAction.Selector, chromedp.ByQuery), nil

	case taskstypes.ActionWaitHidden:
		if taskAction.Selector == "" {
			return nil, fmt.Errorf("wait_hidden action requires a selector")
		}
		// We need to use a simple wait action without timeout options
		return chromedp.WaitNotVisible(taskAction.Selector, chromedp.ByQuery), nil

	case taskstypes.ActionWaitDelay:
		dur, err := time.ParseDuration(taskAction.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value for wait_delay '%s': %w", taskAction.Value, err)
		}
		return chromedp.Sleep(dur), nil

	case taskstypes.ActionClick:
		if taskAction.Selector == "" {
			return nil, fmt.Errorf("click action requires a selector")
		}
		return dom.ClickAction(taskAction.Selector), nil

	case taskstypes.ActionInput: // Changed from ActionType constant name
		if taskAction.Selector == "" {
			return nil, fmt.Errorf("type action requires a selector")
		}
		resolvedValue := resolveValue(taskAction.Value)
		return dom.TypeAction(taskAction.Selector, resolvedValue), nil

	case taskstypes.ActionSelect:
		if taskAction.Selector == "" {
			return nil, fmt.Errorf("select action requires a selector")
		}
		resolvedValue := resolveValue(taskAction.Value) // Resolve value if needed
		return dom.SelectAction(taskAction.Selector, resolvedValue), nil

	case taskstypes.ActionScroll:
		if taskAction.Value == "top" {
			return chromedp.Evaluate(`window.scrollTo(0, 0)`, nil), nil
		} else if taskAction.Value == "bottom" {
			return chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil), nil
		} else if taskAction.Selector != "" {
			return dom.ScrollIntoViewAction(taskAction.Selector), nil
		}
		return nil, fmt.Errorf("invalid scroll action requires 'top', 'bottom', or a selector")

	case taskstypes.ActionScreenshot:
		// Returns an action that populates a []byte pointed to by the result arg of Run.
		// The caller (ExecuteTask) needs to provide a pointer to a byte slice.
		quality := 90 // Default quality
		if q, err := strconv.Atoi(taskAction.Value); err == nil && q >= 0 && q <= 100 {
			quality = q
		}
		// Return the screenshot action directly
		return dom.ScreenshotAction(quality, nil), nil

	case taskstypes.ActionGetDOM:
		// Returns an action that populates a string pointed to by the result arg of Run.
		// The caller (ExecuteTask) needs to provide a pointer to a string.
		sel := taskAction.Selector
		if sel == "" {
			sel = "body" // Default to body
		}
		switch taskAction.Format {
		case "full_html":
			return dom.GetOuterHTMLAction(sel, nil), nil // Expects *string in Run
		case "simplified_html":
			// Needs two steps: get raw HTML, then simplify. The caller must orchestrate this.
			// Returning just the raw fetch for now. Simplification must happen in ExecuteTask.
			// Or return a complex action. Let's return just the raw fetch.
			return dom.GetOuterHTMLAction(sel, nil), nil // Expects *string in Run
		case "text_content":
			fallthrough
		default:
			script := fmt.Sprintf(`document.querySelector('%s') ? document.querySelector('%s').innerText : document.body.innerText`, sel, sel)
			return chromedp.Evaluate(script, nil), nil // Expects *string in Run
		}

	case taskstypes.ActionRunScript:
		if taskAction.Value == "" {
			return nil, fmt.Errorf("run_script action requires script code in value")
		}
		// Returns an action that populates an interface{} pointed to by the result arg of Run.
		return dom.RunScriptAction(taskAction.Value, nil), nil // Expects *interface{} in Run

	case taskstypes.ActionLogin:
		// High-level action, requires credentials passed from the task context.
		if taskCreds == nil || taskCreds.Username == "" || taskCreds.Password == "" {
			return nil, fmt.Errorf("credentials required for login action but not provided or incomplete")
		}
		// Use generic selectors; ideally make these configurable per task/action
		userSel := "#username"
		passSel := "#password"
		submitSel := "button[type='submit'], input[type='submit']"

		// Build sequence
		loginSequence := chromedp.Tasks{
			chromedp.WaitVisible(userSel, chromedp.ByQuery),
			chromedp.SendKeys(userSel, taskCreds.Username, chromedp.ByQuery),
			chromedp.WaitVisible(passSel, chromedp.ByQuery),
			chromedp.SendKeys(passSel, taskCreds.Password, chromedp.ByQuery),
			chromedp.WaitVisible(submitSel, chromedp.ByQuery),
			chromedp.Click(submitSel, chromedp.ByQuery),
		}
		return loginSequence, nil

	default:
		return nil, fmt.Errorf("unknown action type: %s", taskAction.Type)
	}
}
