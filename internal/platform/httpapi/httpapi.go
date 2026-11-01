// Package httpapi is the plumbing of inkd's local HTTP API: a Unix socket
// listener, JSON helpers and a server that stops with its context (ADR 0005:
// the daemon is the only writer, the CLI talks to it over this socket).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
)

// MaxBody caps a request body: a question is a few KB, never more than this.
const MaxBody = 1 << 20

// ListenUnix listens on a Unix socket, replacing a stale socket file and
// keeping the file private to the owner.
func ListenUnix(ctx context.Context, path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("httpapi: socket dir: %w", err)
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("httpapi: remove stale socket: %w", err)
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("httpapi: listen %s: %w", path, err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		_ = ln.Close()

		return nil, fmt.Errorf("httpapi: chmod socket: %w", err)
	}

	return ln, nil
}

// Serve runs h on ln until ctx is done, then shuts down gracefully.
func Serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errc := make(chan error, 1)

	go func() { errc <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpapi: shutdown: %w", err)
		}

		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("httpapi: serve: %w", err)
	}
}

// WriteJSON writes v with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes an api.Error with the given status.
func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, api.Error{Error: err.Error()})
}

// Decode reads a JSON body of at most MaxBody bytes into v.
func Decode(r *http.Request, v any) error {
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, MaxBody))
	if err != nil {
		return fmt.Errorf("httpapi: read body: %w", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("httpapi: bad JSON body: %w", err)
	}

	return nil
}
