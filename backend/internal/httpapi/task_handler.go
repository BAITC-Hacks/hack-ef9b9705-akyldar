package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/model"
	"backend/internal/repository"
)

type taskHandler struct {
	repository *repository.TaskRepository
}

func newTaskHandler(taskRepository *repository.TaskRepository) *taskHandler {
	return &taskHandler{repository: taskRepository}
}

type createTaskRequest struct {
	InitialDescription string `json:"initial_description"`
	Topic              string `json:"topic"`
}

func (h *taskHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request createTaskRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	request.InitialDescription = strings.TrimSpace(request.InitialDescription)
	if request.InitialDescription == "" {
		writeJSONError(w, http.StatusBadRequest, "initial_description is required")
		return
	}

	task := &model.Task{
		InitialDescription: request.InitialDescription,
		Topic:              request.Topic,
	}
	if err := h.repository.Create(r.Context(), task); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *taskHandler) handleByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rawID := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if rawID == "" || strings.Contains(rawID, "/") {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.repository.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrTaskNotFound) {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}
