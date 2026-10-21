package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/models"
)

// UserLookup is the minimal user-store contract the auth middleware needs.
// Mirrors the wider handlers.UserStore but is declared here to avoid an
// import cycle (handlers imports middleware, not the other way around).
type UserLookup interface {
	ByUsername(ctx context.Context, username string) (*models.User, error)
}

// PATLookup resolves a plain personal-access token to its persisted record.
type PATLookup interface {
	Lookup(plainToken string) (*models.PersonalAccessToken, error)
}

// UserByIDLookup extends UserLookup with ID resolution; implemented by the
// same GORM store, so no extra wiring is needed.
type UserByIDLookup interface {
	UserLookup
	ByID(ctx context.Context, id uint) (*models.User, error)
}

// Auth returns a middleware that authenticates requests via JWT, personal
// access token, or Basic Auth (Git HTTP). Pass a UserLookup so the auth path
// is not coupled to *gorm.DB. Optionally pass a PATLookup to enable
// "Authorization: Bearer gitf_..." tokens.
func Auth(users UserLookup, cfg *config.Config, pats ...PATLookup) func(http.Handler) http.Handler {
	return authMiddleware(users, cfg, pats, false)
}

// OptionalAuth works like Auth but allows unauthenticated requests.
func OptionalAuth(users UserLookup, cfg *config.Config, pats ...PATLookup) func(http.Handler) http.Handler {
	return authMiddleware(users, cfg, pats, true)
}

func authMiddleware(users UserLookup, cfg *config.Config, pats []PATLookup, optional bool) func(http.Handler) http.Handler {
	var patLookup PATLookup
	if len(pats) > 0 {
		patLookup = pats[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := authenticate(r, users, cfg)
			if !ok && patLookup != nil {
				if byID, canResolveID := users.(UserByIDLookup); canResolveID {
					user, ok = authenticatePAT(r, byID, patLookup)
				}
			}
			if ok {
				ctx := WithUser(r.Context(), user)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}
			if optional {
				next.ServeHTTP(w, r)
				return
			}
			writeUnauthorized(w, r)
		})
	}
}

// authenticatePAT resolves "Bearer gitf_..." personal access tokens.
func authenticatePAT(r *http.Request, users UserByIDLookup, pats PATLookup) (*models.User, bool) {
	authHeader := r.Header.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, false
	}
	token := parts[1]
	if !strings.HasPrefix(token, "gitf_") {
		return nil, false
	}
	t, err := pats.Lookup(token)
	if err != nil || t == nil {
		return nil, false
	}
	user, err := users.ByID(r.Context(), t.UserID)
	if err != nil || user == nil {
		return nil, false
	}
	return user, true
}

func authenticate(r *http.Request, users UserLookup, cfg *config.Config) (*models.User, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, false
	}
	scheme, token := parts[0], parts[1]

	switch strings.ToLower(scheme) {
	case "bearer":
		claims, err := auth.ValidateToken(token, cfg.JWTSecret)
		if err != nil {
			return nil, false
		}
		// Look up the user from the store so that role flags like IsSiteAdmin
		// reflect the current DB state. This also ensures revocations (e.g. a
		// demoted admin) take effect before the JWT expires, and prevents
		// forged-claim escalation where a tampered token could otherwise
		// surface arbitrary IsSiteAdmin/ID values.
		user, err := users.ByUsername(r.Context(), claims.Username)
		if err != nil || user == nil {
			return nil, false
		}
		if user.ID != claims.UserID {
			// Claims do not match the DB record — treat as invalid.
			return nil, false
		}
		return user, true
	case "basic":
		// Basic auth should not be logged or recorded in audit logs to prevent credential leakage
		decoded, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return nil, false
		}
		creds := strings.SplitN(string(decoded), ":", 2)
		if len(creds) != 2 {
			return nil, false
		}
		username, password := creds[0], creds[1]
		user, err := users.ByUsername(r.Context(), username)
		if err != nil {
			auth.ConstantTimeDelay() // Prevent timing attacks
			return nil, false
		}
		if !auth.CheckPassword(password, user.PasswordHash) {
			auth.ConstantTimeDelay() // Prevent timing attacks
			return nil, false
		}
		// Rate limiting for Basic auth should be implemented at the middleware level
		// This is handled in the git HTTP protocol handler
		return user, true
	}

	return nil, false
}
