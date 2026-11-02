package renderpage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/pages/domain"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Events this slice reacts to and emits.
const (
	EventAsked    = "QuestionAsked"
	EventRendered = "PageRendered"
	EventFailed   = "RenderFailed"
)

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Rendered is the payload of PageRendered.
type Rendered struct {
	QuestionID  string `json:"question_id"`
	QuestionRev int    `json:"question_rev"`
	PageID      string `json:"page_id"`
	PageDir     string `json:"page_dir"`
	PDF         string `json:"pdf"`
	Zones       string `json:"zones"`
	Device      string `json:"device"`
}

// Failed is the payload of RenderFailed.
type Failed struct {
	QuestionID string `json:"question_id"`
	Reason     string `json:"reason"`
}

// Handler renders pages.
type Handler struct {
	bus     Publisher
	typst   *Typst
	dataDir string
	device  domain.Device
	now     func() time.Time
}

// New wires the slice. now may be nil for the wall clock.
func New(bus Publisher, typst *Typst, dataDir string, device domain.Device, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	return &Handler{bus: bus, typst: typst, dataDir: dataDir, device: device, now: now}
}

// Handle renders the page of a QuestionAsked event. Other events are ignored.
// A render error becomes RenderFailed; the error is also returned for the log.
func (h *Handler) Handle(ctx context.Context, e eventlog.Envelope) error {
	if e.Type != EventAsked {
		return nil
	}

	var asked struct {
		Question json.RawMessage `json:"question"`
	}

	if err := json.Unmarshal(e.Data, &asked); err != nil {
		return fmt.Errorf("renderpage: %s: %w", e.ID, err)
	}

	questionID := questionIDOf(e.Subject)

	rendered, err := h.render(ctx, e, asked.Question, questionID)
	if err != nil {
		data, _ := json.Marshal(Failed{QuestionID: questionID, Reason: err.Error()})
		_ = h.bus.Publish(ctx, h.next(e, EventFailed, e.Subject, data))

		return err
	}

	data, err := json.Marshal(rendered)
	if err != nil {
		return fmt.Errorf("renderpage: marshal: %w", err)
	}

	return h.bus.Publish(ctx, h.next(e, EventRendered, "page/"+rendered.PageID, data))
}

// Render builds the page files for a question into pageDir and returns the
// PageRendered payload. It is the testable core of Handle.
func (h *Handler) Render(ctx context.Context, question json.RawMessage, questionID string, questionRev int, pageID, pageDir string) (Rendered, error) {
	page, content, err := domain.FromQuestion(question, pageID, questionID, questionRev, h.device, h.now())
	if err != nil {
		return Rendered{}, err
	}

	if page.QR.Modules, err = modules(page.QR.Payload); err != nil {
		return Rendered{}, err
	}

	if err := os.MkdirAll(pageDir, 0o700); err != nil {
		return Rendered{}, fmt.Errorf("renderpage: %w", err)
	}

	in, err := h.inputs(page, content)
	if err != nil {
		return Rendered{}, err
	}

	if err := writeValidated(filepath.Join(pageDir, "page.json"), contracts.Page, stripModules(in.Page)); err != nil {
		return Rendered{}, err
	}

	if err := os.WriteFile(filepath.Join(pageDir, "content.json"), in.Content, 0o600); err != nil {
		return Rendered{}, fmt.Errorf("renderpage: %w", err)
	}

	if err := h.typst.Compile(ctx, page.Kind, in, filepath.Join(pageDir, "page.pdf")); err != nil {
		return Rendered{}, err
	}

	zones, err := h.zonesDocument(ctx, page, in)
	if err != nil {
		return Rendered{}, err
	}

	if err := writeValidated(filepath.Join(pageDir, "zones.json"), contracts.Zones, zones); err != nil {
		return Rendered{}, err
	}

	return Rendered{
		QuestionID: questionID, QuestionRev: questionRev, PageID: pageID, PageDir: pageDir,
		PDF: "page.pdf", Zones: "zones.json", Device: h.device.Name,
	}, nil
}

