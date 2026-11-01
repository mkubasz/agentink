package config_test

import (
	"path/filepath"
	"testing"

	"github.com/mkubasz/agentink/internal/platform/config"
)

func TestResolve_PrecedenceAndDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		data, sock string
		env        map[string]string
		wantData   string
		wantSock   string
	}{
		"explicit wins": {
			data: "/d", sock: "/s.sock",
			env:      map[string]string{config.EnvData: "/e", config.EnvSocket: "/e.sock"},
			wantData: "/d", wantSock: "/s.sock",
		},
		"env over xdg": {
			env:      map[string]string{config.EnvData: "/e", "XDG_DATA_HOME": "/x", "XDG_RUNTIME_DIR": "/run"},
			wantData: "/e", wantSock: "/run/agentink.sock",
		},
		"xdg defaults": {
			env:      map[string]string{"XDG_DATA_HOME": "/x", "XDG_RUNTIME_DIR": "/run"},
			wantData: "/x/agentink", wantSock: "/run/agentink.sock",
		},
		"no runtime dir falls back to data dir": {
			env:      map[string]string{"XDG_DATA_HOME": "/x"},
			wantData: "/x/agentink", wantSock: filepath.Join("/x/agentink", "agentink.sock"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.Resolve(tc.data, tc.sock, func(k string) string { return tc.env[k] })
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			if cfg.DataDir != tc.wantData || cfg.SocketPath != tc.wantSock {
				t.Fatalf("got %+v, want data=%s sock=%s", cfg, tc.wantData, tc.wantSock)
			}
		})
	}
}
