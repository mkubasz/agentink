package ingestink

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Folder is a synced copy of the share (Syncthing on a Boox, brief §6.3) or
// the share itself: files that change there were written by a reader.
type Folder struct {
	// Device is credited with what appears here.
	Device string
	// Root holds Ink/<project>/… .
	Root string
}

// Watcher polls folders and hands changed files to the handler (spec 3.2b).
type Watcher struct {
	handler  *Handler
	folders  []Folder
	interval time.Duration
	logger   *slog.Logger

	mu   sync.Mutex
	seen map[string]stamp
}

type stamp struct {
	size  int64
	mtime time.Time
}

// NewWatcher returns a watcher; interval <= 0 means 5 seconds.
func NewWatcher(handler *Handler, folders []Folder, interval time.Duration, logger *slog.Logger) *Watcher {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Watcher{handler: handler, folders: folders, interval: interval, logger: logger, seen: map[string]stamp{}}
}

// Run polls until ctx is done. The first pass only records what is there,
// so a restart never re-ingests old files.
func (w *Watcher) Run(ctx context.Context) {
	w.Scan(ctx, false)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Scan(ctx, true)
		}
	}
}

// Scan walks every folder once; with ingest set, new or changed files are
// handed to the handler.
func (w *Watcher) Scan(ctx context.Context, ingest bool) {
	for _, f := range w.folders {
		w.scanFolder(ctx, f, ingest)
	}
}

func (w *Watcher) scanFolder(ctx context.Context, f Folder, ingest bool) {
	root := filepath.Join(f.Root, "Ink")

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil //nolint:nilerr // a vanished file or a folder we cannot read is not the watcher's failure
		}

		info, err := d.Info()
		if err != nil {
			return nil //nolint:nilerr // same
		}

		if !w.changed(path, info) || !ingest {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil //nolint:nilerr // outside the root; skip
		}

		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) != 2 {
			return nil
		}

		up := Upload{Project: parts[0], Device: f.Device, Rel: parts[1], Path: path, Size: info.Size(), Source: "folder", At: time.Now()}
		if err := w.handler.Ingest(ctx, up); err != nil && !isIgnored(err) {
			w.logger.Error("ingest from folder failed", "path", path, "err", err)
		}

		return nil
	})
}

// changed records the file's stamp and says whether it differs from the last one.
func (w *Watcher) changed(path string, info os.FileInfo) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	next := stamp{size: info.Size(), mtime: info.ModTime()}

	prev, ok := w.seen[path]
	w.seen[path] = next

	return !ok || prev != next
}

func isIgnored(err error) bool { return errors.Is(err, ErrIgnored) }
