package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/models"

	"gorm.io/gorm"
)

// minPasswordLength is the minimum acceptable password length on registration.
// Six is too short for serious deployments — operators should add a complexity
// check via UserStore in front of this if their threat model demands it — but
// it matches what the frontend validator promises today.
const minPasswordLength = 8

// AuthHandler holds authentication HTTP handlers. Depends on UserStore and AuditStore (not
// *gorm.DB) so it can be unit-tested without a real database.
type AuthHandler struct {
	Users  UserStore
	Audit  models.AuditStore
	Config *config.Config
}

// AuthResponse returns the user info and JWT.
type AuthResponse struct {
	Token string       `json:"token"`
	User  models.UserDTO `json:"user"`
}

// NewAuthHandler builds an AuthHandler backed by the GORM-based user store.
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		Users:  NewGormUserStore(db),
		Audit:  models.NewGormAuditStore(db),
		Config: cfg,
	}
}

// RegisterRequest is the body for user registration.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the body for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new user account. When self-registration is disabled
// (DISABLE_REGISTRATION=true), this endpoint returns 403.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.Config.DisableRegistration {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "registration is disabled")
		return
	}
	var req RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if err := validateRegister(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeServerError(w, r, "hash password", err)
		return
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
	}
	if err := h.Users.Create(r.Context(), &user); err != nil {
		if errors.Is(err, ErrUserConflict) {
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "username or email already taken")
			return
		}
		writeServerError(w, r, "create user", err)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, h.Config.JWTSecret)
	if err != nil {
		writeServerError(w, r, "generate token", err)
		return
	}

	_ = h.Audit.Log(user.ID, "register", "user")

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, r, http.StatusCreated, AuthResponse{Token: token, User: user.DTO()})
}

// Login authenticates a user and returns a JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := h.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		auth.ConstantTimeDelay()
		if errors.Is(err, ErrUserNotFound) {
			writeAPIError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid credentials")
			return
		}
		writeServerError(w, r, "lookup user", err)
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		auth.ConstantTimeDelay()
		writeAPIError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, h.Config.JWTSecret)
	if err != nil {
		writeServerError(w, r, "generate token", err)
		return
	}

	_ = h.Audit.Log(user.ID, "login", "user")

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, r, http.StatusOK, AuthResponse{Token: token, User: user.DTO()})
}

// Me returns the currently authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, r, http.StatusOK, user.DTO())
}

// RefreshToken refreshes an expired JWT token using a valid refresh token.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	// Generate a new token
	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, h.Config.JWTSecret)
	if err != nil {
		writeServerError(w, r, "generate token", err)
		return
	}

	// Set the new token as an httpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Log the refresh
	_ = h.Audit.Log(user.ID, "refresh", "user")

	// Return success response
	writeJSON(w, r, http.StatusOK, map[string]string{
		"message": "Token refreshed successfully",
	})
}

// RegistrationStatus reports whether public self-registration is enabled.
func (h *AuthHandler) RegistrationStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]bool{
		"enabled": !h.Config.DisableRegistration,
	})
}

// SetupDefaultAdminRequest is the body for the one-time default admin setup.
type SetupDefaultAdminRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SetupDefaultAdmin allows the bootstrap admin to change credentials once.
// After SetupCompleted is set, this endpoint rejects further requests.
func (h *AuthHandler) SetupDefaultAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	if !user.IsDefaultAdmin {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "only the default admin can use this endpoint")
		return
	}

	if user.SetupCompleted {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "setup has already been completed")
		return
	}

	var req SetupDefaultAdminRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "username, email and password are required")
		return
	}
	if len(req.Username) > 50 {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "username must not exceed 50 characters")
		return
	}
	if !validUsername(req.Username) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "username may contain only letters, digits, underscores and hyphens")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid email address")
		return
	}
	if len(req.Password) < minPasswordLength {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeServerError(w, r, "hash password", err)
		return
	}

	user.Username = req.Username
	user.Email = req.Email
	user.PasswordHash = hash
	user.SetupCompleted = true

	if err := h.Users.Update(r.Context(), user); err != nil {
		writeServerError(w, r, "update user", err)
		return
	}

	// Generate a new JWT since the username changed.
	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, h.Config.JWTSecret)
	if err != nil {
		writeServerError(w, r, "generate token", err)
		return
	}

	_ = h.Audit.Log(user.ID, "default_admin_setup", "user")

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, r, http.StatusOK, AuthResponse{Token: token, User: user.DTO()})
}

// validateRegister applies all server-side rules. The frontend mirrors these
// for UX, but the backend never trusts the frontend.
func validateRegister(req *RegisterRequest) error {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return errors.New("username, email and password are required")
	}
	if len(req.Username) > 50 {
		return errors.New("username must not exceed 50 characters")
	}
	// Reuse the same identifier rule the rest of the system uses for owner names.
	if !validUsername(req.Username) {
		return errors.New("username may contain only letters, digits, underscores and hyphens")
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.New("invalid email address")
	}
	if len(req.Password) < minPasswordLength {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}