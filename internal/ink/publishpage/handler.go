// Package publishpage is the slice that puts a rendered page where the readers
// see it: `Ink/<project>/<page-id>-<title>.pdf` on the hub's WebDAV share
// (brief §6.3). It reacts to PageRendered and emits PagePublished.
package publishpage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Events this slice reacts to and emits.
const (
	EventRendered  = "PageRendered"
	EventPublished = "PagePublished"
)

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Share is where the readers' tree lives.
type Share interface {
	ProjectDir(project string) string
}

// Published is the payload of PagePublished.
type Published struct {
	PageID     string `json:"page_id"`
	QuestionID string `json:"question_id"`
	PageDir    string `json:"page_dir"`
	// Path is the file below the share's root, e.g. "Ink/shop/01J…-which-printer.pdf".
	Path   string `json:"path"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Handler publishes pages.
type Handler struct {
	bus   Publisher
	share Share
	now   func() time.Time
}

// New wires the slice.
func New(bus Publisher, share Share, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	return &Handler{bus: bus, share: share, now: now}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Handle copies the page's PDF onto the share and emits PagePublished.
func (h *Handler) Handle(ctx context.Context, e eventlog.Envelope) error {
	if e.Type != EventRendered {
		return nil
	}

	var r struct {
		QuestionID string `json:"question_id"`
		PageID     string `json:"page_id"`
		PageDir    string `json:"page_dir"`
		PDF        string `json:"pdf"`
	}

	if err := json.Unmarshal(e.Data, &r); err != nil {
		return fmt.Errorf("publishpage: %s: %w", e.ID, err)
	}

	file := r.PageID + "-" + titleSlug(filepath.Join(r.PageDir, "page.json")) + ".pdf"
	target := filepath.Join(h.share.ProjectDir(e.Project), file)

	sum, size, err := copyFile(filepath.Join(r.PageDir, r.PDF), target)
	if err != nil {
		return err
	}

	data, err := json.Marshal(Published{
		PageID: r.PageID, QuestionID: r.QuestionID, PageDir: r.PageDir,
		Path: "Ink/" + e.Project + "/" + file, File: file, SHA256: sum, Size: size,
	})
	if err != nil {
		return fmt.Errorf("publishpage: marshal: %w", err)
	}

	at := h.now()
	corr := e.CorrelationID

	if corr == "" {
		corr = e.ID
	}

	return h.bus.Publish(ctx, eventlog.Envelope{
		ID: eventlog.NewID(at), Type: EventPublished, At: at, Project: e.Project, Subject: e.Subject, Rev: e.Rev,
		Data: data, CausationID: e.ID, CorrelationID: corr,
	})
}

// titleSlug reads the page's title and makes a short file-name-safe slug.
func titleSlug(pageJSON string) string {
	raw, err := os.ReadFile(pageJSON) //nolint:gosec // the page directory is the hub's own
	if err != nil {
		return "page"
	}

	var p struct {
		Title string `json:"title"`
	}

	_ = json.Unmarshal(raw, &p)

	slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(p.Title), "-"), "-")
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}

	if slug == "" {
		return "page"
	}

	return slug
}

// copyFile copies src to dst through a temp file and returns the sha256 and size.
func copyFile(src, dst string) (string, int64, error) {
	in, err := os.Open(src) //nolint:gosec // the page directory is the hub's own
	if err != nil {
		return "", 0, fmt.Errorf("publishpage: %w", err)
	}

	defer in.Close() //nolint:errcheck // read-only

	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", 0, fmt.Errorf("publishpage: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".publish-*")
	if err != nil {
		return "", 0, fmt.Errorf("publishpage: %w", err)
	}

	defer os.Remove(tmp.Name()) //nolint:errcheck // gone after the rename

	hash := sha256.New()

	n, err := io.Copy(io.MultiWriter(tmp, hash), in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		return "", 0, fmt.Errorf("publishpage: copy: %w", err)
	}

	if err := os.Rename(tmp.Name(), dst); err != nil {
		return "", 0, fmt.Errorf("publishpage: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), n, nil
}
