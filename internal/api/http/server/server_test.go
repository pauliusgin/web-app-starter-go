package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func silenceDefaultLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

type blockingHealthServer struct {
	server          *http.Server
	listener        net.Listener
	requestStarted  chan struct{}
	releaseHandler  chan struct{}
	stopWatchCalled chan struct{}
}

func startBlockingHealthServer(t *testing.T) *blockingHealthServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	blocking := &blockingHealthServer{
		listener:        listener,
		requestStarted:  make(chan struct{}),
		releaseHandler:  make(chan struct{}),
		stopWatchCalled: make(chan struct{}),
	}

	var announceRequest sync.Once
	blocking.server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			announceRequest.Do(func() { close(blocking.requestStarted) })
			<-blocking.releaseHandler
			io.WriteString(w, `{"status":"ok"}`)
		}),
	}

	t.Cleanup(func() {
		blocking.release()
		listener.Close()
	})

	return blocking
}

func (b *blockingHealthServer) release() {
	select {
	case <-b.releaseHandler:
	default:
		close(b.releaseHandler)
	}
}

func (b *blockingHealthServer) runParams(shutdownTimeout time.Duration) ServeUntilShutdownParams {
	var announceStopWatch sync.Once
	return ServeUntilShutdownParams{
		Server:          b.server,
		Listener:        b.listener,
		ShutdownTimeout: shutdownTimeout,
		StopSignalWatch: func() { announceStopWatch.Do(func() { close(b.stopWatchCalled) }) },
	}
}

func (b *blockingHealthServer) requestHealthInBackground(t *testing.T) {
	t.Helper()

	go func() {
		response, err := http.Get("http://" + b.listener.Addr().String() + "/api/health")
		if err != nil {
			return
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}()
}

func runLoopInBackground(params ServeUntilShutdownParams, shutdownSignal context.Context) <-chan error {
	result := make(chan error, 1)
	go func() { result <- ServeUntilShutdown(shutdownSignal, params) }()
	return result
}

func awaitRunLoop(t *testing.T, result <-chan error) error {
	t.Helper()

	select {
	case err := <-result:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("ServeUntilShutdown did not return within 10s")
		return nil
	}
}

func TestServeUntilShutdownDrainsInFlightRequest(t *testing.T) {
	silenceDefaultLogger(t)

	blocking := startBlockingHealthServer(t)
	shutdownSignal, sendShutdownSignal := context.WithCancel(context.Background())
	defer sendShutdownSignal()

	result := runLoopInBackground(blocking.runParams(5*time.Second), shutdownSignal)

	blocking.requestHealthInBackground(t)
	<-blocking.requestStarted

	sendShutdownSignal()

	select {
	case <-blocking.stopWatchCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("StopSignalWatch was not called while the drain was still in progress")
	}

	blocking.release()

	if err := awaitRunLoop(t, result); err != nil {
		t.Fatalf("ServeUntilShutdown = %v, want nil", err)
	}
}

func TestServeUntilShutdownReportsExpiredShutdownDeadline(t *testing.T) {
	silenceDefaultLogger(t)

	blocking := startBlockingHealthServer(t)
	shutdownSignal, sendShutdownSignal := context.WithCancel(context.Background())
	defer sendShutdownSignal()

	result := runLoopInBackground(blocking.runParams(50*time.Millisecond), shutdownSignal)

	blocking.requestHealthInBackground(t)
	<-blocking.requestStarted

	sendShutdownSignal()

	err := awaitRunLoop(t, result)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeUntilShutdown = %v, want context.DeadlineExceeded", err)
	}
}

func TestServeUntilShutdownReportsServeFailure(t *testing.T) {
	silenceDefaultLogger(t)

	blocking := startBlockingHealthServer(t)
	blocking.listener.Close()

	result := runLoopInBackground(blocking.runParams(5*time.Second), context.Background())

	err := awaitRunLoop(t, result)
	if err == nil {
		t.Fatal("ServeUntilShutdown = nil, want a serve failure")
	}
	if errors.Is(err, ErrStoppedWithoutShutdown) {
		t.Fatalf("ServeUntilShutdown = %v, want the underlying listener error", err)
	}
}

func TestServeUntilShutdownReportsStopWithoutShutdownRequest(t *testing.T) {
	silenceDefaultLogger(t)

	blocking := startBlockingHealthServer(t)
	result := runLoopInBackground(blocking.runParams(5*time.Second), context.Background())

	blocking.server.Close()

	err := awaitRunLoop(t, result)
	if !errors.Is(err, ErrStoppedWithoutShutdown) {
		t.Fatalf("ServeUntilShutdown = %v, want ErrStoppedWithoutShutdown", err)
	}
}
