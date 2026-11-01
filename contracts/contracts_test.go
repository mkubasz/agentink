package contracts_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mkubasz/agentink/contracts"
)

func TestContracts_Examples(t *testing.T) {
	t.Parallel()

	for _, name := range contracts.Names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			valid, _ := filepath.Glob(filepath.Join("examples", name, "*.json"))
			invalid, _ := filepath.Glob(filepath.Join("examples", name, "invalid", "*.json"))

			if len(valid) == 0 || len(invalid) == 0 {
				t.Fatalf("%s: want at least one valid and one invalid example, got %d/%d", name, len(valid), len(invalid))
			}

			for _, path := range valid {
				doc, err := os.ReadFile(path) //nolint:gosec // test fixture path from Glob
				if err != nil {
					t.Fatal(err)
				}

				if err := contracts.Validate(name, doc); err != nil {
					t.Errorf("%s should be valid: %v", path, err)
				}
			}

			for _, path := range invalid {
				doc, err := os.ReadFile(path) //nolint:gosec // test fixture path from Glob
				if err != nil {
					t.Fatal(err)
				}

				if err := contracts.Validate(name, doc); !errors.Is(err, contracts.ErrInvalid) {
					t.Errorf("%s should be invalid, got %v", path, err)
				}
			}
		})
	}
}

func TestContracts_UnknownNameAndBadJSON(t *testing.T) {
	t.Parallel()

	if err := contracts.Validate("nope", []byte(`{}`)); err == nil {
		t.Fatal("unknown contract accepted")
	}

	if err := contracts.Validate(contracts.Event, []byte(`{not json`)); !errors.Is(err, contracts.ErrInvalid) {
		t.Fatalf("bad JSON: err = %v", err)
	}
}
