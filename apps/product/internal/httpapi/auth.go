package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Caller is a service that may call this one, and the bearer token that
// identifies it. Each caller holds a token of its own, so a request names
// its source (ADR-0033).
type Caller struct {
	Name  string
	Token string
}

// RequireBearer admits a request only when its bearer token belongs to one
// of the callers, and answers 401 otherwise.
//
// Every token is compared in constant time, and the loop never stops early,
// so the time a rejection takes does not reveal how much of a token matched
// or which caller it came close to.
func RequireBearer(callers []Caller, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		matched := false
		for _, c := range callers {
			if subtle.ConstantTimeCompare([]byte(token), []byte(c.Token)) == 1 {
				matched = true
			}
		}
		if !ok || !matched {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, r, http.StatusUnauthorized, codeUnauthorized,
				"a bearer token from an accepted caller is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerToken reads the token from the Authorization header. The scheme is
// case insensitive (RFC 9110), and an empty token counts as none.
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
