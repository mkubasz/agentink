// Package config resolves where the hub keeps its data and listens, from
// flags, environment and XDG defaults (ADR 0002, ADR 0005).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Env names the environment variables the hub and the CLI read.
const (
	EnvData   = "AGENTINK_DATA"
	EnvSocket = "AGENTINK_SOCKET"
	EnvTypst  = "AGENTINK_TYPST"
	EnvDevice = "AGENTINK_DEVICE"
	EnvWebDAV = "AGENTINK_WEBDAV"
	EnvWatch  = "AGENTINK_WATCH"
)

// WebDAVOff disables the readers' share.
const WebDAVOff = "off"

// Config is what inkd needs to start and what ink needs to find it.
type Config struct {
	// DataDir holds projects/<slug>/events.jsonl and pages (ADR 0002).
	DataDir string
	// SocketPath is the Unix socket inkd listens on and ink talks to.
	SocketPath string
	// TypstBin is the typst binary (default: `typst` on PATH); empty means "look it up".
	TypstBin string
	// Device is the default device profile pages are rendered for (ADR 0006).
	Device string
	// WebDAVListen is host:port for the readers' share; an empty host means
	// the Tailscale interface; "off" disables it (brief §6.3).
	WebDAVListen string
	// WebDAVAllowAny lets the share bind outside Tailscale and loopback
	// (development only; brief §7 forbids it in use).
	WebDAVAllowAny bool
	// WebDAVURL is what the owner types into a reader (for `ink device add`);
	// empty means "derive from the listen address".
	WebDAVURL string
	// Watch lists synced folders the hub polls for ink that did not come over
	// WebDAV (a Boox returning the annotated PDF through Syncthing, brief §6.3).
	Watch []WatchFolder
}

// WatchFolder is one synced folder and the device credited with what
// appears in it.
type WatchFolder struct {
	Device string
	// Path holds Ink/<project>/… .
	Path string
}

// ParseWatch reads `device=path` entries separated by commas, as $AGENTINK_WATCH
// and each --watch flag give them.
func ParseWatch(spec string) ([]WatchFolder, error) {
	var out []WatchFolder

	for entry := range strings.SplitSeq(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		device, path, ok := strings.Cut(entry, "=")
		if !ok || strings.TrimSpace(device) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("config: watch entry %q: want device=path", entry)
		}

		out = append(out, WatchFolder{Device: strings.TrimSpace(device), Path: filepath.Clean(strings.TrimSpace(path))})
	}

	return out, nil
}

// Resolve fills Config from explicit values, then the environment, then
// XDG defaults. Empty explicit values mean "not given".
func Resolve(dataDir, socketPath string, getenv func(string) string) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("config: home directory: %w", err)
	}

	cfg := Config{
		DataDir: first(dataDir, getenv(EnvData)), SocketPath: first(socketPath, getenv(EnvSocket)),
		TypstBin: getenv(EnvTypst), Device: getenv(EnvDevice),
		WebDAVListen: first(getenv(EnvWebDAV), ":8080"),
	}

	watch, err := ParseWatch(getenv(EnvWatch))
	if err != nil {
		return Config{}, err
	}

	cfg.Watch = watch

	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(first(getenv("XDG_DATA_HOME"), filepath.Join(home, ".local", "share")), "agentink")
	}

	if cfg.SocketPath == "" {
		runtime := getenv("XDG_RUNTIME_DIR")
		if runtime == "" {
			runtime = cfg.DataDir
		}

		cfg.SocketPath = filepath.Join(runtime, "agentink.sock")
	}

	return cfg, nil
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
