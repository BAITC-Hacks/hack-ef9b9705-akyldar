package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	ai "hackalem/ai"

	"backend/internal/database"
	"backend/internal/httpapi"
	"backend/internal/model"
	"backend/internal/repository"
)

func newCombinedTestHandlerAndRepositories(t *testing.T) (http.Handler, *repository.TeamRepository) {
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
	return httpapi.WithCORS(combined, "http://frontend.example"), repository.NewTeamRepository(db)
}

func newCombinedTestHandler(t *testing.T) http.Handler {
	handler, _ := newCombinedTestHandlerAndRepositories(t)
	return handler
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

func TestCompleteMVPFlow(t *testing.T) {
	handler, teamRepository := newCombinedTestHandlerAndRepositories(t)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := server.Client()

	response, body := requestJSON(t, client, http.MethodPost, server.URL+"/api/tasks", `{"initial_description":"Хотим автоматизировать работу склада"}`)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create task: expected %d, got %d: %s", http.StatusCreated, response.StatusCode, body)
	}
	var task model.Task
	decodeBody(t, body, &task)
	if task.Rating != 0 || task.ReadinessLevel != "draft" || task.Published {
		t.Fatalf("unexpected initial task state: %+v", task)
	}

	response, body = requestJSON(t, client, http.MethodPost, server.URL+"/api/ai/questions", `{"description":"Хотим автоматизировать работу склада"}`)
	if response.StatusCode != http.StatusOK || response.Header.Get("X-AI-Mode") != "fallback" {
		t.Fatalf("AI questions: expected 200/fallback, got %d/%q: %s", response.StatusCode, response.Header.Get("X-AI-Mode"), body)
	}
	var questions ai.QuestionsResponse
	decodeBody(t, body, &questions)
	if len(questions.Questions) < 3 {
		t.Fatalf("expected at least 3 AI questions, got %d", len(questions.Questions))
	}

	cardRequest := ai.CardRequest{
		Description: task.InitialDescription,
		Answers: []ai.Answer{
			{Question: questions.Questions[0], Answer: "Кладовщики и руководитель склада."},
			{Question: questions.Questions[1], Answer: "Excel с остатками и история продаж."},
			{Question: questions.Questions[2], Answer: "Веб-интерфейс с уведомлениями о дефиците."},
		},
	}
	cardBody, err := json.Marshal(cardRequest)
	if err != nil {
		t.Fatalf("marshal card request: %v", err)
	}
	response, body = requestJSON(t, client, http.MethodPost, server.URL+"/api/ai/card", string(cardBody))
	if response.StatusCode != http.StatusOK || response.Header.Get("X-AI-Mode") != "fallback" {
		t.Fatalf("AI card: expected 200/fallback, got %d/%q: %s", response.StatusCode, response.Header.Get("X-AI-Mode"), body)
	}
	var card ai.TaskCard
	decodeBody(t, body, &card)
	if card.Context != task.InitialDescription || card.Title != "" || card.Need != "" {
		t.Fatalf("unexpected fallback card: %+v", card)
	}

	taskPath := server.URL + "/api/tasks/" + strconv.FormatInt(task.ID, 10)
	cardBody, err = json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal generated card: %v", err)
	}
	response, body = requestJSON(t, client, http.MethodPut, taskPath, string(cardBody))
	if response.StatusCode != http.StatusOK {
		t.Fatalf("save generated card: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	decodeBody(t, body, &task)
	if task.Rating != 10 || task.ReadinessLevel != "draft" {
		t.Fatalf("expected first real rating 10/draft, got %d/%q", task.Rating, task.ReadinessLevel)
	}

	completeCard := ai.TaskCard{
		Title:             "Автоматизация работы склада",
		Context:           "Сейчас остатки проверяются вручную на складе.",
		Need:              "Сократить время проверки остатков и дефицита.",
		Users:             "Кладовщики и руководитель склада.",
		Data:              "Excel с остатками и история продаж.",
		Constraints:       "Использовать обезличенные данные и браузерный прототип.",
		ExpectedResult:    "Веб-интерфейс с уведомлениями о дефиците.",
		SuccessCriteria:   "Сократить время проверки остатков на 30%.",
		Contact:           "warehouse@example.com",
		InteractionFormat: "Еженедельная онлайн-встреча.",
		Topic:             "logistics",
	}
	completeBody, err := json.Marshal(completeCard)
	if err != nil {
		t.Fatalf("marshal completed card: %v", err)
	}
	response, body = requestJSON(t, client, http.MethodPut, taskPath, string(completeBody))
	if response.StatusCode != http.StatusOK {
		t.Fatalf("save completed card: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	decodeBody(t, body, &task)
	if task.Rating != 100 || task.ReadinessLevel != "priority" {
		t.Fatalf("expected second real rating 100/priority, got %d/%q", task.Rating, task.ReadinessLevel)
	}

	response, body = requestJSON(t, client, http.MethodGet, taskPath+"/rating", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get rating: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	var ratingResult struct {
		Score int    `json:"score"`
		Level string `json:"level"`
	}
	decodeBody(t, body, &ratingResult)
	if ratingResult.Score != 100 || ratingResult.Level != "priority" {
		t.Fatalf("unexpected backend rating: %+v", ratingResult)
	}

	response, body = requestJSON(t, client, http.MethodPost, taskPath+"/confirm", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("confirm task: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	decodeBody(t, body, &task)
	if !task.Confirmed {
		t.Fatal("expected task to be confirmed")
	}

	response, body = requestJSON(t, client, http.MethodPost, taskPath+"/publish", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("publish task: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	decodeBody(t, body, &task)
	if !task.Published {
		t.Fatal("expected task to be published")
	}

	response, body = requestJSON(t, client, http.MethodGet, server.URL+"/api/tasks", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("catalog: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	var catalog []model.Task
	decodeBody(t, body, &catalog)
	if !containsTask(catalog, task.ID) {
		t.Fatalf("published task %d was not visible in catalog", task.ID)
	}

	team := &model.Team{Name: "Integration team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create integration team: %v", err)
	}
	proposalBody := `{"team_id":` + strconv.FormatInt(team.ID, 10) + `,"idea":"Панель контроля склада","plan":"Собрать прототип и проверить его с кладовщиками","deadline":"2026-12-01","prototype_url":"https://example.com/warehouse"}`
	response, body = requestJSON(t, client, http.MethodPost, taskPath+"/proposals", proposalBody)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create proposal: expected %d, got %d: %s", http.StatusCreated, response.StatusCode, body)
	}
	var proposal model.Proposal
	decodeBody(t, body, &proposal)
	if proposal.Status != "pending" {
		t.Fatalf("expected pending proposal, got %q", proposal.Status)
	}

	response, body = requestJSON(t, client, http.MethodPatch, server.URL+"/api/proposals/"+strconv.FormatInt(proposal.ID, 10)+"/status", `{"status":"accepted"}`)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("accept proposal: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	decodeBody(t, body, &proposal)
	if proposal.Status != "accepted" {
		t.Fatalf("expected accepted proposal, got %q", proposal.Status)
	}

	response, body = requestJSON(t, client, http.MethodGet, taskPath+"/proposals", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list proposals: expected %d, got %d: %s", http.StatusOK, response.StatusCode, body)
	}
	var proposals []model.Proposal
	decodeBody(t, body, &proposals)
	if len(proposals) != 1 || proposals[0].Status != "accepted" {
		t.Fatalf("expected persisted accepted proposal, got %+v", proposals)
	}
}

func requestJSON(t *testing.T, client *http.Client, method, url, body string) (*http.Response, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create %s request: %v", method, err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send %s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", method, err)
	}
	return response, data
}

func decodeBody(t *testing.T, body []byte, destination any) {
	t.Helper()
	if err := json.Unmarshal(body, destination); err != nil {
		t.Fatalf("decode response body %q: %v", body, err)
	}
}

func containsTask(tasks []model.Task, id int64) bool {
	for _, task := range tasks {
		if task.ID == id {
			return true
		}
	}
	return false
}
