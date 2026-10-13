// Package server runs an http.Server until a shutdown signal arrives and drains it on the way out.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ServeUntilShutdownParams carries the server, its listener and the deadlines a drain runs under.
type ServeUntilShutdownParams struct {
	Server          *http.Server
	Listener        net.Listener
	ShutdownTimeout time.Duration
	StopSignalWatch func()
}

// ServeUntilShutdown serves until shutdownSignal fires, then drains within ShutdownTimeout.
func ServeUntilShutdown(shutdownSignal context.Context, params ServeUntilShutdownParams) error {
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- params.Server.Serve(params.Listener)
	}()

	select {
	case err := <-serveResult:
		return classifyUnexpectedServeStop(err)
	case <-shutdownSignal.Done():
	}

	params.StopSignalWatch()
	slog.Info("shutdown signal received", "timeout", params.ShutdownTimeout)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), params.ShutdownTimeout)
	defer cancelShutdown()

	shutdownErr := params.Server.Shutdown(shutdownCtx)

	if racedFailure := takeRacedServeFailure(serveResult); racedFailure != nil {
		return racedFailure
	}
	if shutdownErr != nil {
		return shutdownErr
	}

	return nil
}

func classifyUnexpectedServeStop(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return ErrStoppedWithoutShutdown
	}
	return err
}

func takeRacedServeFailure(serveResult <-chan error) error {
	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	default:
		return nil
	}
}
