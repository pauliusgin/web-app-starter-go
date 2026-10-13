package server

import "errors"

// ErrStoppedWithoutShutdown reports a server that stopped serving before any shutdown signal.
var ErrStoppedWithoutShutdown = errors.New("server stopped serving without a shutdown request")
