package eventlog_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

func TestStore_RoutesByProjectAndReplaysInSlugOrder(t *testing.T) {
	t.Parallel()

	s, err := eventlog.OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	at := time.Now()

	for _, p := range []string{"zeta", "alpha", "zeta"} {
		e := sample(eventlog.NewID(at), at)
		e.Project = p

		if err := s.Append(e); err != nil {
			t.Fatalf("Append %s: %v", p, err)
		}
	}

	projects, err := s.Projects()
	if err != nil || len(projects) != 2 || projects[0] != "alpha" || projects[1] != "zeta" {
		t.Fatalf("Projects = %v, %v", projects, err)
	}

	if seen := replayProjects(t, s); strings.Join(seen, ",") != "alpha,zeta,zeta" {
		t.Fatalf("replay order = %v", seen)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func replayProjects(t *testing.T, s *eventlog.Store) []string {
	t.Helper()

	var seen []string

	err := s.ReplayAll(func(e eventlog.Envelope) error {
		seen = append(seen, e.Project)

		return nil
	})
	if err != nil {
		t.Fatalf("ReplayAll: %v", err)
	}

	return seen
}

func TestStore_RefusesBadSlug(t *testing.T) {
	t.Parallel()

	s, _ := eventlog.OpenStore(t.TempDir())
	e := sample(eventlog.NewID(time.Now()), time.Now())
	e.Project = "../etc"

	if err := s.Append(e); !errors.Is(err, eventlog.ErrBadProject) {
		t.Fatalf("err = %v, want ErrBadProject", err)
	}
}