// RenderPNG renders the same page to a PNG at ppi, for golden tests and previews.
func (h *Handler) RenderPNG(ctx context.Context, question json.RawMessage, questionID string, questionRev int, pageID, out string, ppi int) error {
	page, content, err := domain.FromQuestion(question, pageID, questionID, questionRev, h.device, h.now())
	if err != nil {
		return err
	}

	if page.QR.Modules, err = modules(page.QR.Payload); err != nil {
		return err
	}

	in, err := h.inputs(page, content)
	if err != nil {
		return err
	}

	return h.typst.Compile(ctx, page.Kind, in, out, "--format", "png", "--ppi", strconv.Itoa(ppi))
}

func (h *Handler) render(ctx context.Context, e eventlog.Envelope, question json.RawMessage, questionID string) (Rendered, error) {
	pageID := eventlog.NewID(h.now())
	pageDir := filepath.Join(h.dataDir, "projects", e.Project, "pages", pageID)

	return h.Render(ctx, question, questionID, e.Rev, pageID, pageDir)
}

func (h *Handler) inputs(page domain.Page, content domain.Content) (Inputs, error) {
	pageJSON, err := json.Marshal(page)
	if err != nil {
		return Inputs{}, fmt.Errorf("renderpage: marshal page: %w", err)
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return Inputs{}, fmt.Errorf("renderpage: marshal content: %w", err)
	}

	w, hh := h.device.SizeMM()

	sizeJSON, err := json.Marshal(map[string]float64{"w": w, "h": hh})
	if err != nil {
		return Inputs{}, fmt.Errorf("renderpage: marshal size: %w", err)
	}

	return Inputs{Page: pageJSON, Content: contentJSON, Size: sizeJSON}, nil
}

func (h *Handler) zonesDocument(ctx context.Context, page domain.Page, in Inputs) ([]byte, error) {
	raw, err := h.typst.Zones(ctx, page.Kind, in)
	if err != nil {
		return nil, err
	}

	zones := make([]map[string]any, 0, len(raw))

	for _, r := range raw {
		var z map[string]any
		if err := json.Unmarshal(r, &z); err != nil {
			return nil, fmt.Errorf("renderpage: zone: %w", err)
		}

		for _, k := range []string{"x", "y", "w", "h"} {
			if v, ok := z[k].(float64); ok {
				z[k] = math.Round(v*100) / 100
			}
		}

		zones = append(zones, z)
	}

	w, hh := h.device.SizeMM()

	doc, err := json.Marshal(map[string]any{
		"page_id": page.ID, "rev": page.Rev,
		"page_size_mm": map[string]float64{"w": w, "h": hh},
		"zones":        zones,
	})
	if err != nil {
		return nil, fmt.Errorf("renderpage: marshal zones: %w", err)
	}

	return doc, nil
}

// stripModules drops the QR matrix from page.json: the contract keeps only the payload.
func stripModules(pageJSON []byte) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(pageJSON, &m); err != nil {
		return pageJSON
	}

	var qr struct {
		Payload string `json:"payload"`
	}

	_ = json.Unmarshal(m["qr"], &qr)
	m["qr"], _ = json.Marshal(map[string]string{"payload": qr.Payload})
	out, _ := json.Marshal(m)

	return out
}

func writeValidated(path, contract string, doc []byte) error {
	if err := contracts.Validate(contract, doc); err != nil {
		return fmt.Errorf("renderpage: %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, doc, 0o600); err != nil {
		return fmt.Errorf("renderpage: %w", err)
	}

	return nil
}

func (h *Handler) next(e eventlog.Envelope, typ, subject string, data json.RawMessage) eventlog.Envelope {
	at := h.now()
	corr := e.CorrelationID

	if corr == "" {
		corr = e.ID
	}

	return eventlog.Envelope{ID: eventlog.NewID(at), Type: typ, At: at, Project: e.Project, Subject: subject, Rev: 1, Data: data, CausationID: e.ID, CorrelationID: corr}
}

func questionIDOf(subject string) string {
	const prefix = "question/"
	if len(subject) > len(prefix) && subject[:len(prefix)] == prefix {
		return subject[len(prefix):]
	}

	return subject
}

// ErrNoTypst tells the composition root the renderer is off.
var ErrNoTypst = errors.New("renderpage: typst unavailable, rendering disabled")
