// Package ingestink is the slice that turns what a reader sent back into
// InkReceived: an upload on the WebDAV share or a changed file in a synced
// folder, matched to its page by the id at the front of the file name, taken
// in once per bytes, copied into the page directory (spec task 3.2).
package ingestink

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Upload is one file a reader sent, from the share or a watched folder.
type Upload struct {
	Project string
	Device  string
	// Rel is the path below Ink/<project>/.
	Rel  string
	Path string
	Size int64
	// Source is "webdav" or "folder".
	Source string
	At     time.Time
}

// Received is the payload of InkReceived.
type Received struct {
	PageID      string `json:"page_id"`
	Rev         int    `json:"rev"`
	QuestionID  string `json:"question_id"`
	QuestionRev int    `json:"question_rev"`
	PageDir     string `json:"page_dir"`
	Artifact    string `json:"artifact"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Device      string `json:"device"`
	Source      string `json:"source"`
}

// Rejected is the payload of IngestRejected.
type Rejected struct {
	Project string `json:"project"`
	Rel     string `json:"rel"`
	Device  string `json:"device"`
	Reason  string `json:"reason"`
}

// ErrIgnored marks an upload that is neither an answer nor a mistake: the
// sync echoing the original, or bytes already taken in.
var ErrIgnored = errors.New("ingestink: ignored")

var (
	pageIDRe = regexp.MustCompile(`^([0-9A-HJKMNP-TV-Z]{26})`)
	// artifacts the worker can read: an annotated PDF, a Supernote .mark, a
	// Supernote .note, a PNG.
	artifactExt = map[string]bool{".pdf": true, ".mark": true, ".note": true, ".png": true}
)

// Handler takes uploads in.
type Handler struct {
	bus    Publisher
	pages  *domain.Pages
	logger *slog.Logger
	now    func() time.Time
}

// New wires the slice.
func New(bus Publisher, pages *domain.Pages, logger *slog.Logger, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{bus: bus, pages: pages, logger: logger, now: now}
}

// Ingest decides what an upload is and emits InkReceived or IngestRejected.
// It returns ErrIgnored for an echo or a duplicate, nil when an event was
// emitted, and another error only when the hub itself failed.
func (h *Handler) Ingest(ctx context.Context, up Upload) error {
	page, reason := h.resolve(up)
	if reason != "" {
		return h.reject(ctx, up, reason)
	}

	sum, size, err := hashFile(up.Path)
	if err != nil {
		return err
	}

	if size == 0 {
		return h.reject(ctx, up, "empty file")
	}

	if sum == page.OriginalSHA256 {
		h.logger.Debug("ingest: the original page came back unchanged", "page", page.PageID, "device", up.Device)

		return ErrIgnored
	}

	if h.pages.Received(page.PageID, page.Rev, sum) {
		h.logger.Debug("ingest: same bytes already received", "page", page.PageID, "device", up.Device)

		return ErrIgnored
	}

	artifact := "answer" + strings.ToLower(filepath.Ext(up.Rel))
	if err := copyFile(up.Path, filepath.Join(page.PageDir, artifact)); err != nil {
		return err
	}

	data, err := json.Marshal(Received{
		PageID: page.PageID, Rev: page.Rev, QuestionID: page.QuestionID, QuestionRev: page.QuestionRev,
		PageDir: page.PageDir, Artifact: artifact, SHA256: sum, Size: size, Device: up.Device, Source: up.Source,
	})
	if err != nil {
		return fmt.Errorf("ingestink: marshal: %w", err)
	}

	at := h.now()

	return h.bus.Publish(ctx, eventlog.Envelope{
		ID: eventlog.NewID(at), Type: domain.EventInkReceived, At: at, Project: page.Project,
		Subject: "page/" + page.PageID, Rev: page.Rev, Data: data,
	})
}

// resolve finds the page an upload belongs to, or says why it cannot.
func (h *Handler) resolve(up Upload) (domain.PublishedPage, string) {
	base := filepath.Base(up.Rel)
	if strings.HasPrefix(base, ".") {
		return domain.PublishedPage{}, "hidden or temporary file"
	}

	if !artifactExt[strings.ToLower(filepath.Ext(base))] {
		return domain.PublishedPage{}, "not a pdf, mark, note or png"
	}

	m := pageIDRe.FindStringSubmatch(base)
	if m == nil {
		return domain.PublishedPage{}, "file name does not start with a page id"
	}

	page, ok := h.pages.Get(m[1])
	if !ok {
		return domain.PublishedPage{}, "no page with id " + m[1]
	}

	if page.Project != up.Project {
		return domain.PublishedPage{}, "page belongs to another project"
	}

	return page, ""
}

func (h *Handler) reject(ctx context.Context, up Upload, reason string) error {
	data, err := json.Marshal(Rejected{Project: up.Project, Rel: up.Rel, Device: up.Device, Reason: reason})
	if err != nil {
		return fmt.Errorf("ingestink: marshal: %w", err)
	}

	at := h.now()

	return h.bus.Publish(ctx, eventlog.Envelope{
		ID: eventlog.NewID(at), Type: domain.EventIngestRejected, At: at, Project: up.Project,
		Subject: "device/" + firstNonEmpty(up.Device, "unknown"), Rev: 0, Data: data,
	})
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path) //nolint:gosec // the share directory is the hub's own
	if err != nil {
		return "", 0, fmt.Errorf("ingestink: %w", err)
	}

	defer f.Close() //nolint:errcheck // read-only

	hash := sha256.New()

	n, err := io.Copy(hash, f)
	if err != nil {
		return "", 0, fmt.Errorf("ingestink: hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), n, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // the share directory is the hub's own
	if err != nil {
		return fmt.Errorf("ingestink: %w", err)
	}

	defer in.Close() //nolint:errcheck // read-only

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".ingest-*")
	if err != nil {
		return fmt.Errorf("ingestink: %w", err)
	}

	defer os.Remove(tmp.Name()) //nolint:errcheck // gone after the rename

	_, err = io.Copy(tmp, in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		return fmt.Errorf("ingestink: copy: %w", err)
	}

	if err := os.Rename(tmp.Name(), dst); err != nil {
		return fmt.Errorf("ingestink: %w", err)
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
