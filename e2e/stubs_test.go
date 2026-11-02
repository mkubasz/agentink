//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// The stand-ins below subscribe to the bus in the order of the brief's event
// chain. Each one is replaced by a real slice in the phase named in its
// comment; the event it emits is the contract the slice must honour.

func installStubs(t *testing.T, a *app.App, workerURL string, adapter *fakeAdapter) {
	t.Helper()

	b := a.Bus()
	publish := func(ctx context.Context, e eventlog.Envelope) {
		if err := b.Publish(ctx, e); err != nil {
			t.Errorf("publish %s: %v", e.Type, err)
		}
	}

	// Phase 2, renderpage: the real slice (internal/pages/renderpage) renders
	// page.json, zones.json and page.pdf and emits PageRendered.

	// Phase 3, publishpage: the PDF lands in Ink/<project>/ on the WebDAV share.
	b.Subscribe("stub.publish", func(ctx context.Context, e eventlog.Envelope) error {
		publish(ctx, next(e, "PagePublished", e.Subject, map[string]any{
			"page_dir": pageDirOf(e), "path": "Ink/" + e.Project + "/" + e.Subject[5:] + ".pdf",
		}))

		return nil
	}, "PageRendered")

	// Phase 3, ingestink: the reader uploads the annotated file; here it is a
	// marker file written straight into the page directory.
	b.Subscribe("stub.ingest", func(ctx context.Context, e eventlog.Envelope) error {
		pageDir := pageDirOf(e)
		if pageDir == "" {
			return fmt.Errorf("stub.ingest: %s carries no page_dir", e.ID)
		}

		artifact := filepath.Join(pageDir, "answer.mark")
		content := []byte("fake ink for " + e.Subject)

		if err := os.WriteFile(artifact, content, 0o600); err != nil {
			return err
		}

		sum := sha256.Sum256(content)
		publish(ctx, next(e, "InkReceived", e.Subject, map[string]any{
			"page_dir": pageDir, "artifact": "answer.mark", "sha256": hex.EncodeToString(sum[:]), "device": "supernote-a5x",
		}))

		return nil
	}, "PagePublished")

	// Phase 4, recognizeanswer: inkd calls the worker's API (ADR 0007).
	b.Subscribe("stub.recognize", func(ctx context.Context, e eventlog.Envelope) error {
		var d struct {
			PageDir string `json:"page_dir"`
			Device  string `json:"device"`
			SHA256  string `json:"sha256"`
		}

		_ = json.Unmarshal(e.Data, &d)

		ocr, err := callWorker(workerURL, d.PageDir, d.Device)
		if err != nil {
			return err
		}

		answer := answerDoc(e.Subject[5:], d, ocr, "recognized")
		publish(ctx, next(e, "AnswerRecognized", e.Subject, answer))

		return nil
	}, "InkReceived")

	// Phase 4, confirmanswer: every text field is confident, so the words are
	// the owner's without a confirmation page.
	b.Subscribe("stub.confirm", func(ctx context.Context, e eventlog.Envelope) error {
		var answer map[string]any

		_ = json.Unmarshal(e.Data, &answer)

		answer["status"] = "confirmed"
		answer["words"] = "A, tylko przy kasie"
		answer["confirmed_at"] = e.At.UTC().Format("2006-01-02T15:04:05.000Z")

		qid, _ := answer["question_id"].(string)
		publish(ctx, next(e, "AnswerConfirmed", "question/"+qid, answer))

		return nil
	}, "AnswerRecognized")

	// Phase 5, syncspec: the answer in the owner's words, the decision line and
	// the `clarify answered` log line go into the spec file.
	b.Subscribe("stub.syncspec", stubSyncSpec(t, a, publish), "AnswerConfirmed")

	// Phase 5, deliveranswer: the adapter prompts the agent with the question
	// id and the command only — never with the recognized text (brief §7).
	b.Subscribe("stub.deliver", func(ctx context.Context, e eventlog.Envelope) error {
		var d struct {
			QuestionID string `json:"question_id"`
		}

		_ = json.Unmarshal(e.Data, &d)

		prompt := fmt.Sprintf("AgentInk: the owner answered question %s. Read it with: ink answer %s --json", d.QuestionID, d.QuestionID)
		adapter.deliver(prompt)
		publish(ctx, next(e, "AnswerDelivered", "question/"+d.QuestionID, map[string]any{"adapter": "fake-herdr", "target": "w1:p3"}))

		return nil
	}, "SpecChanged")
}

// stubSyncSpec stands in for the syncspec slice: the real one parses the spec
// flow's grammar; this one edits three known lines of the fixture.
func stubSyncSpec(t *testing.T, a *app.App, publish func(context.Context, eventlog.Envelope)) func(context.Context, eventlog.Envelope) error {
	t.Helper()

	return func(ctx context.Context, e eventlog.Envelope) error {
		specPath := specPathOf(t, a, e)
		if specPath == "" {
			return nil
		}

		today := e.At.UTC().Format("2006-01-02")
		spec := string(mustRead(t, specPath))
		spec = strings.Replace(spec, "**Answer:** —", fmt.Sprintf(`**Answer:** A — owner, %s: "tylko przy kasie"`, today), 1)
		spec = strings.Replace(spec, "## Decisions\n", fmt.Sprintf("## Decisions\n- D1 — Which printer does the label go to? → A, The thermal printer at the till — owner, %s\n", today), 1)
		spec = strings.TrimRight(spec, "\n") + fmt.Sprintf("\n- %s — clarify answered\n", e.At.UTC().Format("2006-01-02 15:04"))

		if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
			return err
		}

		publish(ctx, next(e, "SpecChanged", "spec/"+filepath.Base(specPath), map[string]any{"path": specPath, "question_id": questionID(e.Subject)}))

		return nil
	}
}

