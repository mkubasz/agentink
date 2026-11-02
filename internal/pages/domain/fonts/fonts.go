// Package fonts embeds the typefaces every page is set in (ADR 0006): Caveat,
// a handwriting-style face with the full Polish alphabet, under the SIL OFL.
package fonts

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// FS holds the font files and their licence.
//
//go:embed *.ttf OFL-*.txt
var FS embed.FS

// Handwriting is the family name of the handwriting-style face.
const Handwriting = "Caveat"

// Dir writes the fonts into dir (once per content) and returns it, for Typst's
// --font-path. Fonts are embedded so a render is identical on every machine.
func Dir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("fonts: %w", err)
	}

	entries, err := FS.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("fonts: %w", err)
	}

	for _, e := range entries {
		data, err := FS.ReadFile(e.Name())
		if err != nil {
			return "", fmt.Errorf("fonts: %w", err)
		}

		path := filepath.Join(dir, e.Name())
		if existing, err := os.ReadFile(path); err == nil && len(existing) == len(data) { //nolint:gosec // our own data directory
			continue
		}

		if err := os.WriteFile(path, data, 0o600); err != nil {
			return "", fmt.Errorf("fonts: %w", err)
		}
	}

	return dir, nil
}
