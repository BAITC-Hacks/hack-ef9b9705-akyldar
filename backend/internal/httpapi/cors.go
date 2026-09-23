package httpapi

import (
	"net/http"
	"strings"
)

const DefaultFrontendOrigin = "http://localhost:5173"

const allowedCORSMethods = "GET, POST, PUT, PATCH, OPTIONS"

// WithCORS adds the small CORS policy needed by the separate frontend.
func WithCORS(next http.Handler, allowedOrigin string) http.Handler {
	allowedOrigin = strings.TrimSpace(allowedOrigin)
	if allowedOrigin == "" {
		allowedOrigin = DefaultFrontendOrigin
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originAllowed := r.Header.Get("Origin") == allowedOrigin
		if originAllowed {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", allowedCORSMethods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Expose-Headers", "X-AI-Mode")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
