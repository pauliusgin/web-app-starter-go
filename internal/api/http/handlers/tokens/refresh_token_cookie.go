package tokens

import (
	"net/http"
	"time"
)

// RefreshTokenCookieName names the cookie the refresh token travels in.
const RefreshTokenCookieName = "refresh_token"

// RefreshTokenCookiePath scopes the cookie to the token routes so it never rides along on other API calls.
// It must stay equal to routes.AuthRoutePath; routes_test guards the pair.
const RefreshTokenCookiePath = "/api/v1/tokens"

// SetRefreshTokenCookieParams carries the issued refresh token the cookie holds.
type SetRefreshTokenCookieParams struct {
	Plaintext string
	ExpiresAt time.Time
}

// SetRefreshTokenCookie hands the refresh token to the browser as an httpOnly cookie expiring with the token.
func SetRefreshTokenCookie(w http.ResponseWriter, params SetRefreshTokenCookieParams) {
	cookie := newRefreshTokenCookie(params.Plaintext)
	cookie.Expires = params.ExpiresAt
	cookie.MaxAge = int(time.Until(params.ExpiresAt).Seconds())
	http.SetCookie(w, cookie)
}

// ClearRefreshTokenCookie tells the browser to drop the refresh token cookie.
func ClearRefreshTokenCookie(w http.ResponseWriter) {
	cookie := newRefreshTokenCookie("")
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}

// ReadRefreshTokenCookie returns the refresh token the request's cookie carries.
func ReadRefreshTokenCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil || cookie.Value == "" {
		return "", ErrRefreshTokenCookieMissing{}
	}
	return cookie.Value, nil
}

// Secure is set unconditionally: browsers accept Secure cookies on http://localhost, and the Vite dev
// proxy keeps the API same-site, so dev needs no exception.
func newRefreshTokenCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    value,
		Path:     RefreshTokenCookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}
