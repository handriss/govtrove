package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
)

// OriginVerifyHeader is set by a Cloudflare Transform Rule on every request
// proxied to api.govtrove.com. Requests arriving without it reached the App
// Runner origin directly, bypassing the WAF, bot protection and rate limiting.
const OriginVerifyHeader = "X-Origin-Verify"

// RequireOriginVerify rejects requests that did not come through Cloudflare.
//
// An empty secret disables the check. That is deliberate: the Cloudflare rule
// and this middleware cannot land atomically, so the secret is set only once
// the rule is confirmed to be adding the header. Deploying with it unset leaves
// behaviour exactly as before. Startup logs which mode is active.
func RequireOriginVerify(secret string, logger *slog.Logger) func(http.Handler) http.Handler {
	expected := []byte(secret)

	return func(next http.Handler) http.Handler {
		if len(expected) == 0 {
			logger.Warn("origin verification disabled: ORIGIN_VERIFY_SECRET is unset; the App Runner origin is reachable directly, bypassing Cloudflare")
			return next
		}

		logger.Info("origin verification enabled", "header", OriginVerifyHeader)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			presented := []byte(r.Header.Get(OriginVerifyHeader))
			if subtle.ConstantTimeCompare(presented, expected) != 1 {
				http.Error(w, "Not found.", http.StatusNotFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
