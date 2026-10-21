package handlers

import (
	"errors"
	"net/http"

	"gitf/internal/auth"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type PATHandler struct {
	Tokens auth.TokenStore
}

func NewPATHandler(db *gorm.DB) *PATHandler {
	return &PATHandler{Tokens: auth.NewTokenStore(db)}
}

func (h *PATHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	var req struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}
	if !decodeJSON(w, r, &req) { return }
	if req.Name == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "name is required")
		return
	}
	if req.Scopes == "" { req.Scopes = "repo" }
	plain, token, err := h.Tokens.Create(user.ID, req.Name, req.Scopes)
	if err != nil {
		writeServerError(w, r, "create token", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{
		"token": plain,
		"info":  token.DTO(),
	})
}

func (h *PATHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	tokens, err := h.Tokens.List(user.ID)
	if err != nil {
		writeServerError(w, r, "list tokens", err)
		return
	}
	items := make([]any, len(tokens))
	for i, t := range tokens {
		items[i] = t.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *PATHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	id := atoiOrZero(chi.URLParam(r, "tokenID"))
	if err := h.Tokens.Delete(user.ID, uint(id)); err != nil {
		if errors.Is(err, auth.ErrTokenNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "token not found")
			return
		}
		writeServerError(w, r, "delete token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
