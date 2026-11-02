//go:build e2e

// Package e2e runs the whole loop of the brief with stand-ins for the slices
// later phases add (spec task 1.7). A fake agent runs bin/ink against an
// in-process inkd; the stubs in stubs_test.go play render, publish, ingest,
// the worker, confirmation, syncspec and the adapter. Each phase replaces one
// stub with the real slice and this test keeps passing.
package e2e_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

func TestE2E_AskToDelivery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	socket := filepath.Join(dir, "d.sock")
	dataDir := filepath.Join(dir, "data")
	repo := filepath.Join(dir, "repos", "shop")
	specPath := filepath.Join(repo, "docs", "specs", "SHP-12-labels.md")

	copyFile(t, filepath.Join("testdata", "SHP-12-labels.md"), specPath)

	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not on PATH: the real renderer is part of the loop (ADR 0006)")
	}

	a, err := app.New(config.Config{DataDir: dataDir, SocketPath: socket}, "e2e", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	worker := fakeWorker(t)
	adapter := &fakeAdapter{}

	installStubs(t, a, worker.URL, adapter)
	serve(t, a, socket)

	ink := binary(t, "ink")

	env := append(os.Environ(), "AGENTINK_SOCKET="+socket, "HERDR_PANE_ID=w1:p3", "HERDR_WORKSPACE_ID=w1")

	mustRun(t, inkCmd(ink, "project", "add", repo, "--slug", "shop", "--orchestrator", "herdr"), env, repo)

	question := `{"kind":"decision","title":"Which printer does the label go to?",` +
		`"spec":{"path":"docs/specs/SHP-12-labels.md","anchor":"Q1"},` +
		`"options":[{"key":"A","label":"The thermal printer at the till","recommended":true,"cost":"one driver call"},` +
		`{"key":"B","label":"Any system printer","cost":"a layout per printer"}]}`

	ask := inkCmd(ink, "--json", "ask", "-")
	ask.Stdin = strings.NewReader(question)

	var asked struct {
		QuestionID string `json:"question_id"`
	}

	if err := json.Unmarshal(mustRun(t, ask, env, repo), &asked); err != nil || len(asked.QuestionID) != 26 {
		t.Fatalf("ask: %v", err)
	}

	// The stubs run synchronously on the bus, so the chain is complete when
	// ask returns; a real pipeline would take minutes and the agent would be
	// resumed by its orchestrator. Wait briefly anyway for the fake worker.
	waitFor(t, 5*time.Second, func() bool { return adapter.delivered() == 1 })

	assertSpecAnswered(t, specPath)
	assertPromptCarriesNoAnswer(t, adapter.lastPrompt(), asked.QuestionID)
	assertAgentReadsAnswer(t, ink, env, repo, asked.QuestionID)
	assertChain(t, filepath.Join(dataDir, "projects", "shop", "events.jsonl"))
}

// 1. The spec file holds the answer in the owner's words, the decision line
// and the log line, and the open answer line is gone.
func assertSpecAnswered(t *testing.T, specPath string) {
	t.Helper()

	spec := string(mustRead(t, specPath))

	for _, want := range []string{
		`**Answer:** A — owner, `,
		`"tylko przy kasie"`,
		"- D1 — Which printer does the label go to? → A, The thermal printer at the till — owner, ",
		"— clarify answered",
	} {
		if !strings.Contains(spec, want) {
			t.Errorf("spec lacks %q:\n%s", want, spec)
		}
	}

	if strings.Contains(spec, "**Answer:** —") {
		t.Error("the open answer line is still there")
	}
}

// 2. The adapter's prompt names the question and the command, and carries
// none of the recognized text (brief §7).
func assertPromptCarriesNoAnswer(t *testing.T, prompt, questionID string) {
	t.Helper()

	if !strings.Contains(prompt, questionID) || !strings.Contains(prompt, "ink answer "+questionID) {
		t.Errorf("prompt does not point at the answer: %q", prompt)
	}

	if strings.Contains(prompt, "tylko przy kasie") || strings.Contains(prompt, "thermal") {
		t.Errorf("prompt carries answer text: %q", prompt)
	}
}

// 3. The agent reads the answer with ink answer (exit 0, the words, status
// delivered) and the inbox is empty again.
func assertAgentReadsAnswer(t *testing.T, ink string, env []string, repo, questionID string) {
	t.Helper()

	out := mustRun(t, inkCmd(ink, "--json", "answer", questionID), env, repo)

	var answer struct {
		Status string `json:"status"`
		Words  string `json:"words"`
	}

	if err := json.Unmarshal(out, &answer); err != nil || answer.Status != "delivered" || !strings.Contains(answer.Words, "tylko przy kasie") {
		t.Fatalf("ink answer: %s (%v)", out, err)
	}

	if inbox := string(mustRun(t, inkCmd(ink, "inbox"), env, repo)); !strings.Contains(inbox, "nothing waits on you") {
		t.Errorf("inbox after delivery: %s", inbox)
	}
}

// 4. The log holds the whole chain, in order.
func assertChain(t *testing.T, logPath string) {
	t.Helper()

	want := "ProjectRegistered QuestionAsked PageRendered PagePublished InkReceived AnswerRecognized AnswerConfirmed SpecChanged AnswerDelivered"
	if got := strings.Join(eventTypes(t, logPath), " "); got != want {
		t.Errorf("event chain:\n got %s\nwant %s", got, want)
	}
}

// inkCmd runs our own ink binary, built by make build.
func inkCmd(ink string, args ...string) *exec.Cmd {
	return exec.CommandContext(context.Background(), ink, args...)
}

// serve runs the app on the socket for the test's life.
func serve(t *testing.T, a *app.App, socket string) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	ln, err := httpapi.ListenUnix(ctx, socket)
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() { _ = httpapi.Serve(ctx, ln, a.Handler()); close(done) }()

	t.Cleanup(func() { cancel(); <-done; _ = a.Close() })
}

// binary finds bin/<name> built by `make build`.
func binary(t *testing.T, name string) string {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("..", "bin", name))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s not built: run make build (%v)", name, err)
	}

	return path
}

func mustRun(t *testing.T, cmd *exec.Cmd, env []string, dir string) []byte {
	t.Helper()

	cmd.Env = env
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}

	return out
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path) //nolint:gosec // test fixtures and files under t.TempDir
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(to, mustRead(t, from), 0o600); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}

		time.Sleep(20 * time.Millisecond)
	}
}
