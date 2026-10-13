package tokens

import "fmt"

// VerificationTokenRequiredText is the client-facing answer to a verify request with no token.
const VerificationTokenRequiredText = "a verification token is required"

const accessTokenIssuingFailedText = "failed issuing the access token for the user"

const refreshTokenIssuingFailedText = "failed issuing the refresh token for the user"

// ErrVerificationTokenAbsent reports a verify request that carried no token query parameter.
type ErrVerificationTokenAbsent struct{}

func (e ErrVerificationTokenAbsent) Error() string {
	return fmt.Sprintf("missing verification token: %s", VerificationTokenRequiredText)
}

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
