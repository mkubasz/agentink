package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mkubasz/agentink/internal/platform/api"
)

func (c *cli) project(args []string) error {
	if len(args) == 0 {
		return usageError{"project needs a subcommand: add or list"}
	}

	switch args[0] {
	case "add":
		return c.projectAdd(args[1:])
	case "list":
		return c.projectList()
	default:
		return usageError{fmt.Sprintf("unknown project subcommand %q", args[0])}
	}
}

func (c *cli) projectAdd(args []string) error {
	fs := flag.NewFlagSet("ink project add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	slug := fs.String("slug", "", "slug (default: the directory name)")
	name := fs.String("name", "", "display name (default: the slug)")
	orch := fs.String("orchestrator", "", "herdr, rex, headless or none")

	positional, err := parseInterspersed(fs, args)
	if err != nil || len(positional) != 1 {
		return usageError{"project add takes one argument: the repository path"}
	}

	path := positional[0]
	if !filepath.IsAbs(path) {
		path = filepath.Join(c.cwd, path)
	}

	path = filepath.Clean(path)

	if *slug == "" {
		*slug = strings.ToLower(filepath.Base(path))
	}

	ctx, cancel := c.ctx()
	defer cancel()

	p, err := c.client.RegisterProject(ctx, api.RegisterProjectRequest{Slug: *slug, Name: *name, Path: path, Orchestrator: *orch})
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	return c.print(p, func() string {
		return fmt.Sprintf("registered %s (%s) at %s, specs in %s, orchestrator %s\n", p.Slug, p.Name, p.Path, p.SpecsDir, p.Orchestrator)
	})
}

func (c *cli) projectList() error {
	ctx, cancel := c.ctx()
	defer cancel()

	projects, err := c.client.Projects(ctx)
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	if projects == nil {
		projects = []api.Project{}
	}

	return c.print(projects, func() string {
		if len(projects) == 0 {
			return "no projects: ink project add <path>\n"
		}

		var b strings.Builder
		for _, p := range projects {
			fmt.Fprintf(&b, "%-16s %-10s %s\n", p.Slug, p.Orchestrator, p.Path)
		}

		return b.String()
	})
}
