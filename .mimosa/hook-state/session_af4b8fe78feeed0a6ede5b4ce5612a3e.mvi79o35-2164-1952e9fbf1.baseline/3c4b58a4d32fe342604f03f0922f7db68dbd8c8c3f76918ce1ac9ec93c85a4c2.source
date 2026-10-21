package handlers

import (
	"context"

	"gitf/internal/middleware"
	"gitf/internal/models"
)

// UserFromContext retrieves the authenticated user from request context.
// Thin re-export so handler files don't have to import the middleware package
// directly for this very common operation.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	return middleware.UserFromContext(ctx)
}
