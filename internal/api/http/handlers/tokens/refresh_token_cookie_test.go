package tokens

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func findRefreshTokenCookie(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == RefreshTokenCookieName {
			return cookie
		}
	}
	t.Fatal("response sets no refresh token cookie")
	return nil
}

func assertRefreshTokenCookieCleared(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	cookie := findRefreshTokenCookie(t, recorder)
	if cookie.MaxAge >= 0 || cookie.Value != "" {
		t.Errorf("refresh token cookie = %+v, want an expired empty cookie", cookie)
	}
	if cookie.Path != RefreshTokenCookiePath {
		t.Errorf("cleared cookie path = %q, want %q so it replaces the set one", cookie.Path, RefreshTokenCookiePath)
	}
}

func TestSetRefreshTokenCookieHidesTheTokenFromScripts(t *testing.T) {
	expiresAt := liveTokenExpiresAt
	recorder := httptest.NewRecorder()

	SetRefreshTokenCookie(recorder, SetRefreshTokenCookieParams{
		Plaintext: presentedRefreshTokenForTest,
		ExpiresAt: expiresAt,
	})

	cookie := findRefreshTokenCookie(t, recorder)
	if cookie.Value != presentedRefreshTokenForTest {
		t.Errorf("value = %q, want %q", cookie.Value, presentedRefreshTokenForTest)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("flags = %+v, want HttpOnly, Secure, SameSite=Strict", cookie)
	}
	if cookie.Path != RefreshTokenCookiePath {
		t.Errorf("path = %q, want %q", cookie.Path, RefreshTokenCookiePath)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Errorf("expires = %s, want %s", cookie.Expires, expiresAt)
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("max age = %d, want a positive lifetime", cookie.MaxAge)
	}
}

func TestReadRefreshTokenCookieReturnsTheCarriedToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, RefreshTokenCookiePath+"/refresh", nil)
	request.AddCookie(&http.Cookie{Name: RefreshTokenCookieName, Value: presentedRefreshTokenForTest})

	presentedToken, err := ReadRefreshTokenCookie(request)

	if err != nil {
		t.Fatalf("ReadRefreshTokenCookie error = %v, want none", err)
	}
	if presentedToken != presentedRefreshTokenForTest {
		t.Errorf("token = %q, want %q", presentedToken, presentedRefreshTokenForTest)
	}
}

func TestReadRefreshTokenCookieRejectsAMissingOrEmptyCookie(t *testing.T) {
	cases := []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "absent"},
		{name: "empty", cookie: &http.Cookie{Name: RefreshTokenCookieName, Value: ""}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, RefreshTokenCookiePath+"/refresh", nil)
			if testCase.cookie != nil {
				request.AddCookie(testCase.cookie)
			}

			_, err := ReadRefreshTokenCookie(request)

			if !errors.As(err, &ErrRefreshTokenCookieMissing{}) {
				t.Errorf("error = %v, want ErrRefreshTokenCookieAbsent", err)
			}
		})
	}
}
