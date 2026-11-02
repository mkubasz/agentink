// Package renderpage is the slice that turns a question into a page: the
// page document, the PDF rendered by Typst, and the zones read back from the
// same template (ADR 0006). It subscribes to QuestionAsked and emits
// PageRendered, or RenderFailed with the reason.
package renderpage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mkubasz/agentink/internal/pages/domain/fonts"
	"github.com/mkubasz/agentink/internal/pages/domain/templates"
)

// EnvTypst names the environment variable that points at the typst binary.
const EnvTypst = "AGENTINK_TYPST"

// Pinned is the Typst minor version the templates are proven against.
const Pinned = "0.15"

// ErrTypst wraps every failure of the typst binary.
var ErrTypst = errors.New("renderpage: typst")

var versionRe = regexp.MustCompile(`typst (\d+\.\d+\.\d+)`)

// Typst runs the pinned typst binary with our fonts and templates.
type Typst struct {
	bin         string
	fontDir     string
	templateDir string
}

// NewTypst locates the binary (bin, else $AGENTINK_TYPST, else `typst` on
// PATH), checks its version against Pinned, and lays out the embedded fonts
// and templates under workDir.
func NewTypst(ctx context.Context, bin, workDir string) (*Typst, error) {
	if bin == "" {
		bin = os.Getenv(EnvTypst)
	}

	if bin == "" {
		bin = "typst"
	}

	path, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("%w: not found (%s): %w", ErrTypst, bin, err)
	}

	fontDir, err := fonts.Dir(filepath.Join(workDir, "fonts"))
	if err != nil {
		return nil, err
	}

	templateDir, err := templates.Dir(filepath.Join(workDir, "templates"))
	if err != nil {
		return nil, err
	}

	t := &Typst{bin: path, fontDir: fontDir, templateDir: templateDir}

	version, err := t.Version(ctx)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(version, Pinned+".") && version != Pinned {
		return nil, fmt.Errorf("%w: version %s, the templates are pinned to %s.x (ADR 0006)", ErrTypst, version, Pinned)
	}

	return t, nil
}

// Version returns the binary's major.minor.patch.
func (t *Typst) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, t.bin, "--version").Output() //nolint:gosec // the binary was resolved by LookPath; the argument is fixed
	if err != nil {
		return "", fmt.Errorf("%w: --version: %w", ErrTypst, err)
	}

	m := versionRe.FindSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("%w: cannot read version from %q", ErrTypst, out)
	}

	return string(m[1]), nil
}

// Inputs are the JSON documents a template reads through sys.inputs.
type Inputs struct {
	Page    []byte
	Content []byte
	Size    []byte
}

func (in Inputs) args() []string {
	return []string{
		"--input", "page=" + string(in.Page),
		"--input", "content=" + string(in.Content),
		"--input", "size=" + string(in.Size),
	}
}

func (t *Typst) common() []string {
	return []string{"--root", t.templateDir, "--font-path", t.fontDir, "--ignore-system-fonts"}
}

// Compile renders the template for kind to out (a .pdf, or a .png pattern
// such as page-{p}.png).
func (t *Typst) Compile(ctx context.Context, kind string, in Inputs, out string, extra ...string) error {
	if !templates.Has(kind) {
		return fmt.Errorf("%w: no template for %q", ErrTypst, kind)
	}

	args := append([]string{"compile"}, t.common()...)
	args = append(args, in.args()...)
	args = append(args, extra...)
	args = append(args, filepath.Join(t.templateDir, kind+".typ"), out)

	return t.run(ctx, args, nil)
}

// Zones evaluates the template's metadata: every zone the page declares.
func (t *Typst) Zones(ctx context.Context, kind string, in Inputs) ([]json.RawMessage, error) {
	if !templates.Has(kind) {
		return nil, fmt.Errorf("%w: no template for %q", ErrTypst, kind)
	}

	args := append([]string{"eval"}, t.common()...)
	args = append(args, in.args()...)
	args = append(args, "--in", filepath.Join(t.templateDir, kind+".typ"), "query(metadata).map(it => it.value)")

	var out bytes.Buffer
	if err := t.run(ctx, args, &out); err != nil {
		return nil, err
	}

	var zones []json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &zones); err != nil {
		return nil, fmt.Errorf("%w: zones are not JSON: %w", ErrTypst, err)
	}

	return zones, nil
}

func (t *Typst) run(ctx context.Context, args []string, stdout *bytes.Buffer) error {
	cmd := exec.CommandContext(ctx, t.bin, args...) //nolint:gosec // the binary was resolved by NewTypst, the arguments are ours

	var stderr bytes.Buffer

	cmd.Stderr = &stderr
	if stdout != nil {
		cmd.Stdout = stdout
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w %s: %w: %s", ErrTypst, args[0], err, strings.TrimSpace(stderr.String()))
	}

	return nil
}
