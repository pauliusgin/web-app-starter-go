package handlers

import (
	"errors"
	"fmt"
)

const responseBodyFailureText = "could not write the response body"

// RouteNotProtectedText is logged when an authenticated route runs without the auth middleware.
const RouteNotProtectedText = "an authenticated route was reached with no authenticated user, so it is registered without the authentication middleware"

var (
	errEmptyRequestBody    = errors.New("the request body is empty")
	errTrailingRequestBody = errors.New("the request body holds more than one JSON value")
)

// ErrJSONRequestBodyUndecodable reports a request body that is empty, malformed or the wrong shape.
type ErrJSONRequestBodyUndecodable struct {
	Cause error
}

func (e ErrJSONRequestBodyUndecodable) Error() string {
	return fmt.Sprintf("could not read the request body: %s", e.Cause)
}

func (e ErrJSONRequestBodyUndecodable) Unwrap() error {
	return e.Cause
}

// ErrRequestBodyTooLarge reports a request body over the decoder's byte limit.
type ErrRequestBodyTooLarge struct {
	Limit int64
}

func (e ErrRequestBodyTooLarge) Error() string {
	return fmt.Sprintf("the request body is larger than the %d byte limit", e.Limit)
}

// ErrUnsupportedContentType reports a request whose Content-Type is not application/json.
type ErrUnsupportedContentType struct {
	ContentType string
}

func (e ErrUnsupportedContentType) Error() string {
	return fmt.Sprintf("unsupported Content-Type %q: expected %s", e.ContentType, jsonMediaType)
}
