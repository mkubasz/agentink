// Package arch holds the import-boundary test of ADR 0001: a slice imports
// only its context's domain, the platform and the contracts; a domain imports
// only the platform and the contracts; the platform imports no context.
package arch_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/mkubasz/agentink/"

func TestArch_SlicesImportOnlyDomainAndPlatform(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")

	var violations []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}

		rel, _ := filepath.Rel(root, path)
		pkgDir := filepath.ToSlash(filepath.Dir(rel))

		for _, imp := range imports(t, path) {
			if !allowed(pkgDir, imp) {
				violations = append(violations, rel+" imports "+imp)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(violations) > 0 {
		t.Fatalf("import boundary (ADR 0001) broken:\n  %s", strings.Join(violations, "\n  "))
	}
}

// allowed says whether a file in pkgDir (e.g. internal/inbox/askquestion)
// may import imp.
func allowed(pkgDir, imp string) bool {
	if !strings.HasPrefix(imp, module) {
		return true // standard library and third parties
	}

	target := strings.TrimPrefix(imp, module)
	parts := strings.Split(pkgDir, "/")

	switch parts[1] {
	case "app", "arch":
		return true // the composition root wires everything
	case "platform":
		return target == "contracts" || strings.HasPrefix(target, "internal/platform/")
	}

	// A context package: internal/<ctx>/<domain|slice>. The domain and its
	// sub-packages (assets such as templates and fonts) are shared inside the
	// context; slices never import each other.
	ctx := parts[1]
	domain := "internal/" + ctx + "/domain"

	if target == "contracts" || strings.HasPrefix(target, "internal/platform/") {
		return true
	}

	if target == domain || strings.HasPrefix(target, domain+"/") {
		return len(parts) < 3 || parts[2] != "domain" || target != domain
	}

	return false
}

func imports(t *testing.T, path string) []string {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}

	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		out = append(out, strings.Trim(imp.Path.Value, `"`))
	}

	return out
}

func TestArch_RejectsACrossContextImport(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		pkg, imp string
		ok       bool
	}{
		"slice to own domain":     {"internal/inbox/askquestion", module + "internal/inbox/domain", true},
		"slice to other domain":   {"internal/inbox/askquestion", module + "internal/projects/domain", false},
		"slice to other slice":    {"internal/inbox/askquestion", module + "internal/inbox/listinbox", false},
		"slice to platform":       {"internal/inbox/askquestion", module + "internal/platform/bus", true},
		"domain to own context":   {"internal/inbox/domain", module + "internal/inbox/askquestion", false},
		"platform to a context":   {"internal/platform/bus", module + "internal/inbox/domain", false},
		"platform to contracts":   {"internal/platform/bus", module + "contracts", true},
		"anything to the std lib": {"internal/inbox/domain", "encoding/json", true},
		"app to everything":       {"internal/app", module + "internal/inbox/askquestion", true},
	}

	for name, tc := range cases {
		if got := allowed(tc.pkg, tc.imp); got != tc.ok {
			t.Errorf("%s: allowed(%s, %s) = %v, want %v", name, tc.pkg, tc.imp, got, tc.ok)
		}
	}
}
