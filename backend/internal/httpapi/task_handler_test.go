package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/repository"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize test schema: %v", err)
	}

	return NewRouter(repository.NewTaskRepository(db))
}

func TestCreateTask(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Need warehouse automation","topic":"logistics"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var task model.Task
	if err := json.NewDecoder(recorder.Body).Decode(&task); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if task.ID <= 0 {
		t.Fatalf("expected positive task ID, got %d", task.ID)
	}
	if task.InitialDescription != "Need warehouse automation" {
		t.Fatalf("unexpected initial description %q", task.InitialDescription)
	}
	if task.ReadinessLevel != "draft" || task.Rating != 0 || task.Confirmed || task.Published {
		t.Fatalf("unexpected initial task state: %+v", task)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing description", body: `{"topic":"logistics"}`},
		{name: "whitespace description", body: `{"initial_description":"   "}`},
		{name: "invalid JSON", body: `{"initial_description":`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(test.body))
			recorder := httptest.NewRecorder()

			newTestRouter(t).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestCreateTaskRejectsInternalFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{
		"initial_description":"Need warehouse automation",
		"rating":5,
		"readiness_level":"ready",
		"confirmed":true,
		"published":true,
		"id":42,
		"created_at":"2026-01-01T00:00:00Z",
		"updated_at":"2026-01-01T00:00:00Z"
	}`))
	recorder := httptest.NewRecorder()

	newTestRouter(t).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
}

func TestHealthMethodNotAllowedReturnsJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	recorder := httptest.NewRecorder()

	NewRouter(nil).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var response struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error != "method not allowed" {
		t.Fatalf("expected error %q, got %q", "method not allowed", response.Error)
	}
}

func TestGetTaskByID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tasks.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize test schema: %v", err)
	}

	repo := repository.NewTaskRepository(db)
	task := &model.Task{InitialDescription: "Need a task"}
	if err := repo.Create(t.Context(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	router := NewRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(task.ID, 10), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var got model.Task
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if got.ID != task.ID {
		t.Fatalf("expected ID %d, got %d", task.ID, got.ID)
	}
}

func TestGetTaskByIDErrors(t *testing.T) {
	router := newTestRouter(t)

	tests := []struct {
		name       string
		path       string
		expectCode int
		expectBody string
	}{
		{name: "missing task", path: "/api/tasks/999", expectCode: http.StatusNotFound, expectBody: "task not found"},
		{name: "invalid task ID", path: "/api/tasks/invalid", expectCode: http.StatusBadRequest, expectBody: "invalid task id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != test.expectCode {
				t.Fatalf("expected status %d, got %d", test.expectCode, recorder.Code)
			}

			var response struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Error != test.expectBody {
				t.Fatalf("expected error %q, got %q", test.expectBody, response.Error)
			}
		})
	}
}
