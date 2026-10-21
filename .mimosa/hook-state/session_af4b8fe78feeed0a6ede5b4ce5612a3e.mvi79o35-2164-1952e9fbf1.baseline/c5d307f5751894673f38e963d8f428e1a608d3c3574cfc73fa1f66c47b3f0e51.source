package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/models"

	"gorm.io/gorm"
)

// UserHandler provides admin-only user management endpoints.
type UserHandler struct {
	Users  UserStore
	Audit  models.AuditStore
	Config *config.Config
}

// NewUserHandler creates a user management handler.
func NewUserHandler(db *gorm.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{
		Users:  NewGormUserStore(db),
		Audit:  models.NewGormAuditStore(db),
		Config: cfg,
	}
}

// ListUsersResponse is the API shape for GET /api/v1/admin/users.
type ListUsersResponse struct {
	Items      []models.UserDTO `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

// CreateUserRequest is the body for POST /api/v1/admin/users.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// UpdateUserRequest is the body for PATCH /api/v1/admin/users/{id}.
// All fields are optional; only provided fields are applied.
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
}

// ListUsers returns a paginated, searchable list of all users.
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := atoiOrZero(r.URL.Query().Get("page"))
	perPage := atoiOrZero(r.URL.Query().Get("per_page"))
	page, perPage = clampPage(page, perPage)
	offset := (page - 1) * perPage

	users, err := h.Users.List(r.Context(), search, perPage, offset)
	if err != nil {
		writeServerError(w, r, "list users", err)
		return
	}

	total, err := h.Users.Count(r.Context())
	if err != nil {
		writeServerError(w, r, "count users", err)
		return
	}

	dtos := make([]models.UserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, u.DTO())
	}

	writeJSON(w, r, http.StatusOK, ListUsersResponse{
		Items:      dtos,
		Pagination: NewPagination(page, perPage, int(total)),
	})
}

// GetUser returns a single user by ID.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user id")
		return
	}

	user, err := h.Users.ByID(r.Context(), uint(id))
	if err != nil {
		if err == ErrUserNotFound {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "user not found")
			return
		}
		writeServerError(w, r, "lookup user", err)
		return
	}

	writeJSON(w, r, http.StatusOK, user.DTO())
}

// CreateUser creates a new user (admin only).
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := requireUser(w, r)
	if !ok {
		return
	}

	var req CreateUserRequest
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
	if len(req.Password) < minPasswordLength {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "password must be at least 8 characters")
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
		IsSiteAdmin:  req.IsAdmin,
	}
	if err := h.Users.Create(r.Context(), &user); err != nil {
		writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "username or email already taken")
		return
	}

	_ = h.Audit.Log(currentUser.ID, "admin_create_user", "user")
	writeJSON(w, r, http.StatusCreated, user.DTO())
}

// UpdateUser partially updates a user (admin only).
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := requireUser(w, r)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user id")
		return
	}

	target, err := h.Users.ByID(r.Context(), uint(id))
	if err != nil {
		if err == ErrUserNotFound {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "user not found")
			return
		}
		writeServerError(w, r, "lookup user", err)
		return
	}

	var req UpdateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	changed := false

	if req.Username != nil {
		v := strings.TrimSpace(*req.Username)
		if v == "" || len(v) > 50 || !validUsername(v) {
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid username")
			return
		}
		target.Username = v
		changed = true
	}

	if req.Email != nil {
		v := strings.TrimSpace(*req.Email)
		if v == "" {
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "email cannot be empty")
			return
		}
		target.Email = v
		changed = true
	}

	if req.Password != nil {
		if len(*req.Password) < minPasswordLength {
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "password must be at least 8 characters")
			return
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			writeServerError(w, r, "hash password", err)
			return
		}
		target.PasswordHash = hash
		changed = true
	}

	if req.IsAdmin != nil {
		target.IsSiteAdmin = *req.IsAdmin
		changed = true
	}

	if !changed {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "no fields to update")
		return
	}

	if err := h.Users.Update(r.Context(), target); err != nil {
		writeServerError(w, r, "update user", err)
		return
	}

	_ = h.Audit.Log(currentUser.ID, "admin_update_user", "user")
	writeJSON(w, r, http.StatusOK, target.DTO())
}

// DeleteUser deletes a user (admin only). Cannot delete the bootstrap admin or
// the currently logged-in user.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := requireUser(w, r)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user id")
		return
	}

	if currentUser.ID == uint(id) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "cannot delete yourself")
		return
	}

	target, err := h.Users.ByID(r.Context(), uint(id))
	if err != nil {
		if err == ErrUserNotFound {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "user not found")
			return
		}
		writeServerError(w, r, "lookup user", err)
		return
	}

	if target.IsDefaultAdmin {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "cannot delete the default admin account")
		return
	}

	// Prevent deleting the last site admin.
	if target.IsSiteAdmin {
		admins, err := h.listAdminUsers(r.Context())
		if err != nil {
			writeServerError(w, r, "count admins", err)
			return
		}
		if len(admins) <= 1 {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "cannot remove the last site administrator")
			return
		}
	}

	if err := h.Users.Delete(r.Context(), uint(id)); err != nil {
		writeServerError(w, r, "delete user", err)
		return
	}

	_ = h.Audit.Log(currentUser.ID, "admin_delete_user", "user")
	w.WriteHeader(http.StatusNoContent)
}

// listAdminUsers returns all users with IsSiteAdmin=true.
func (h *UserHandler) listAdminUsers(ctx context.Context) ([]models.User, error) {
	all, err := h.Users.List(ctx, "", 0, 0)
	if err != nil {
		return nil, err
	}
	return filterAdminUsers(all), nil
}

// filterAdminUsers filters a user list down to site admins only.
func filterAdminUsers(users []models.User) []models.User {
	var admins []models.User
	for _, u := range users {
		if u.IsSiteAdmin {
			admins = append(admins, u)
		}
	}
	return admins
}
