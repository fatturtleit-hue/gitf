package middleware

import "net/http"

// SecurityHeaders sets HTTP response headers that put back-stops behind every
// other defense in the application.
//
// CSP: script-src restricts scripts to same-origin only. Vue/Vite builds
// external hashed chunks, so inline scripts are unnecessary.

const csp = "" +
	"default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"object-src 'none'"

const cspTLS = csp +
	"; block-all-mixed-content" +
	"; upgrade-insecure-requests"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), interest-cohort=()")

		if r.TLS != nil {
			h.Set("Content-Security-Policy", cspTLS)
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		} else {
			h.Set("Content-Security-Policy", csp)
		}
		next.ServeHTTP(w, r)
	})
}
