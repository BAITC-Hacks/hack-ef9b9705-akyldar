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
	"backend/internal/rating"
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

	return NewRouter(repository.NewTaskRepository(db), repository.NewTeamRepository(db), repository.NewProposalRepository(db))
}

func newCatalogTestRouter(t *testing.T) (http.Handler, *repository.TaskRepository) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatalf("open catalog test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize catalog test schema: %v", err)
	}

	repo := repository.NewTaskRepository(db)
	return NewRouter(repo, repository.NewTeamRepository(db), repository.NewProposalRepository(db)), repo
}

func newProposalTestRouter(t *testing.T) (http.Handler, *repository.TeamRepository) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "proposals.db"))
	if err != nil {
		t.Fatalf("open proposal test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize proposal test schema: %v", err)
	}

	taskRepository := repository.NewTaskRepository(db)
	teamRepository := repository.NewTeamRepository(db)
	proposalRepository := repository.NewProposalRepository(db)
	return NewRouter(taskRepository, teamRepository, proposalRepository), teamRepository
}

func createHTTPTask(t *testing.T, router http.Handler, description string) model.Task {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"`+description+`"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected task creation status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var task model.Task
	if err := json.NewDecoder(recorder.Body).Decode(&task); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	return task
}

func publishHTTPTask(t *testing.T, router http.Handler, taskID int64) {
	t.Helper()
	path := "/api/tasks/" + strconv.FormatInt(taskID, 10)
	confirmRequest := httptest.NewRequest(http.MethodPost, path+"/confirm", nil)
	confirmRecorder := httptest.NewRecorder()
	router.ServeHTTP(confirmRecorder, confirmRequest)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected confirm status %d, got %d", http.StatusOK, confirmRecorder.Code)
	}

	publishRequest := httptest.NewRequest(http.MethodPost, path+"/publish", nil)
	publishRecorder := httptest.NewRecorder()
	router.ServeHTTP(publishRecorder, publishRequest)
	if publishRecorder.Code != http.StatusOK {
		t.Fatalf("expected publish status %d, got %d", http.StatusOK, publishRecorder.Code)
	}
}

func seedPublishedCatalogTask(t *testing.T, repo *repository.TaskRepository, task *model.Task) {
	t.Helper()

	if err := repo.Create(t.Context(), task); err != nil {
		t.Fatalf("create catalog task: %v", err)
	}
	calculation := rating.Calculate(*task)
	task.Rating = calculation.Score
	task.ReadinessLevel = calculation.Level
	if err := repo.Update(t.Context(), task); err != nil {
		t.Fatalf("update catalog task rating: %v", err)
	}
	if err := repo.Confirm(t.Context(), task); err != nil {
		t.Fatalf("confirm catalog task: %v", err)
	}
	if err := repo.Publish(t.Context(), task); err != nil {
		t.Fatalf("publish catalog task: %v", err)
	}
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

	NewRouter(nil, nil, nil).ServeHTTP(recorder, req)

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

	router := NewRouter(repo, repository.NewTeamRepository(db), repository.NewProposalRepository(db))
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

func TestUpdateTask(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Original draft description","topic":"original-topic"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}

	updateBody := `{
		"title":"Warehouse automation system",
		"context":"Warehouse operations are tracked manually.",
		"need":"Reduce manual work and inventory errors.",
		"users":"Warehouse employees",
		"data":"Existing inventory CSV files",
		"constraints":"Prototype must run locally",
		"expected_result":"Working inventory management prototype",
		"success_criteria":"Reduce manual inventory operations",
		"contact":"business@example.com",
		"interaction_format":"Weekly consultation",
		"topic":"logistics"
	}`
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/tasks/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(updateBody))
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d", http.StatusOK, updateRecorder.Code)
	}

	var updated model.Task
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated task: %v", err)
	}
	if updated.Title != "Warehouse automation system" || updated.Topic != "logistics" {
		t.Fatalf("updated values missing from response: %+v", updated)
	}
	if updated.InitialDescription != "Original draft description" {
		t.Fatalf("expected initial description to remain unchanged, got %q", updated.InitialDescription)
	}
	if updated.Rating != 100 || updated.ReadinessLevel != "priority" {
		t.Fatalf("expected recalculated priority rating, got rating=%d level=%q", updated.Rating, updated.ReadinessLevel)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(created.ID, 10), nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d", http.StatusOK, getRecorder.Code)
	}

	var fetched model.Task
	if err := json.NewDecoder(getRecorder.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched task: %v", err)
	}
	if fetched.Title != updated.Title || fetched.Context != updated.Context || fetched.Topic != updated.Topic {
		t.Fatalf("GET did not return updated values: %+v", fetched)
	}
}

func TestUpdateTaskErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		expectCode int
	}{
		{name: "invalid ID", path: "/api/tasks/invalid", body: `{}`, expectCode: http.StatusBadRequest},
		{name: "missing task", path: "/api/tasks/999", body: `{}`, expectCode: http.StatusNotFound},
		{name: "malformed JSON", path: "/api/tasks/1", body: `{"title":`, expectCode: http.StatusBadRequest},
		{name: "unknown rating field", path: "/api/tasks/1", body: `{"title":"Example","rating":100}`, expectCode: http.StatusBadRequest},
		{name: "unknown readiness field", path: "/api/tasks/1", body: `{"title":"Example","readiness_level":"priority"}`, expectCode: http.StatusBadRequest},
		{name: "initial description field", path: "/api/tasks/1", body: `{"initial_description":"Changed"}`, expectCode: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, test.path, bytes.NewBufferString(test.body))
			recorder := httptest.NewRecorder()
			newTestRouter(t).ServeHTTP(recorder, req)

			if recorder.Code != test.expectCode {
				t.Fatalf("expected status %d, got %d", test.expectCode, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestUpdateTaskRecalculatesRating(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Original draft description"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}

	put := func(body string) model.Task {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/tasks/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(body))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected update status %d, got %d", http.StatusOK, recorder.Code)
		}

		var updated model.Task
		if err := json.NewDecoder(recorder.Body).Decode(&updated); err != nil {
			t.Fatalf("decode updated task: %v", err)
		}
		return updated
	}

	first := put(`{"context":"Warehouse operations"}`)
	if first.Rating != 10 || first.ReadinessLevel != "draft" {
		t.Fatalf("expected context-only rating 10/draft, got %d/%q", first.Rating, first.ReadinessLevel)
	}

	second := put(`{"context":"Warehouse operations","need":"Reduce manual work"}`)
	if second.Rating != 20 || second.ReadinessLevel != "draft" {
		t.Fatalf("expected context-and-need rating 20/draft, got %d/%q", second.Rating, second.ReadinessLevel)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(created.ID, 10), nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d", http.StatusOK, getRecorder.Code)
	}
	var fetched model.Task
	if err := json.NewDecoder(getRecorder.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched task: %v", err)
	}
	if fetched.Rating != 20 || fetched.ReadinessLevel != "draft" {
		t.Fatalf("expected persisted rating 20/draft, got %d/%q", fetched.Rating, fetched.ReadinessLevel)
	}
}

func TestGetTaskRating(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Original draft description"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/api/tasks/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(`{"context":"Context","need":"Need","data":"Data"}`))
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d", http.StatusOK, updateRecorder.Code)
	}

	ratingRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(created.ID, 10)+"/rating", nil)
	ratingRecorder := httptest.NewRecorder()
	router.ServeHTTP(ratingRecorder, ratingRequest)
	if ratingRecorder.Code != http.StatusOK {
		t.Fatalf("expected rating status %d, got %d", http.StatusOK, ratingRecorder.Code)
	}

	var result rating.Result
	if err := json.NewDecoder(ratingRecorder.Body).Decode(&result); err != nil {
		t.Fatalf("decode rating response: %v", err)
	}
	if result.Score != 40 || result.Level != "working" {
		t.Fatalf("expected rating 40/working, got %d/%q", result.Score, result.Level)
	}
	if result.Breakdown["context_need"] != 20 || result.Breakdown["data"] != 20 {
		t.Fatalf("unexpected rating breakdown: %+v", result.Breakdown)
	}
	if len(result.Missing) != 6 || result.Missing[0] != "expected_result" {
		t.Fatalf("unexpected missing fields: %v", result.Missing)
	}
}

func TestGetTaskRatingErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectCode int
	}{
		{name: "invalid ID", path: "/api/tasks/invalid/rating", expectCode: http.StatusBadRequest},
		{name: "non-positive ID", path: "/api/tasks/0/rating", expectCode: http.StatusBadRequest},
		{name: "missing task", path: "/api/tasks/999/rating", expectCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			newTestRouter(t).ServeHTTP(recorder, req)

			if recorder.Code != test.expectCode {
				t.Fatalf("expected status %d, got %d", test.expectCode, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestConfirmAndPublishTask(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Low-rated task"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	taskPath := "/api/tasks/" + strconv.FormatInt(created.ID, 10)

	confirmRequest := httptest.NewRequest(http.MethodPost, taskPath+"/confirm", nil)
	confirmRecorder := httptest.NewRecorder()
	router.ServeHTTP(confirmRecorder, confirmRequest)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected confirm status %d, got %d", http.StatusOK, confirmRecorder.Code)
	}

	var confirmed model.Task
	if err := json.NewDecoder(confirmRecorder.Body).Decode(&confirmed); err != nil {
		t.Fatalf("decode confirmed task: %v", err)
	}
	if !confirmed.Confirmed || confirmed.Published {
		t.Fatalf("unexpected confirmation state: confirmed=%t published=%t", confirmed.Confirmed, confirmed.Published)
	}
	if confirmed.Rating != 0 || confirmed.ReadinessLevel != "draft" {
		t.Fatalf("unexpected low-rating confirmation state: rating=%d level=%q", confirmed.Rating, confirmed.ReadinessLevel)
	}

	confirmAgainRequest := httptest.NewRequest(http.MethodPost, taskPath+"/confirm", nil)
	confirmAgainRecorder := httptest.NewRecorder()
	router.ServeHTTP(confirmAgainRecorder, confirmAgainRequest)
	if confirmAgainRecorder.Code != http.StatusOK {
		t.Fatalf("expected repeated confirm status %d, got %d", http.StatusOK, confirmAgainRecorder.Code)
	}

	publishRequest := httptest.NewRequest(http.MethodPost, taskPath+"/publish", nil)
	publishRecorder := httptest.NewRecorder()
	router.ServeHTTP(publishRecorder, publishRequest)
	if publishRecorder.Code != http.StatusOK {
		t.Fatalf("expected publish status %d, got %d", http.StatusOK, publishRecorder.Code)
	}

	var published model.Task
	if err := json.NewDecoder(publishRecorder.Body).Decode(&published); err != nil {
		t.Fatalf("decode published task: %v", err)
	}
	if !published.Confirmed || !published.Published {
		t.Fatalf("unexpected publication state: confirmed=%t published=%t", published.Confirmed, published.Published)
	}
	if published.Rating != confirmed.Rating || published.ReadinessLevel != confirmed.ReadinessLevel {
		t.Fatalf("publication changed rating state: before=%d/%q after=%d/%q", confirmed.Rating, confirmed.ReadinessLevel, published.Rating, published.ReadinessLevel)
	}

	publishAgainRequest := httptest.NewRequest(http.MethodPost, taskPath+"/publish", nil)
	publishAgainRecorder := httptest.NewRecorder()
	router.ServeHTTP(publishAgainRecorder, publishAgainRequest)
	if publishAgainRecorder.Code != http.StatusOK {
		t.Fatalf("expected repeated publish status %d, got %d", http.StatusOK, publishAgainRecorder.Code)
	}
}

func TestConfirmRecalculatesRating(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Structured task"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	taskPath := "/api/tasks/" + strconv.FormatInt(created.ID, 10)

	updateRequest := httptest.NewRequest(http.MethodPut, taskPath, bytes.NewBufferString(`{"context":"Context","need":"Need","data":"Data"}`))
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d", http.StatusOK, updateRecorder.Code)
	}

	confirmRequest := httptest.NewRequest(http.MethodPost, taskPath+"/confirm", nil)
	confirmRecorder := httptest.NewRecorder()
	router.ServeHTTP(confirmRecorder, confirmRequest)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected confirm status %d, got %d", http.StatusOK, confirmRecorder.Code)
	}

	var confirmed model.Task
	if err := json.NewDecoder(confirmRecorder.Body).Decode(&confirmed); err != nil {
		t.Fatalf("decode confirmed task: %v", err)
	}
	if confirmed.Rating != 40 || confirmed.ReadinessLevel != "working" {
		t.Fatalf("expected recalculated rating 40/working, got %d/%q", confirmed.Rating, confirmed.ReadinessLevel)
	}
}

func TestPublishRequiresConfirmation(t *testing.T) {
	router := newTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Unconfirmed task"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	var created model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(created.ID, 10)+"/publish", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
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
	if response.Error != "task must be confirmed before publishing" {
		t.Fatalf("unexpected error %q", response.Error)
	}
}

func TestConfirmAndPublishTaskErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectCode int
	}{
		{name: "invalid confirm ID", path: "/api/tasks/invalid/confirm", expectCode: http.StatusBadRequest},
		{name: "invalid publish ID", path: "/api/tasks/0/publish", expectCode: http.StatusBadRequest},
		{name: "missing confirm task", path: "/api/tasks/999/confirm", expectCode: http.StatusNotFound},
		{name: "missing publish task", path: "/api/tasks/999/publish", expectCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, test.path, nil)
			recorder := httptest.NewRecorder()
			newTestRouter(t).ServeHTTP(recorder, req)

			if recorder.Code != test.expectCode {
				t.Fatalf("expected status %d, got %d", test.expectCode, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestListTasksCatalog(t *testing.T) {
	router, repo := newCatalogTestRouter(t)

	unpublished := &model.Task{Title: "Unpublished", Topic: "logistics"}
	if err := repo.Create(t.Context(), unpublished); err != nil {
		t.Fatalf("create unpublished task: %v", err)
	}

	seedPublishedCatalogTask(t, repo, &model.Task{Title: "Draft", Topic: "logistics", Context: "Context", Need: "Need"})
	seedPublishedCatalogTask(t, repo, &model.Task{Title: "Working", Topic: "logistics", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result"})
	seedPublishedCatalogTask(t, repo, &model.Task{Title: "Ready", Topic: "operations", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result", Constraints: "Constraints", Users: "Users"})
	seedPublishedCatalogTask(t, repo, &model.Task{Title: "Priority", Topic: "logistics", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result", SuccessCriteria: "Criteria", Constraints: "Constraints", Users: "Users", Contact: "contact@example.com"})

	request := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected catalog status %d, got %d", http.StatusOK, recorder.Code)
	}

	var tasks []model.Task
	if err := json.NewDecoder(recorder.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("expected 4 published tasks, got %d", len(tasks))
	}
	if tasks[0].Title != "Priority" || tasks[3].Title != "Draft" {
		t.Fatalf("expected newest-first catalog order, got %+v", tasks)
	}
	for _, task := range tasks {
		if task.ID == unpublished.ID {
			t.Fatalf("unpublished task was returned: %+v", task)
		}
	}

	sortedRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?sort=rating", nil)
	sortedRecorder := httptest.NewRecorder()
	router.ServeHTTP(sortedRecorder, sortedRequest)
	var sorted []model.Task
	if err := json.NewDecoder(sortedRecorder.Body).Decode(&sorted); err != nil {
		t.Fatalf("decode sorted catalog: %v", err)
	}
	for index, expected := range []int{95, 75, 55, 20} {
		if sorted[index].Rating != expected {
			t.Fatalf("expected rating %d at index %d, got %d", expected, index, sorted[index].Rating)
		}
	}

	topicRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?topic=%20logistics%20", nil)
	topicRecorder := httptest.NewRecorder()
	router.ServeHTTP(topicRecorder, topicRequest)
	var logistics []model.Task
	if err := json.NewDecoder(topicRecorder.Body).Decode(&logistics); err != nil {
		t.Fatalf("decode topic catalog: %v", err)
	}
	if len(logistics) != 3 {
		t.Fatalf("expected 3 logistics tasks, got %d", len(logistics))
	}

	for _, level := range []string{"draft", "working", "ready", "priority"} {
		levelRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?level="+level, nil)
		levelRecorder := httptest.NewRecorder()
		router.ServeHTTP(levelRecorder, levelRequest)
		var levelTasks []model.Task
		if err := json.NewDecoder(levelRecorder.Body).Decode(&levelTasks); err != nil {
			t.Fatalf("decode %s catalog: %v", level, err)
		}
		if len(levelTasks) != 1 || levelTasks[0].ReadinessLevel != level {
			t.Fatalf("unexpected %s catalog: %+v", level, levelTasks)
		}
	}

	combinedRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?topic=logistics&level=priority&sort=rating", nil)
	combinedRecorder := httptest.NewRecorder()
	router.ServeHTTP(combinedRecorder, combinedRequest)
	var combined []model.Task
	if err := json.NewDecoder(combinedRecorder.Body).Decode(&combined); err != nil {
		t.Fatalf("decode combined catalog: %v", err)
	}
	if len(combined) != 1 || combined[0].Title != "Priority" {
		t.Fatalf("unexpected combined catalog: %+v", combined)
	}

	emptyRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?topic=missing", nil)
	emptyRecorder := httptest.NewRecorder()
	router.ServeHTTP(emptyRecorder, emptyRequest)
	if emptyRecorder.Code != http.StatusOK {
		t.Fatalf("expected empty catalog status %d, got %d", http.StatusOK, emptyRecorder.Code)
	}
	var empty []model.Task
	if err := json.NewDecoder(emptyRecorder.Body).Decode(&empty); err != nil {
		t.Fatalf("decode empty catalog: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected empty array, got %+v", empty)
	}
}

func TestListTasksCatalogValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "invalid sort", path: "/api/tasks?sort=abc"},
		{name: "invalid level", path: "/api/tasks?level=abc"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
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

func TestProposalEndpoints(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	teamOne := &model.Team{Name: "Team One", Interests: []string{"logistics"}}
	teamTwo := &model.Team{Name: "Team Two", Skills: []string{"backend"}}
	if err := teamRepository.CreateTeam(t.Context(), teamOne); err != nil {
		t.Fatalf("create first team: %v", err)
	}
	if err := teamRepository.CreateTeam(t.Context(), teamTwo); err != nil {
		t.Fatalf("create second team: %v", err)
	}

	task := createHTTPTask(t, router, "Published business task")
	publishHTTPTask(t, router, task.ID)
	proposalBody := func(teamID int64, idea string) string {
		return `{"team_id":` + strconv.FormatInt(teamID, 10) + `,"idea":"` + idea + `","plan":"Analyze data and build a prototype","deadline":"2026-10-15","prototype_url":"https://example.com/prototype"}`
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/proposals", bytes.NewBufferString(proposalBody(teamOne.ID, "Build a lightweight inventory dashboard")))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected proposal creation status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var first model.Proposal
	if err := json.NewDecoder(createRecorder.Body).Decode(&first); err != nil {
		t.Fatalf("decode created proposal: %v", err)
	}
	if first.ID <= 0 || first.TaskID != task.ID || first.TeamID != teamOne.ID || first.Status != "pending" {
		t.Fatalf("unexpected created proposal: %+v", first)
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/proposals", bytes.NewBufferString(proposalBody(teamTwo.ID, "Build a reporting prototype")))
	secondRecorder := httptest.NewRecorder()
	router.ServeHTTP(secondRecorder, secondRequest)
	if secondRecorder.Code != http.StatusCreated {
		t.Fatalf("expected second proposal creation status %d, got %d", http.StatusCreated, secondRecorder.Code)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/proposals", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected proposal list status %d, got %d", http.StatusOK, listRecorder.Code)
	}

	var proposals []model.Proposal
	if err := json.NewDecoder(listRecorder.Body).Decode(&proposals); err != nil {
		t.Fatalf("decode proposal list: %v", err)
	}
	if len(proposals) != 2 || proposals[0].ID <= proposals[1].ID {
		t.Fatalf("expected two newest-first proposals, got %+v", proposals)
	}

	emptyTask := createHTTPTask(t, router, "Published task with no proposals")
	publishHTTPTask(t, router, emptyTask.ID)
	emptyRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(emptyTask.ID, 10)+"/proposals", nil)
	emptyRecorder := httptest.NewRecorder()
	router.ServeHTTP(emptyRecorder, emptyRequest)
	if emptyRecorder.Code != http.StatusOK {
		t.Fatalf("expected empty proposal list status %d, got %d", http.StatusOK, emptyRecorder.Code)
	}
	var empty []model.Proposal
	if err := json.NewDecoder(emptyRecorder.Body).Decode(&empty); err != nil {
		t.Fatalf("decode empty proposal list: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected empty proposal array, got %+v", empty)
	}
}

func TestProposalCreationRules(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	team := &model.Team{Name: "Proposal Team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	unpublished := createHTTPTask(t, router, "Unpublished task")
	validBody := `{"team_id":` + strconv.FormatInt(team.ID, 10) + `,"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`
	unpublishedRequest := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(unpublished.ID, 10)+"/proposals", bytes.NewBufferString(validBody))
	unpublishedRecorder := httptest.NewRecorder()
	router.ServeHTTP(unpublishedRecorder, unpublishedRequest)
	if unpublishedRecorder.Code != http.StatusConflict {
		t.Fatalf("expected unpublished task status %d, got %d", http.StatusConflict, unpublishedRecorder.Code)
	}

	published := createHTTPTask(t, router, "Published task")
	publishHTTPTask(t, router, published.ID)
	nonexistentTeamBody := `{"team_id":999,"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`
	nonexistentTeamRequest := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(published.ID, 10)+"/proposals", bytes.NewBufferString(nonexistentTeamBody))
	nonexistentTeamRecorder := httptest.NewRecorder()
	router.ServeHTTP(nonexistentTeamRecorder, nonexistentTeamRequest)
	if nonexistentTeamRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected nonexistent team status %d, got %d", http.StatusNotFound, nonexistentTeamRecorder.Code)
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "missing team", body: `{"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`},
		{name: "empty idea", body: `{"team_id":1,"idea":"   ","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`},
		{name: "empty plan", body: `{"team_id":1,"idea":"Idea","plan":"   ","deadline":"2026-10-15","prototype_url":"https://example.com"}`},
		{name: "empty deadline", body: `{"team_id":1,"idea":"Idea","plan":"Plan","deadline":"   ","prototype_url":"https://example.com"}`},
		{name: "empty prototype URL", body: `{"team_id":1,"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"   "}`},
		{name: "unknown status", body: `{"team_id":1,"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com","status":"accepted"}`},
		{name: "malformed JSON", body: `{"team_id":1,"idea":`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(published.ID, 10)+"/proposals", bytes.NewBufferString(test.body))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestProposalEndpointErrors(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	team := &model.Team{Name: "Proposal Team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		expectCode int
	}{
		{name: "invalid task ID on create", method: http.MethodPost, path: "/api/tasks/invalid/proposals", expectCode: http.StatusBadRequest},
		{name: "missing task on create", method: http.MethodPost, path: "/api/tasks/999/proposals", expectCode: http.StatusNotFound},
		{name: "invalid task ID on list", method: http.MethodGet, path: "/api/tasks/0/proposals", expectCode: http.StatusBadRequest},
		{name: "missing task on list", method: http.MethodGet, path: "/api/tasks/999/proposals", expectCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(`{"team_id":`+strconv.FormatInt(team.ID, 10)+`,"idea":"Idea","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.expectCode {
				t.Fatalf("expected status %d, got %d", test.expectCode, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}
