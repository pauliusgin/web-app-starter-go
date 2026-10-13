package refreshtoken

import "fmt"

const refreshTokenNotFoundText = "no stored refresh token matches the presented one"
const refreshTokenLookupFailedText = "failed finding the refresh token by its hash"
const saveRefreshTokenFailedText = "failed saving the refresh token"
const revokeRefreshTokenFailedText = "failed revoking the refresh token"
const deleteRefreshTokensFailedText = "failed deleting the user's refresh tokens"
const deleteUnusedRefreshTokensFailedText = "failed deleting the unused refresh tokens"

// ErrRefreshTokenNotFound reports a presented refresh token that matches no live stored row.
type ErrRefreshTokenNotFound struct{}

func (e ErrRefreshTokenNotFound) Error() string {
	return fmt.Sprintf("unknown refresh token: %s", refreshTokenNotFoundText)
}

// ErrRefreshTokenLookupFailed reports a lookup by token hash the database could not complete.
type ErrRefreshTokenLookupFailed struct {
	Cause error
}

func (e ErrRefreshTokenLookupFailed) Error() string {
	return fmt.Sprintf("%s: %s", refreshTokenLookupFailedText, e.Cause)
}

func (e ErrRefreshTokenLookupFailed) Unwrap() error {
	return e.Cause
}

// ErrSaveRefreshTokenFailed reports a refresh token the database refused to store.
type ErrSaveRefreshTokenFailed struct {
	Cause error
}

func (e ErrSaveRefreshTokenFailed) Error() string {
	return fmt.Sprintf("%s: %s", saveRefreshTokenFailedText, e.Cause)
}

func (e ErrSaveRefreshTokenFailed) Unwrap() error {
	return e.Cause
}

// ErrRevokeRefreshTokenFailed reports a revocation the database could not complete.
type ErrRevokeRefreshTokenFailed struct {
	Cause error
}

func (e ErrRevokeRefreshTokenFailed) Error() string {
	return fmt.Sprintf("%s: %s", revokeRefreshTokenFailedText, e.Cause)
}

func (e ErrRevokeRefreshTokenFailed) Unwrap() error {
	return e.Cause
}

// ErrDeleteRefreshTokensFailed reports a user's refresh tokens the database could not delete.
type ErrDeleteRefreshTokensFailed struct {
	Cause error
}

func (e ErrDeleteRefreshTokensFailed) Error() string {
	return fmt.Sprintf("%s: %s", deleteRefreshTokensFailedText, e.Cause)
}

func (e ErrDeleteRefreshTokensFailed) Unwrap() error {
	return e.Cause
}

// ErrDeleteUnusedRefreshTokensFailed reports a purge of dead refresh tokens the database refused.
type ErrDeleteUnusedRefreshTokensFailed struct {
	Cause error
}

func (e ErrDeleteUnusedRefreshTokensFailed) Error() string {
	return fmt.Sprintf("%s: %s", deleteUnusedRefreshTokensFailedText, e.Cause)
}

func (e ErrDeleteUnusedRefreshTokensFailed) Unwrap() error {
	return e.Cause
}