// next builds the envelope that follows e in the chain.
func next(e eventlog.Envelope, typ, subject string, data any) eventlog.Envelope {
	raw, _ := json.Marshal(data)
	at := e.At.Add(time.Millisecond)
	corr := e.CorrelationID

	if corr == "" {
		corr = e.ID
	}

	return eventlog.Envelope{
		ID: eventlog.NewID(at), Type: typ, At: at, Project: e.Project, Subject: subject, Rev: 1,
		Data: raw, CausationID: e.ID, CorrelationID: corr,
	}
}

func questionID(subject string) string { return strings.TrimPrefix(subject, "question/") }

func pageDirOf(e eventlog.Envelope) string {
	var d struct {
		PageDir string `json:"page_dir"`
	}

	_ = json.Unmarshal(e.Data, &d)

	return d.PageDir
}

// specPathOf finds the spec file the question named, under the project's path.
func specPathOf(t *testing.T, a *app.App, e eventlog.Envelope) string {
	t.Helper()

	var (
		questionRel string
		projectPath string
	)

	_, _ = eventlog.Read(filepath.Join(a.DataDir(), "projects", e.Project, "events.jsonl"), func(x eventlog.Envelope) error {
		switch x.Type {
		case "ProjectRegistered":
			var d struct {
				Path string `json:"path"`
			}

			_ = json.Unmarshal(x.Data, &d)
			projectPath = d.Path
		case "QuestionAsked":
			if x.Subject == e.Subject {
				var d struct {
					Question struct {
						Spec struct {
							Path string `json:"path"`
						} `json:"spec"`
					} `json:"question"`
				}

				_ = json.Unmarshal(x.Data, &d)
				questionRel = d.Question.Spec.Path
			}
		}

		return nil
	})

	if questionRel == "" || projectPath == "" {
		return ""
	}

	return filepath.Join(projectPath, questionRel)
}

func answerDoc(pageID string, in struct {
	PageDir string `json:"page_dir"`
	Device  string `json:"device"`
	SHA256  string `json:"sha256"`
}, ocr map[string]any, status string,
) map[string]any {
	page := readJSON(in.PageDir + "/page.json")
	fields := map[string]any{}

	ocrFields, _ := ocr["fields"].(map[string]any)
	for id, f := range ocrFields {
		ff, _ := f.(map[string]any)
		fields[id] = map[string]any{"kind": ff["kind"], "value": ff["value"], "confidence": ff["confidence"]}
	}

	return map[string]any{
		"page_id": pageID, "rev": 1, "question_id": page["question_id"], "question_rev": 1, "status": status,
		"source": map[string]any{"device": in.Device, "artifact": "answer.mark", "sha256": in.SHA256},
		"fields": fields,
	}
}

// fakeWorker speaks the ADR 0007 API with the stub engine: every choice is A,
// every text field reads "tylko przy kasie" with high confidence.
func fakeWorker(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /v1/recognize", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PageDir string `json:"page_dir"`
			Device  string `json:"device"`
			Engine  string `json:"engine"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		page := readJSON(req.PageDir + "/page.json")

		pageID, ok := page["id"].(string)
		if !ok {
			http.Error(w, "no page.json in "+req.PageDir, http.StatusBadRequest)

			return
		}

		ocr := map[string]any{
			"page_id": pageID, "rev": 1,
			"engine":    map[string]any{"name": "stub", "version": "0.0.1"},
			"alignment": map[string]any{"ok": true, "qr_payload": "INK:" + pageID + ":1", "residual_px": 0.5},
			"fields": map[string]any{
				"choice":    map[string]any{"kind": "choice", "value": "A", "confidence": 0.99, "ink_ratio": 0.4},
				"otherwise": map[string]any{"kind": "text", "value": "tylko przy kasie", "raw": "tylko przy kasie", "confidence": 0.95},
				"sketch":    map[string]any{"kind": "sketch", "value": nil, "confidence": 1},
			},
		}

		body, _ := json.Marshal(ocr)
		if err := contracts.Validate(contracts.OCR, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		_ = os.WriteFile(filepath.Join(req.PageDir, "ocr.json"), body, 0o600)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func callWorker(url, pageDir, device string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]string{"page_dir": pageDir, "device": device, "engine": "stub"})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, url+"/v1/recognize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("worker: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck // read-only response body

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worker: %d %s", resp.StatusCode, raw)
	}

	var ocr map[string]any
	if err := json.Unmarshal(raw, &ocr); err != nil {
		return nil, fmt.Errorf("worker: %w", err)
	}

	return ocr, nil
}

// fakeAdapter records what a real herdr/rex/headless adapter would send.
type fakeAdapter struct {
	mu      sync.Mutex
	prompts []string
}

func (f *fakeAdapter) deliver(prompt string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.prompts = append(f.prompts, prompt)
}

func (f *fakeAdapter) delivered() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.prompts)
}

func (f *fakeAdapter) lastPrompt() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.prompts) == 0 {
		return ""
	}

	return f.prompts[len(f.prompts)-1]
}

func readJSON(path string) map[string]any {
	raw, _ := os.ReadFile(path) //nolint:gosec // files under t.TempDir

	var v map[string]any

	_ = json.Unmarshal(raw, &v)

	return v
}

func eventTypes(t *testing.T, logPath string) []string {
	t.Helper()

	var types []string

	if _, err := eventlog.Read(logPath, func(e eventlog.Envelope) error {
		types = append(types, e.Type)

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	return types
}
