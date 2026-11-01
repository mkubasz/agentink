package eventlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

// ErrBadProject is returned for a project slug the store will not use as a
// directory name.
var ErrBadProject = errors.New("eventlog: bad project slug")

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Store holds one Log per project under <dir>/projects/<slug>/events.jsonl
// and routes every envelope to its project's log (ADR 0005).
type Store struct {
	dir  string
	mu   sync.Mutex
	logs map[string]*Log
}

// OpenStore opens the store rooted at dir, creating it when missing.
func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "projects"), 0o700); err != nil {
		return nil, fmt.Errorf("eventlog: open store %s: %w", dir, err)
	}

	return &Store{dir: dir, logs: map[string]*Log{}}, nil
}

// Dir is the store's root directory.
func (s *Store) Dir() string { return s.dir }

// LogPath is where a project's events live.
func (s *Store) LogPath(project string) string {
	return filepath.Join(s.dir, "projects", project, "events.jsonl")
}

// Append routes e to its project's log, opening the log on first use.
func (s *Store) Append(e Envelope) error {
	if err := e.Validate(); err != nil {
		return err
	}

	l, err := s.log(e.Project)
	if err != nil {
		return err
	}

	return l.Append(e)
}

func (s *Store) log(project string) (*Log, error) {
	if !slugRe.MatchString(project) {
		return nil, fmt.Errorf("%w: %q", ErrBadProject, project)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if l, ok := s.logs[project]; ok {
		return l, nil
	}

	l, err := Open(s.LogPath(project))
	if err != nil {
		return nil, err
	}

	s.logs[project] = l

	return l, nil
}

// Projects lists the slugs that have a log on disk, sorted.
func (s *Store) Projects() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, "projects"))
	if err != nil {
		return nil, fmt.Errorf("eventlog: list projects: %w", err)
	}

	var out []string

	for _, en := range entries {
		if !en.IsDir() || !slugRe.MatchString(en.Name()) {
			continue
		}

		if _, err := os.Stat(s.LogPath(en.Name())); err == nil {
			out = append(out, en.Name())
		}
	}

	sort.Strings(out)

	return out, nil
}

// ReplayAll reads every project's log, project by project in slug order,
// and hands each envelope to fn. It is how projections are rebuilt at start.
func (s *Store) ReplayAll(fn func(Envelope) error) error {
	projects, err := s.Projects()
	if err != nil {
		return err
	}

	for _, p := range projects {
		if _, err := Read(s.LogPath(p), fn); err != nil {
			return err
		}
	}

	return nil
}

// Close closes every open log.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	for p, l := range s.logs {
		if err := l.Close(); err != nil {
			errs = append(errs, err)
		}

		delete(s.logs, p)
	}

	return errors.Join(errs...)
}
