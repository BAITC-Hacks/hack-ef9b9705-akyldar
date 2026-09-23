package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerContract(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ForceFallback = true
	s, err := NewService(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(s)
	cases := []struct {
		name, path, method, contentType, body string
		status                                int
		mode                                  Mode
	}{
		{"questions", "/api/ai/questions", "POST", "application/json", `{"description":"Хотим автоматизировать склад"}`, 200, ModeFallback},
		{"card", "/api/ai/card", "POST", "application/json; charset=utf-8", `{"description":"Қойма керек","answers":[]}`, 200, ModeFallback},
		{"empty", "/api/ai/questions", "POST", "application/json", `{"description":" "}`, 400, ""},
		{"malformed", "/api/ai/questions", "POST", "application/json", `{"description":`, 400, ""},
		{"missing_answers", "/api/ai/card", "POST", "application/json", `{"description":"a"}`, 400, ""},
		{"null_answer", "/api/ai/card", "POST", "application/json", `{"description":"a","answers":[{"question":"q","answer":null}]}`, 400, ""},
		{"duplicate", "/api/ai/questions", "POST", "application/json", `{"description":"a","description":"b"}`, 400, ""},
		{"extra", "/api/ai/questions", "POST", "application/json", `{"description":"a","score":100}`, 400, ""},
		{"method", "/api/ai/questions", "GET", "application/json", `{}`, 405, ""},
		{"content_type", "/api/ai/questions", "POST", "text/plain", `{"description":"a"}`, 415, ""},
		{"missing_type", "/api/ai/questions", "POST", "", `{"description":"a"}`, 415, ""},
		{"too_long_text", "/api/ai/questions", "POST", "application/json", `{"description":"` + strings.Repeat("x", MaxDescriptionRunes+1) + `"}`, 413, ""},
		{"too_long_body", "/api/ai/questions", "POST", "application/json", strings.Repeat(" ", MaxRequestBytes+1), 413, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tc.status || recorder.Header().Get("X-AI-Mode") != string(tc.mode) {
				t.Fatalf("status=%d mode=%s body=%s", recorder.Code, recorder.Header().Get("X-AI-Mode"), recorder.Body.String())
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store")
			}
			if !json.Valid(recorder.Body.Bytes()) {
				t.Fatal("response is not JSON")
			}
			if tc.status == 405 && recorder.Header().Get("Allow") != "POST" {
				t.Fatal("missing Allow")
			}
			if tc.status >= 400 {
				var envelope map[string]json.RawMessage
				_ = json.Unmarshal(recorder.Body.Bytes(), &envelope)
				if len(envelope) != 1 || envelope["error"] == nil {
					t.Fatal("invalid error contract")
				}
			} else if tc.path == "/api/ai/card" {
				card, err := ValidateCard(recorder.Body.Bytes())
				if err != nil || card.Context != "Қойма керек" || card.Title != "" {
					t.Fatalf("%+v %v", card, err)
				}
			} else if _, err := ValidateQuestions(recorder.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHandlerLiveAndCancelled(t *testing.T) {
	calls := 0
	s := testService(t, fakeProvider(func(context.Context, GenerationRequest) ([]byte, error) {
		calls++
		return []byte(validQuestionsJSON), nil
	}))
	handler := NewHandler(s)
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelled {
			cancel()
		}
		req := httptest.NewRequest(http.MethodPost, "/api/ai/questions", strings.NewReader(`{"description":"Склад"}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		cancel()
		if cancelled {
			if recorder.Code != 408 || recorder.Header().Get("X-AI-Mode") != "" {
				t.Fatal("cancelled request returned successful fallback")
			}
		} else if recorder.Code != 200 || recorder.Header().Get("X-AI-Mode") != "live" {
			t.Fatal("fake live contract failed")
		}
	}
	if calls != 1 {
		t.Fatalf("provider calls=%d", calls)
	}
}

func TestInvalidInputDoesNotCallProvider(t *testing.T) {
	s := testService(t, fakeProvider(func(context.Context, GenerationRequest) ([]byte, error) {
		t.Fatal("provider called for invalid input")
		return nil, nil
	}))
	if _, _, err := s.Questions(context.Background(), QuestionsRequest{}); err == nil {
		t.Fatal("empty description accepted")
	}
	if _, _, err := s.Card(context.Background(), CardRequest{Description: "a", Answers: nil}); err == nil {
		t.Fatal("nil answers accepted")
	}
}
