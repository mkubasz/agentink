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
	"strings"
	"syscall"

	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
	"github.com/mkubasz/agentink/internal/platform/webdav"
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
	davListen := fs.String("webdav-listen", "", "host:port of the readers' share (default $"+config.EnvWebDAV+" or the Tailscale interface, port 8080; off to disable)")
	davAny := fs.Bool("webdav-allow-any", false, "let the share bind outside Tailscale and loopback (development only)")
	davURL := fs.String("webdav-url", "", "the address readers type in, when a proxy such as `tailscale serve` fronts the share")

	var watch watchFlag

	fs.Var(&watch, "watch", "device=path of a synced folder to poll for ink (repeatable; also $"+config.EnvWatch+")")

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

	if *davListen != "" {
		cfg.WebDAVListen = *davListen
	}

	cfg.WebDAVAllowAny = *davAny
	cfg.WebDAVURL = *davURL
	cfg.Watch = append(cfg.Watch, watch...)

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

	errc := make(chan error, 2)

	go func() { errc <- httpapi.Serve(ctx, ln, a.Handler()) }()

	go func() { errc <- serveShare(ctx, cfg, a, stdout, logger) }()

	go a.Watch(ctx)

	var code int

	for range 2 {
		if err := <-errc; err != nil {
			fmt.Fprintln(stderr, err)

			code = 1
		}
	}

	return code
}

// serveShare runs the readers' WebDAV share on its own listener; "off" skips it.
func serveShare(ctx context.Context, cfg config.Config, a *app.App, stdout io.Writer, logger *slog.Logger) error {
	if cfg.WebDAVListen == config.WebDAVOff {
		logger.Info("webdav share off")

		return nil
	}

	ln, err := webdav.Listen(ctx, cfg.WebDAVListen, cfg.WebDAVAllowAny)
	if err != nil {
		return err
	}

	if cfg.WebDAVURL == "" {
		a.SetShareURL("http://" + ln.Addr().String() + "/Ink/")
	}

	fmt.Fprintf(stdout, "readers' share at %s\n", a.ShareURL())

	return httpapi.Serve(ctx, ln, a.ShareHandler())
}

// watchFlag collects --watch device=path entries.
type watchFlag []config.WatchFolder

func (f *watchFlag) String() string {
	parts := make([]string, 0, len(*f))
	for _, w := range *f {
		parts = append(parts, w.Device+"="+w.Path)
	}

	return strings.Join(parts, ",")
}

func (f *watchFlag) Set(v string) error {
	folders, err := config.ParseWatch(v)
	if err != nil {
		return err
	}

	*f = append(*f, folders...)

	return nil
}
