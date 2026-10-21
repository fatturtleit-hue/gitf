package handlers

// APIError represents a structured API error response. The frontend reads
// {code, message} verbatim and shows `message` to the user.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Predefined error codes — the contract between backend and frontend.
// Renaming any of these is a breaking change.
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeRateLimited  = "RATE_LIMITED"
	ErrCodeInternal     = "INTERNAL_ERROR"
)
