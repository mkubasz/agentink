package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mkubasz/agentink/internal/platform/api"
)

// ask reads a question (a file or stdin), adds the agent's address from the
// environment when the question has none, and submits it. It returns at
// once: the answer comes back through the orchestrator (brief §6.1).
func (c *cli) ask(args []string) error {
	fs := flag.NewFlagSet("ink ask", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	project := fs.String("project", "", "project slug (default: the one containing the working directory)")
	wait := fs.Duration("wait", 0, "wait up to this long for the answer (not yet implemented; the default is asynchronous)")

	positional, err := parseInterspersed(fs, args)
	if err != nil || len(positional) != 1 {
		return usageError{"ask takes one argument: a JSON file, or - for stdin"}
	}

	if *wait != 0 {
		return usageError{"--wait is not implemented yet: ink ask is asynchronous, end your turn and read the answer when it comes"}
	}

	raw, err := c.readQuestion(positional[0])
	if err != nil {
		return err
	}

	raw, err = withAgent(raw, c.getenv)
	if err != nil {
		return err
	}

	ctx, cancel := c.ctx()
	defer cancel()

	res, err := c.client.Ask(ctx, api.AskRequest{Project: *project, Cwd: c.cwd, Question: raw})
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	return c.print(res, func() string {
		return fmt.Sprintf("asked %s in %s (rev %d)\nThe answer arrives through your orchestrator; do not wait or poll. When it comes: ink answer %s --json\n",
			res.QuestionID, res.Project, res.Rev, res.QuestionID)
	})
}

func (c *cli) readQuestion(arg string) (json.RawMessage, error) {
	var (
		raw []byte
		err error
	)

	if arg == "-" {
		raw, err = io.ReadAll(io.LimitReader(c.stdin, 1<<20))
	} else {
		raw, err = os.ReadFile(filepath.Clean(arg))
	}

	if err != nil {
		return nil, fmt.Errorf("ink ask: read question: %w", err)
	}

	if !json.Valid(raw) {
		return nil, fmt.Errorf("ink ask: %s is not JSON", arg)
	}

	return raw, nil
}

// withAgent adds `agent` from HERDR_* or REX_* when the question has none,
// so the hub knows where the answer goes (brief §6.1).
func withAgent(raw json.RawMessage, getenv func(string) string) (json.RawMessage, error) {
	var q map[string]json.RawMessage
	if err := json.Unmarshal(raw, &q); err != nil {
		return nil, fmt.Errorf("ink ask: question must be a JSON object: %w", err)
	}

	if _, has := q["agent"]; has {
		return raw, nil
	}

	agent := agentFromEnv(getenv)
	if agent == nil {
		return raw, nil
	}

	b, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("ink ask: agent: %w", err)
	}

	q["agent"] = b

	out, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("ink ask: question: %w", err)
	}

	return out, nil
}

func agentFromEnv(getenv func(string) string) map[string]any {
	if pane := getenv("HERDR_PANE_ID"); pane != "" {
		return map[string]any{
			"orchestrator": "herdr",
			"herdr":        compact(map[string]string{"pane_id": pane, "workspace_id": getenv("HERDR_WORKSPACE_ID"), "tab_id": getenv("HERDR_TAB_ID")}),
		}
	}

	if block, session := getenv("REX_BLOCK"), getenv("REX_SESSION"); block != "" || session != "" {
		return map[string]any{
			"orchestrator": "rex",
			"rex":          compact(map[string]string{"session": session, "block": block, "server": getenv("REX_SERVER")}),
		}
	}

	return nil
}

func compact(m map[string]string) map[string]string {
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			delete(m, k)
		}
	}

	return m
}
