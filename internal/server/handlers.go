package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/copyleftdev/goscry/internal/tasks"
	"github.com/copyleftdev/goscry/internal/taskstypes"
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
	CallbackURL   string                      `json:"callback_url,omitempty"`
}

type SubmitTaskResponse struct {
	TaskID string `json:"task_id"`
}

type Provide2FACodeRequest struct {
	Code string `json:"code"`
}

func (h *APIHandler) HandleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}
	defer r.Body.Close()

	if len(req.Actions) == 0 {
		h.respondError(w, http.StatusBadRequest, "Task must contain at least one action")
		return
	}

	// Note: req.Credentials contains sensitive data. Avoid logging it directly.
	// Create a new Task using taskstypes
	task := &taskstypes.Task{
		ID:            uuid.New(),
		Status:        taskstypes.StatusPending,
		Actions:       req.Actions,
		Credentials:   req.Credentials,
		TwoFactorAuth: req.TwoFactorAuth,
		CallbackURL:   req.CallbackURL,
		TfaCodeChan:   make(chan string, 1),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := h.taskManager.SubmitTask(task)
	if err != nil {
		// Handle specific errors, e.g., duplicate ID, although unlikely with UUIDs here
		h.logger.Printf("Error submitting task: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to submit task: %v", err)
		return
	}

	h.logger.Printf("Submitted new task with ID: %s", task.ID.String())
	h.respondJSON(w, http.StatusAccepted, SubmitTaskResponse{TaskID: task.ID.String()})
}

func (h *APIHandler) HandleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid task ID format: %v", err)
		return
	}

	task, err := h.taskManager.GetTaskStatus(taskID)
	if err != nil {
		// Assuming GetTaskStatus returns a specific error type for not found
		// For now, check the error string (improve with typed errors later)
		if errors.Is(err, fmt.Errorf("task with ID %s not found", taskIDStr)) || strings.Contains(err.Error(), "not found") {
			h.respondError(w, http.StatusNotFound, "Task not found")
		} else {
			h.logger.Printf("Error retrieving task status for %s: %v", taskIDStr, err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve task status")
		}
		return
	}

	h.respondJSON(w, http.StatusOK, task)
}

func (h *APIHandler) HandleProvide2FACode(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid task ID format: %v", err)
		return
	}

	var req Provide2FACodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}
	defer r.Body.Close()

	if req.Code == "" {
		h.respondError(w, http.StatusBadRequest, "2FA code cannot be empty")
		return
	}

	err = h.taskManager.Provide2FACode(taskID, req.Code)
	if err != nil {
		// Check error type for better status codes
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, http.StatusNotFound, "%s", err.Error())
		} else if strings.Contains(err.Error(), "is not waiting for 2FA code") {
			h.respondError(w, http.StatusConflict, "%s", err.Error()) // 409 Conflict might fit
		} else if strings.Contains(err.Error(), "failed to signal 2FA code") {
			h.respondError(w, http.StatusRequestTimeout, "%s", err.Error()) // 408 might indicate timeout
		} else {
			h.logger.Printf("Error providing 2FA code for task %s: %v", taskIDStr, err)
			h.respondError(w, http.StatusInternalServerError, "%s", err.Error())
		}
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "2FA code received"})
}

// --- Helper Functions ---

func (h *APIHandler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Printf("Error marshalling JSON response: %v", err)
		// Fallback to plain text error response
		h.respondError(w, http.StatusInternalServerError, "Failed to marshal JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(response)
	if err != nil {
		h.logger.Printf("Error writing JSON response: %v", err)
	}
}

func (h *APIHandler) respondError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	errorMessage := fmt.Sprintf(format, args...)
	response := map[string]string{"error": errorMessage}
	jsonResponse, err := json.Marshal(response)
	// If marshalling fails, send plain text error
	contentType := "application/json; charset=utf-8"
	if err != nil {
		h.logger.Printf("Error marshalling JSON error response: %v", err)
		jsonResponse = []byte(fmt.Sprintf(`{"error":"%s"}`, errorMessage)) // Basic fallback
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, writeErr := w.Write(jsonResponse)
	if writeErr != nil {
		h.logger.Printf("Error writing error response: %v", writeErr)
	}
}
