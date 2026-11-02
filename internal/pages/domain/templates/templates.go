// Package templates embeds the Typst templates, one per page kind, plus the
// shared library (ADR 0006).
package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// FS holds *.typ.
//
//go:embed *.typ
var FS embed.FS

// Dir writes the templates into dir and returns it; typst compiles from a
// file, so the embedded sources are laid out once per run.
func Dir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("templates: %w", err)
	}

	entries, err := FS.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("templates: %w", err)
	}

	for _, e := range entries {
		data, err := FS.ReadFile(e.Name())
		if err != nil {
			return "", fmt.Errorf("templates: %w", err)
		}

		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o600); err != nil {
			return "", fmt.Errorf("templates: %w", err)
		}
	}

	return dir, nil
}

// Has reports whether a template exists for the page kind.
func Has(kind string) bool {
	_, err := FS.Open(kind + ".typ")

	return err == nil
}
