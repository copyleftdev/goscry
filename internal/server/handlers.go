package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/copyleftdev/goscry/internal/dom"
	"github.com/copyleftdev/goscry/internal/tasks"
	"github.com/copyleftdev/goscry/internal/taskstypes"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type APIHandler struct {
	taskManager *tasks.Manager
	logger      *log.Logger
}

func NewAPIHandler(tm *tasks.Manager, logger *log.Logger) *APIHandler {
	return &APIHandler{
		taskManager: tm,
		logger:      logger,
	}
}

type SubmitTaskRequest struct {
	Actions       []taskstypes.Action          `json:"actions"`
	Credentials   *taskstypes.Credentials      `json:"credentials,omitempty"` // Sent in request, handled securely
	TwoFactorAuth taskstypes.TwoFactorAuthInfo `json:"two_factor_auth"`
	CallbackURL   string                       `json:"callback_url,omitempty"`
}

type SubmitTaskResponse struct {
	TaskID string `json:"task_id"`
}

type Provide2FACodeRequest struct {
	Code string `json:"code"`
}

type GetDomASTRequest struct {
	URL            string `json:"url"`
	ParentSelector string `json:"parent_selector,omitempty"`
}

func (h *APIHandler) HandleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}
	defer r.Body.Close()

	// Create a task ID
	task := &taskstypes.Task{
		ID:            uuid.New(),
		Status:        taskstypes.StatusPending,
		Actions:       req.Actions,
		Credentials:   req.Credentials,
		TwoFactorAuth: req.TwoFactorAuth,
		CallbackURL:   req.CallbackURL,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		TfaCodeChan:   make(chan string, 1), // Buffered channel for 2FA code
	}

	// Queue the task
	err := h.taskManager.SubmitTask(task)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to submit task: %v", err)
		return
	}

	resp := SubmitTaskResponse{
		TaskID: task.ID.String(),
	}
	h.respondJSON(w, http.StatusAccepted, resp)
}

func (h *APIHandler) HandleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	task, err := h.taskManager.GetTaskStatus(taskID)
	if err != nil {
		// Check for not found error based on error message
		if errors.Is(err, fmt.Errorf("task not found")) || 
		   err.Error() == "task not found" {
			h.respondError(w, http.StatusNotFound, "Task not found")
		} else {
			h.respondError(w, http.StatusInternalServerError, "Failed to get task: %v", err)
		}
		return
	}

	h.respondJSON(w, http.StatusOK, task)
}

// HandleGetDomAST handles requests to get a DOM AST from a URL with optional parent selector
func (h *APIHandler) HandleGetDomAST(w http.ResponseWriter, r *http.Request) {
	var req GetDomASTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}
	defer r.Body.Close()

	if req.URL == "" {
		h.respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	h.logger.Printf("Processing DOM AST request for URL: %s, parent selector: %s", req.URL, req.ParentSelector)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set up ChromeDP
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.WindowSize(1280, 1024),
	)

	// Create allocator
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Initialize result
	var domAST dom.DomNode

	// Run the DOM AST action
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(req.URL),
		chromedp.Sleep(5*time.Second), // Increased wait time to ensure page loads fully
		dom.GetDomASTAction(req.ParentSelector, &domAST),
	)

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to get DOM AST: %v", err)
		return
	}

	h.respondJSON(w, http.StatusOK, domAST)
}

func (h *APIHandler) HandleProvide2FACode(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	var req Provide2FACodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}
	defer r.Body.Close()

	if req.Code == "" {
		h.respondError(w, http.StatusBadRequest, "2FA code is required")
		return
	}

	task, err := h.taskManager.GetTaskStatus(taskID)
	if err != nil {
		// Check for not found error based on error message
		if errors.Is(err, fmt.Errorf("task not found")) || 
		   err.Error() == "task not found" {
			h.respondError(w, http.StatusNotFound, "Task not found")
		} else {
			h.respondError(w, http.StatusInternalServerError, "Failed to get task: %v", err)
		}
		return
	}

	if string(task.Status) != string(tasks.StatusWaitingFor2FA) {
		h.respondError(w, http.StatusBadRequest, "Task is not waiting for 2FA")
		return
	}

	err = h.taskManager.Provide2FACode(taskID, req.Code)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to provide 2FA code: %v", err)
		return
	}

	h.respondJSON(w, http.StatusAccepted, map[string]string{"status": "2FA code accepted"})
}

// --- Helper Functions ---

func (h *APIHandler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Printf("Error marshalling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func (h *APIHandler) respondError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	h.logger.Printf("Error response: %s", message)

	response, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		h.logger.Printf("Error marshalling error response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}
