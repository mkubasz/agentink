package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/client"
	"github.com/mkubasz/agentink/internal/platform/config"
)

// Exit codes, documented in skills/ink/SKILL.md.
const (
	exitOK          = 0
	exitError       = 1
	exitUsage       = 2
	exitUnreachable = 3
)

const usage = `ink — talk to the AgentInk hub (inkd)

Usage:
  ink [--socket PATH] [--json] <command> [args]

Commands:
  ask <file|->            ask the owner a question (JSON, contracts/question.schema.json)
  answer <id>             where a question stands; the owner's words once confirmed
  inbox [--project SLUG]  list the open questions
  status                  is inkd up, how many projects and questions
  project add <path> [--slug S] [--name N] [--orchestrator herdr|rex|headless|none]
  project list
  version

Exit codes: 0 ok · 1 error · 2 usage · 3 inkd is not running · 4 no answer yet
`

type cli struct {
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	getenv  func(string) string
	cwd     string
	json    bool
	client  *client.Client
	timeout time.Duration
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string, cwd string) int {
	fs := flag.NewFlagSet("ink", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	socket := fs.String("socket", "", "inkd socket (default $"+config.EnvSocket+" or $XDG_RUNTIME_DIR/agentink.sock)")
	asJSON := fs.Bool("json", false, "machine-readable output")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	cfg, err := config.Resolve("", *socket, getenv)
	if err != nil {
		fmt.Fprintln(stderr, err)

		return exitError
	}

	c := &cli{
		stdin: stdin, stdout: stdout, stderr: stderr, getenv: getenv, cwd: cwd,
		json: *asJSON, client: client.New(cfg.SocketPath), timeout: 30 * time.Second,
	}

	return c.dispatch(rest[0], rest[1:])
}

func (c *cli) dispatch(cmd string, args []string) int {
	switch cmd {
	case "ask":
		return c.exit(c.ask(args))
	case "answer":
		return c.exit(c.answer(args))
	case "inbox":
		return c.exit(c.inbox(args))
	case "status":
		return c.exit(c.status())
	case "project":
		return c.exit(c.project(args))
	case "version":
		fmt.Fprintln(c.stdout, "ink", Version)

		return exitOK
	case "help", "-h", "--help":
		fmt.Fprint(c.stdout, usage)

		return exitOK
	default:
		fmt.Fprintf(c.stderr, "ink: unknown command %q\n\n%s", cmd, usage)

		return exitUsage
	}
}

// usageError is a wrong invocation: exit 2.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func (c *cli) exit(err error) int {
	var ue usageError

	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &ue):
		fmt.Fprintf(c.stderr, "ink: %s\n\n%s", ue.msg, usage)

		return exitUsage
	case errors.Is(err, client.ErrUnreachable):
		fmt.Fprintln(c.stderr, "ink: inkd is not running: start it with 'inkd' (then 'ink status')")

		return exitUnreachable
	case errors.Is(err, errPending):
		return exitPending
	default:
		fmt.Fprintln(c.stderr, "ink:", err)

		return exitError
	}
}

func (c *cli) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

func (c *cli) print(v any, text func() string) error {
	if c.json {
		enc := json.NewEncoder(c.stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(v) //nolint:wrapcheck // the writer is the terminal; nothing to add
	}

	fmt.Fprint(c.stdout, text())

	return nil
}

func (c *cli) status() error {
	ctx, cancel := c.ctx()
	defer cancel()

	st, err := c.client.Status(ctx)
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	return c.print(st, func() string {
		return fmt.Sprintf("inkd %s · data %s · render %s · %d projects · %d open questions\n", st.Version, st.DataDir, st.Render, st.Projects, st.OpenQuestions)
	})
}

func (c *cli) inbox(args []string) error {
	fs := flag.NewFlagSet("ink inbox", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	project := fs.String("project", "", "only this project")

	if positional, err := parseInterspersed(fs, args); err != nil || len(positional) != 0 {
		return usageError{"inbox takes no arguments"}
	}

	ctx, cancel := c.ctx()
	defer cancel()

	rows, err := c.client.Inbox(ctx, *project)
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	if rows == nil {
		rows = []api.InboxRow{}
	}

	return c.print(rows, func() string { return formatInbox(rows, time.Now()) })
}

func formatInbox(rows []api.InboxRow, now time.Time) string {
	if len(rows) == 0 {
		return "nothing waits on you\n"
	}

	var b strings.Builder

	for _, r := range rows {
		mark := " "
		if r.Critical {
			mark = "!"
		}

		fmt.Fprintf(&b, "%s %s  %-9s %-10s %-8s %6s  %s\n", mark, r.QuestionID, r.Kind, r.Project, r.Status, age(now.Sub(r.AskedAt)), r.Title)
	}

	return b.String()
}

func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
