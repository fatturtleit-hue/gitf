package middleware

import (
	"encoding/json"
	"log"
	"net/http"
)

// Error codes mirror handlers package constants to avoid import cycles.
const (
	errCodeUnauthorized = "UNAUTHORIZED"
	errCodeInternal     = "INTERNAL_ERROR"
)

// APIError represents a structured API error response.
// This is a copy of handlers.APIError but in the middleware package to avoid import cycles.
// Field names and codes must match handlers.APIError exactly.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// writeAPIError writes a structured error response using the same format as handlers.
// This function exists in the middleware package to avoid import cycles.
func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(APIError{Code: code, Message: message}); err != nil {
		// Log encoding errors but don't panic — response headers already sent.
		// This is best-effort; the client might get a truncated response.
	}
}

// writeUnauthorized writes a 401 response with the standard error format.
// This function now uses the shared APIError type to maintain consistency.
func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="gitf"`)
	writeAPIError(w, r, http.StatusUnauthorized, errCodeUnauthorized, "unauthorized")
}

// writeServerError writes a 500 response with the standard error format.
func writeServerError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	// Log the error for debugging
	log.Printf("Server error during %s: %v", operation, err)

	// Don't expose the actual error to the client
	writeAPIError(w, r, http.StatusInternalServerError, errCodeInternal, "internal server error")
}

// writeJSON writes a JSON response with the given status code and data.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log encoding errors but don't panic — response headers already sent.
		// This is best-effort; the client might get a truncated response.
		log.Printf("Error encoding JSON response: %v", err)
	}
}
