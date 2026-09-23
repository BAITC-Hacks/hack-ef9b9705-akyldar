package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSConfiguredOrigin(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "http://frontend.example")

	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set("Origin", "http://frontend.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://frontend.example" {
		t.Fatalf("expected configured origin, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if exposed := recorder.Header().Get("Access-Control-Expose-Headers"); exposed != "X-AI-Mode" {
		t.Fatalf("expected X-AI-Mode to be exposed, got %q", exposed)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not reach the wrapped handler")
	}), "http://frontend.example")

	request := httptest.NewRequest(http.MethodOptions, "/api/tasks", nil)
	request.Header.Set("Origin", "http://frontend.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if methods := recorder.Header().Get("Access-Control-Allow-Methods"); methods != allowedCORSMethods {
		t.Fatalf("expected allowed methods %q, got %q", allowedCORSMethods, methods)
	}
	if headers := recorder.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type" {
		t.Fatalf("expected allowed headers Content-Type, got %q", headers)
	}
}

func TestCORSRejectsUnrelatedOrigin(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "http://frontend.example")

	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set("Origin", "http://unrelated.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("unrelated origin unexpectedly received CORS access: %q", origin)
	}
}
