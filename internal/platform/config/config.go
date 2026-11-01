// Package config resolves where the hub keeps its data and listens, from
// flags, environment and XDG defaults (ADR 0002, ADR 0005).
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Env names the environment variables the hub and the CLI read.
const (
	EnvData   = "AGENTINK_DATA"
	EnvSocket = "AGENTINK_SOCKET"
)

// Config is what inkd needs to start and what ink needs to find it.
type Config struct {
	// DataDir holds projects/<slug>/events.jsonl and pages (ADR 0002).
	DataDir string
	// SocketPath is the Unix socket inkd listens on and ink talks to.
	SocketPath string
}

// Resolve fills Config from explicit values, then the environment, then
// XDG defaults. Empty explicit values mean "not given".
func Resolve(dataDir, socketPath string, getenv func(string) string) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("config: home directory: %w", err)
	}

	cfg := Config{DataDir: first(dataDir, getenv(EnvData)), SocketPath: first(socketPath, getenv(EnvSocket))}

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
