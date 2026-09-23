package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// RegisterRoutes integrates with a standard Go router without owning the server.
// Authentication, rate limiting, CORS, persistence and rating belong to the host.
func RegisterRoutes(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("/api/ai/questions", func(w http.ResponseWriter, r *http.Request) {
		raw, ok := readRequest(w, r)
		if !ok {
			return
		}
		req, err := ParseQuestionsRequest(raw)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		result, mode, err := service.Questions(r.Context(), req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		w.Header().Set("X-AI-Mode", string(mode))
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/ai/card", func(w http.ResponseWriter, r *http.Request) {
		raw, ok := readRequest(w, r)
		if !ok {
			return
		}
		req, err := ParseCardRequest(raw)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		result, mode, err := service.Card(r.Context(), req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		w.Header().Set("X-AI-Mode", string(mode))
		writeJSON(w, http.StatusOK, result)
	})
}

func NewHandler(service *Service) http.Handler {
	mux := http.NewServeMux()
	RegisterRoutes(mux, service)
	return mux
}

func readRequest(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST for this endpoint.")
		return nil, false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.")
		return nil, false
	}
	if r.Body == nil {
		writeServiceError(w, invalidInput())
		return nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			writeServiceError(w, inputLimit())
		} else {
			writeServiceError(w, invalidInput())
		}
		return nil, false
	}
	return raw, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	var inputErr *InputError
	switch {
	case errors.As(err, &inputErr):
		status := http.StatusBadRequest
		if inputErr.TooLarge {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, inputErr.Code, inputErr.Message)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// The disconnected caller might never receive this response; no fallback
		// is generated and no more provider calls are made after cancellation.
		writeError(w, http.StatusRequestTimeout, "request_cancelled", "The request was cancelled or its deadline expired.")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "The request could not be processed.")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{Error: struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
