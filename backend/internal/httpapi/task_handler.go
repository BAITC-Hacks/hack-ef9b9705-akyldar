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

type updateTaskRequest struct {
	Title             string `json:"title"`
	Context           string `json:"context"`
	Need              string `json:"need"`
	Users             string `json:"users"`
	Data              string `json:"data"`
	Constraints       string `json:"constraints"`
	ExpectedResult    string `json:"expected_result"`
	SuccessCriteria   string `json:"success_criteria"`
	Contact           string `json:"contact"`
	InteractionFormat string `json:"interaction_format"`
	Topic             string `json:"topic"`
}

func (h *taskHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request createTaskRequest
	if err := decodeJSONBody(r.Body, &request); err != nil {
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
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if r.Method == http.MethodPut {
		h.updateByID(w, r, id)
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

func (h *taskHandler) updateByID(w http.ResponseWriter, r *http.Request, id int64) {
	var request updateTaskRequest
	if err := decodeJSONBody(r.Body, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
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

	task.Title = request.Title
	task.Context = request.Context
	task.Need = request.Need
	task.Users = request.Users
	task.Data = request.Data
	task.Constraints = request.Constraints
	task.ExpectedResult = request.ExpectedResult
	task.SuccessCriteria = request.SuccessCriteria
	task.Contact = request.Contact
	task.InteractionFormat = request.InteractionFormat
	task.Topic = request.Topic

	if err := h.repository.Update(r.Context(), task); err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			writeJSONError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func parseTaskID(path string) (int64, error) {
	rawID := strings.TrimPrefix(path, "/api/tasks/")
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, errors.New("invalid task id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}

func decodeJSONBody(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request must contain a single JSON object")
	}

	return nil
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
