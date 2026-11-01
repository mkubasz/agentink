package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// daemon starts inkd's app on a Unix socket under a short temp dir.
func daemon(t *testing.T) (socket, dataDir string) {
	t.Helper()

	dir := t.TempDir()
	dataDir = filepath.Join(dir, "data")
	socket = filepath.Join(dir, "d.sock")

	a, err := app.New(config.Config{DataDir: dataDir, SocketPath: socket}, "test", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ln, err := httpapi.ListenUnix(ctx, socket)
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() { _ = httpapi.Serve(ctx, ln, a.Handler()); close(done) }()

	t.Cleanup(func() { cancel(); <-done; _ = a.Close() })

	return socket, dataDir
}

type result struct {
	code   int
	stdout string
	stderr string
}

func ink(socket string, env map[string]string, cwd, stdin string, args ...string) result {
	var out, errb bytes.Buffer

	all := append([]string{"--socket", socket}, args...)
	code := run(all, strings.NewReader(stdin), &out, &errb, func(k string) string { return env[k] }, cwd)

	return result{code: code, stdout: out.String(), stderr: errb.String()}
}

func TestInk_UnreachableDaemonExits3(t *testing.T) {
	t.Parallel()

	r := ink("/tmp/ink-nope.sock", nil, "/", "", "status")
	if r.code != exitUnreachable || !strings.Contains(r.stderr, "inkd is not running") {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestInk_UsageExits2(t *testing.T) {
	t.Parallel()

	if r := ink("/tmp/x.sock", nil, "/", "", "frobnicate"); r.code != exitUsage {
		t.Fatalf("unknown command: %+v", r)
	}

	if r := ink("/tmp/x.sock", nil, "/", "", "ask"); r.code != exitUsage {
		t.Fatalf("ask without a file: %+v", r)
	}

	if r := ink("/tmp/x.sock", nil, "/", "{}", "ask", "--wait", "5m", "-"); r.code != exitUsage || !strings.Contains(r.stderr, "asynchronous") {
		t.Fatalf("--wait: %+v", r)
	}
}

// seedShop registers the shop project and asks one decision from inside it
// with a herdr pane in the environment; it returns the question id.
func seedShop(t *testing.T, socket, repo string) string {
	t.Helper()

	if r := ink(socket, nil, repo, "", "project", "add", repo, "--orchestrator", "herdr"); r.code != exitOK || !strings.Contains(r.stdout, "registered shop") {
		t.Fatalf("project add: %+v", r)
	}

	question := `{"kind":"decision","title":"Which printer?","options":[{"key":"A","label":"thermal","recommended":true},{"key":"B","label":"laser"}]}`
	env := map[string]string{"HERDR_PANE_ID": "w1:p3", "HERDR_WORKSPACE_ID": "w1"}

	r := ink(socket, env, filepath.Join(repo, "internal"), question, "--json", "ask", "-")
	if r.code != exitOK {
		t.Fatalf("ask: %+v", r)
	}

	var res struct {
		QuestionID string `json:"question_id"`
		Project    string `json:"project"`
	}

	if err := json.Unmarshal([]byte(r.stdout), &res); err != nil || res.Project != "shop" || len(res.QuestionID) != 26 {
		t.Fatalf("ask json: %q %v", r.stdout, err)
	}

	return res.QuestionID
}

func TestInk_ProjectAskInboxRoundTrip(t *testing.T) {
	t.Parallel()

	socket, dataDir := daemon(t)
	repo := filepath.Join(filepath.Dir(dataDir), "repos", "shop")
	_ = os.MkdirAll(filepath.Join(repo, "internal"), 0o750)

	questionID := seedShop(t, socket, repo)

	if got := loggedAgents(t, dataDir); len(got) != 1 || got[0] != "herdr:w1:p3" {
		t.Fatalf("agent in log = %v", got)
	}

	r := ink(socket, nil, "/", "", "inbox")
	if r.code != exitOK || !strings.Contains(r.stdout, "Which printer?") || !strings.Contains(r.stdout, questionID) {
		t.Fatalf("inbox: %+v", r)
	}

	r = ink(socket, nil, "/", "", "status")
	if r.code != exitOK || !strings.Contains(r.stdout, "1 projects · 1 open questions") {
		t.Fatalf("status: %+v", r)
	}

	r = ink(socket, nil, "/", `{"kind":"decision","title":"no options"}`, "ask", "--project", "shop", "-")
	if r.code != exitError || !strings.Contains(r.stderr, "does not match the schema") {
		t.Fatalf("invalid question: %+v", r)
	}
}

// loggedAgents reads the agent address of every QuestionAsked in the shop log.
func loggedAgents(t *testing.T, dataDir string) []string {
	t.Helper()

	var agents []string

	_, err := eventlog.Read(filepath.Join(dataDir, "projects", "shop", "events.jsonl"), func(e eventlog.Envelope) error {
		if e.Type != "QuestionAsked" {
			return nil
		}

		var d struct {
			Question struct {
				Agent struct {
					Orchestrator string `json:"orchestrator"`
					Herdr        struct {
						PaneID string `json:"pane_id"`
					} `json:"herdr"`
				} `json:"agent"`
			} `json:"question"`
		}

		_ = json.Unmarshal(e.Data, &d)
		agents = append(agents, d.Question.Agent.Orchestrator+":"+d.Question.Agent.Herdr.PaneID)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return agents
}

func TestAgentFromEnv(t *testing.T) {
	t.Parallel()

	rex := agentFromEnv(func(k string) string {
		return map[string]string{"REX_SESSION": "session:1", "REX_BLOCK": "block:2", "REX_SERVER": "127.0.0.1:7"}[k]
	})

	fields, ok := rex["rex"].(map[string]string)
	if rex["orchestrator"] != "rex" || !ok || fields["block"] != "block:2" {
		t.Fatalf("rex = %v", rex)
	}

	if none := agentFromEnv(func(string) string { return "" }); none != nil {
		t.Fatalf("no env must give no agent, got %v", none)
	}

	kept, err := withAgent(json.RawMessage(`{"kind":"brief","title":"x","agent":{"orchestrator":"none"}}`), func(string) string { return "w1:p1" })
	if err != nil || !strings.Contains(string(kept), `"orchestrator":"none"`) {
		t.Fatalf("an explicit agent must win: %s %v", kept, err)
	}
}
