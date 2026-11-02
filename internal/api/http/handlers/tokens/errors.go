package tokens

import "fmt"

const accessTokenIssuingFailedText = "failed issuing the access token for the user"

const refreshTokenIssuingFailedText = "failed issuing the refresh token for the user"

const refreshTokenCookieMissingText = "missing refresh token cookie"

// ErrAccessTokenIssuingFailed reports an access token that could not be issued after signup or sign-in.
type ErrAccessTokenIssuingFailed struct {
	Cause error
}

func (e ErrAccessTokenIssuingFailed) Error() string {
	return fmt.Sprintf("%s: %s", accessTokenIssuingFailedText, e.Cause)
}

func (e ErrAccessTokenIssuingFailed) Unwrap() error {
	return e.Cause
}

// ErrRefreshTokenIssuingFailed reports a refresh token that could not be issued after signup or sign-in.
type ErrRefreshTokenIssuingFailed struct {
	Cause error
}

func (e ErrRefreshTokenIssuingFailed) Error() string {
	return fmt.Sprintf("%s: %s", refreshTokenIssuingFailedText, e.Cause)
}

func (e ErrRefreshTokenIssuingFailed) Unwrap() error {
	return e.Cause
}

// ErrRefreshTokenCookieMissing reports a token request that carried no refresh token cookie.
type ErrRefreshTokenCookieMissing struct{}

func (e ErrRefreshTokenCookieMissing) Error() string {
	return refreshTokenCookieMissingText
}
