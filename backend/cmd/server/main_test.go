package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	ai "hackalem/ai"

	"backend/internal/database"
	"backend/internal/httpapi"
	"backend/internal/model"
	"backend/internal/repository"
)

func newCombinedTestHandler(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "integration.db"))
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize integration schema: %v", err)
	}

	aiConfig := ai.DefaultConfig()
	aiConfig.ForceFallback = true
	aiService, err := ai.NewService(aiConfig, nil)
	if err != nil {
		t.Fatalf("initialize test AI service: %v", err)
	}

	combined := http.NewServeMux()
	ai.RegisterRoutes(combined, aiService)
	combined.Handle("/", httpapi.NewRouter(
		repository.NewTaskRepository(db),
		repository.NewTeamRepository(db),
		repository.NewProposalRepository(db),
	))
	return httpapi.WithCORS(combined, "http://frontend.example")
}

func TestCombinedHandlerRoutesAIAndBackend(t *testing.T) {
	handler := newCombinedTestHandler(t)

	healthRequest := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	healthRecorder := httptest.NewRecorder()
	handler.ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("expected health status %d, got %d", http.StatusOK, healthRecorder.Code)
	}

	questionsRequest := httptest.NewRequest(http.MethodPost, "/api/ai/questions", bytes.NewBufferString(`{"description":"Improve warehouse handoffs"}`))
	questionsRequest.Header.Set("Content-Type", "application/json")
	questionsRequest.Header.Set("Origin", "http://frontend.example")
	questionsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(questionsRecorder, questionsRequest)
	if questionsRecorder.Code != http.StatusOK {
		t.Fatalf("expected AI questions status %d, got %d: %s", http.StatusOK, questionsRecorder.Code, questionsRecorder.Body.String())
	}
	if questionsRecorder.Header().Get("X-AI-Mode") != "fallback" {
		t.Fatalf("expected fallback AI mode, got %q", questionsRecorder.Header().Get("X-AI-Mode"))
	}
	if questionsRecorder.Header().Get("Access-Control-Allow-Origin") != "http://frontend.example" {
		t.Fatalf("expected configured CORS origin, got %q", questionsRecorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if questionsRecorder.Header().Get("Access-Control-Expose-Headers") != "X-AI-Mode" {
		t.Fatalf("expected exposed AI mode header, got %q", questionsRecorder.Header().Get("Access-Control-Expose-Headers"))
	}
	var questions ai.QuestionsResponse
	if err := json.NewDecoder(questionsRecorder.Body).Decode(&questions); err != nil {
		t.Fatalf("decode AI questions response: %v", err)
	}
	if len(questions.Questions) < 3 {
		t.Fatalf("expected fallback questions, got %+v", questions)
	}

	cardRequest := httptest.NewRequest(http.MethodPost, "/api/ai/card", bytes.NewBufferString(`{"description":"Improve warehouse handoffs","answers":[]}`))
	cardRequest.Header.Set("Content-Type", "application/json")
	cardRecorder := httptest.NewRecorder()
	handler.ServeHTTP(cardRecorder, cardRequest)
	if cardRecorder.Code != http.StatusOK {
		t.Fatalf("expected AI card status %d, got %d: %s", http.StatusOK, cardRecorder.Code, cardRecorder.Body.String())
	}
	if cardRecorder.Header().Get("X-AI-Mode") != "fallback" {
		t.Fatalf("expected fallback AI mode for card, got %q", cardRecorder.Header().Get("X-AI-Mode"))
	}
	var card ai.TaskCard
	if err := json.NewDecoder(cardRecorder.Body).Decode(&card); err != nil {
		t.Fatalf("decode AI card response: %v", err)
	}
	if card.Context != "Improve warehouse handoffs" {
		t.Fatalf("unexpected fallback card: %+v", card)
	}

	taskRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Backend task"}`))
	taskRecorder := httptest.NewRecorder()
	handler.ServeHTTP(taskRecorder, taskRequest)
	if taskRecorder.Code != http.StatusCreated {
		t.Fatalf("expected backend task status %d, got %d", http.StatusCreated, taskRecorder.Code)
	}
	var task model.Task
	if err := json.NewDecoder(taskRecorder.Body).Decode(&task); err != nil {
		t.Fatalf("decode backend task: %v", err)
	}
	if task.ID <= 0 {
		t.Fatalf("expected backend task ID, got %d", task.ID)
	}

	optionsRequest := httptest.NewRequest(http.MethodOptions, "/api/ai/questions", nil)
	optionsRequest.Header.Set("Origin", "http://frontend.example")
	optionsRequest.Header.Set("Access-Control-Request-Method", http.MethodPost)
	optionsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(optionsRecorder, optionsRequest)
	if optionsRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected AI preflight status %d, got %d", http.StatusNoContent, optionsRecorder.Code)
	}
	if optionsRecorder.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT, PATCH, OPTIONS" {
		t.Fatalf("unexpected preflight methods: %q", optionsRecorder.Header().Get("Access-Control-Allow-Methods"))
	}
}
