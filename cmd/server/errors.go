package main

import (
	"fmt"
	"strings"
)

// ErrEmailGatewayUnavailable reports an email gateway that could not be built from its config.
type ErrEmailGatewayUnavailable struct {
	Variables []string
	Cause     error
}

func (e ErrEmailGatewayUnavailable) Error() string {
	return fmt.Sprintf("email gateway unavailable: check %s: %v",
		strings.Join(e.Variables, ", "), e.Cause)
}

func (e ErrEmailGatewayUnavailable) Unwrap() error {
	return e.Cause
}

// ErrFileStorageUnavailable reports a file storage that could not be built from its config.
type ErrFileStorageUnavailable struct {
	Variables []string
	Cause     error
}

func (e ErrFileStorageUnavailable) Error() string {
	return fmt.Sprintf("file storage unavailable: check %s: %v",
		strings.Join(e.Variables, ", "), e.Cause)
}

func (e ErrFileStorageUnavailable) Unwrap() error {
	return e.Cause
}

// ErrImageProcessorUnavailable reports an image processor that could not start, e.g. libvips missing.
type ErrImageProcessorUnavailable struct {
	Cause error
}

func (e ErrImageProcessorUnavailable) Error() string {
	return fmt.Sprintf("image processor unavailable: is libvips installed? %v", e.Cause)
}

func (e ErrImageProcessorUnavailable) Unwrap() error {
	return e.Cause
}
