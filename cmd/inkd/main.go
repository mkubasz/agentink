// Command inkd is the AgentInk hub daemon: the only writer of the event logs,
// the owner of every projection, the local API `ink` talks to (ADR 0005).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// Version is stamped by the Makefile's ldflags.
var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)

	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inkd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data", "", "data directory (default $"+config.EnvData+" or $XDG_DATA_HOME/agentink)")
	socket := fs.String("socket", "", "Unix socket to listen on (default $"+config.EnvSocket+" or $XDG_RUNTIME_DIR/agentink.sock)")
	verbose := fs.Bool("v", false, "debug logging")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	cfg, err := config.Resolve(*dataDir, *socket, os.Getenv)
	if err != nil {
		fmt.Fprintln(stderr, err)

		return 1
	}

	a, err := app.New(cfg, Version, logger)
	if err != nil {
		fmt.Fprintln(stderr, err)

		return 1
	}

	defer a.Close() //nolint:errcheck // nothing left to do with the error at exit

	ln, err := httpapi.ListenUnix(ctx, cfg.SocketPath)
	if err != nil {
		fmt.Fprintln(stderr, err)

		return 1
	}

	fmt.Fprintf(stdout, "inkd %s listening on %s, data in %s\n", Version, cfg.SocketPath, cfg.DataDir)

	if err := httpapi.Serve(ctx, ln, a.Handler()); err != nil {
		fmt.Fprintln(stderr, err)

		return 1
	}

	return 0
}
