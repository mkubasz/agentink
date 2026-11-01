// Package domain is the projects context: a project is a repository the hub
// knows, with its specs folder and its orchestrator binding (ADR 0001).
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// EventRegistered is the type of the event that creates a project.
const EventRegistered = "ProjectRegistered"

// Orchestrators the agents context knows how to deliver to.
var Orchestrators = []string{"herdr", "rex", "headless", "none"}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Validation errors.
var (
	ErrBadSlug         = errors.New("projects: slug must be lowercase letters, digits and hyphens")
	ErrBadPath         = errors.New("projects: path must be absolute")
	ErrBadOrchestrator = errors.New("projects: unknown orchestrator")
	ErrNotFound        = errors.New("projects: no such project")
)

// Registered is the payload of ProjectRegistered.
type Registered struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	SpecsDir     string `json:"specs_dir"`
	Orchestrator string `json:"orchestrator"`
}

// Validate normalises and checks a registration.
func (r *Registered) Validate() error {
	if !slugRe.MatchString(r.Slug) {
		return fmt.Errorf("%w: %q", ErrBadSlug, r.Slug)
	}

	if !filepath.IsAbs(r.Path) {
		return fmt.Errorf("%w: %q", ErrBadPath, r.Path)
	}

	r.Path = filepath.Clean(r.Path)

	if r.Name == "" {
		r.Name = r.Slug
	}

	if r.SpecsDir == "" {
		r.SpecsDir = "docs/specs"
	}

	if r.Orchestrator == "" {
		r.Orchestrator = "none"
	}

	for _, o := range Orchestrators {
		if o == r.Orchestrator {
			return nil
		}
	}

	return fmt.Errorf("%w: %q", ErrBadOrchestrator, r.Orchestrator)
}

// Registry is the projection of every ProjectRegistered event, rebuilt from
// the log at start. It is safe for concurrent use.
type Registry struct {
	mu   sync.RWMutex
	byID map[string]api.Project
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{byID: map[string]api.Project{}} }

// Apply folds one event; events of other types are ignored.
func (r *Registry) Apply(e eventlog.Envelope) error {
	if e.Type != EventRegistered {
		return nil
	}

	var d Registered
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return fmt.Errorf("projects: bad %s data in %s: %w", e.Type, e.ID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[d.Slug] = api.Project{
		Slug: d.Slug, Name: d.Name, Path: d.Path, SpecsDir: d.SpecsDir,
		Orchestrator: d.Orchestrator, RegisteredAt: e.At,
	}

	return nil
}

// Get returns one project.
func (r *Registry) Get(slug string) (api.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.byID[slug]
	if !ok {
		return api.Project{}, fmt.Errorf("%w: %q", ErrNotFound, slug)
	}

	return p, nil
}

// List returns every project, sorted by slug.
func (r *Registry) List() []api.Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]api.Project, 0, len(r.byID))
	for _, p := range r.byID {
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

	return out
}

// ByCwd returns the project whose path is the longest prefix of cwd: how an
// agent's working directory names its project without a flag.
func (r *Registry) ByCwd(cwd string) (api.Project, error) {
	cwd = filepath.Clean(cwd)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var best api.Project

	for _, p := range r.byID {
		if (cwd == p.Path || strings.HasPrefix(cwd, p.Path+string(filepath.Separator))) && len(p.Path) > len(best.Path) {
			best = p
		}
	}

	if best.Slug == "" {
		return api.Project{}, fmt.Errorf("%w: no project contains %s", ErrNotFound, cwd)
	}

	return best, nil
}
