package handlers

import "net/http"

// RequireAdmin is middleware that ensures the current user is a site administrator.
// Must be placed after middleware.Auth or middleware.OptionalAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || user == nil {
			writeAPIError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "unauthorized")
			return
		}
		if !user.IsSiteAdmin {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
