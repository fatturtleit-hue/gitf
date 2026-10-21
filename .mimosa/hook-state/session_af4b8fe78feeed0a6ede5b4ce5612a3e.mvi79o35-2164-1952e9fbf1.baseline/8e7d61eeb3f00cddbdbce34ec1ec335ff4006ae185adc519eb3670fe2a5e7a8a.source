package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"gitf/internal/middleware"
	"gitf/internal/models"
)

// MaxBodyBytes caps the size of any JSON request body the API accepts.
// 1 MiB is plenty for register/login/repo-create payloads.
const MaxBodyBytes = 1 << 20

// writeJSON writes a JSON response and logs encoding errors with the request id.
// Encoding errors are non-fatal — the status header has already been sent — but
// they MUST be logged so that broken-pipe / disconnect issues are observable.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil && !isClientGone(err) {
		slog.WarnContext(r.Context(), "json encode failed",
			"error", err,
			"path", r.URL.Path,
			"request_id", middleware.GetRequestID(r.Context()),
		)
	}
}

// writeAPIError writes a structured error response. All error responses in this
// service flow through this single function so the wire format cannot drift.
func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, r, status, APIError{Code: code, Message: message})
}

// writeServerError logs an internal error with full context and returns a
// generic 500 to the client (no internal details leaked).
func writeServerError(w http.ResponseWriter, r *http.Request, op string, err error) {
	slog.ErrorContext(r.Context(), op,
		"error", err,
		"path", r.URL.Path,
		"method", r.Method,
		"request_id", middleware.GetRequestID(r.Context()),
	)
	writeAPIError(w, r, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
}

// requireUser returns the authenticated user or writes a 401 and reports false.
// Replaces the four-line UserFromContext+401 boilerplate that was duplicated in
// every protected handler.
func requireUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeAPIError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "unauthorized")
		return nil, false
	}
	return user, true
}

// decodeJSON decodes a JSON request body into v with a hard byte cap and
// strict field handling. On error it has already written a response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeAPIError(w, r, http.StatusRequestEntityTooLarge, ErrCodeValidation, "request body too large")
			return false
		}
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid request body")
		return false
	}
	return true
}

// isClientGone reports whether err looks like a client disconnect rather than
// a server-side bug worth alerting on.
func isClientGone(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, io.EOF) {
		return true
	}
	return false
}

// atoiOrZero parses an integer or returns zero. Used for query params where
// an unparseable value semantically means "not provided".
func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// isPathInside reports whether target is the same as or inside base after both
// paths are cleaned. Used to guard against path-traversal attacks on
// user-supplied filenames (e.g. release asset uploads).
func isPathInside(target, base string) bool {
	t := filepath.Clean(target)
	b := filepath.Clean(base)
	if t == b {
		return true
	}
	return strings.HasPrefix(t, b+string(filepath.Separator))
}
