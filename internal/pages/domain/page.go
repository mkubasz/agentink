package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Kinds of pages. Every question kind has a page; confirmation and inbox are
// pages the hub makes on its own.
const (
	KindDecision     = "decision"
	KindCritique     = "critique"
	KindEscalation   = "escalation"
	KindStamp        = "stamp"
	KindProposals    = "proposals"
	KindBrief        = "brief"
	KindSummary      = "summary"
	KindInbox        = "inbox"
	KindConfirmation = "confirmation"
)

// Field kinds, as in contracts/page.schema.json.
const (
	FieldChoice = "choice"
	FieldCheck  = "check"
	FieldText   = "text"
	FieldSketch = "sketch"
	FieldStamp  = "stamp"
)

// Option is one choice the owner can mark.
type Option struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Recommended bool   `json:"recommended,omitempty"`
	Why         string `json:"why,omitempty"`
	Cost        string `json:"cost,omitempty"`
}

// Field is one answer zone on the page.
type Field struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Label    string   `json:"label"`
	Options  []Option `json:"options,omitempty"`
	Required bool     `json:"required,omitempty"`
}

// DeviceSpec is the device block of page.json.
type DeviceSpec struct {
	Profile  string `json:"profile"`
	WidthPx  int    `json:"width_px"`
	HeightPx int    `json:"height_px"`
	Format   string `json:"format"`
}

// QR is the code printed on the page: its payload and, for the template, the
// module matrix as rows of "0"/"1" characters.
type QR struct {
	Payload string   `json:"payload"`
	Modules []string `json:"modules,omitempty"`
}

// Page is page.json (contracts/page.schema.json) plus the render-only parts
// the template needs: the body text and the question's items.
type Page struct {
	ID          string     `json:"id"`
	Rev         int        `json:"rev"`
	QuestionID  string     `json:"question_id"`
	QuestionRev int        `json:"question_rev"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Device      DeviceSpec `json:"device"`
	Fields      []Field    `json:"fields"`
	QR          QR         `json:"qr"`
	RenderedAt  string     `json:"rendered_at"` // RFC 3339 with milliseconds, UTC (the contract's pattern)
}

// atLayout is the contract's timestamp shape.
const atLayout = "2006-01-02T15:04:05.000Z"

// Content is what the template shows besides the fields: the question's body
// and items, and the UI strings in the owner's language.
type Content struct {
	Body     string            `json:"body,omitempty"`
	Items    []Item            `json:"items,omitempty"`
	Critical bool              `json:"critical,omitempty"`
	UI       map[string]string `json:"ui"`
}

// Item is a finding or a proposed action, one check zone each.
type Item struct {
	ID       string `json:"id"`
	Severity string `json:"severity,omitempty"`
	Where    string `json:"where,omitempty"`
	Text     string `json:"text"`
}

// question is the part of contracts/question.schema.json the page needs.
type question struct {
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Critical bool     `json:"critical"`
	Options  []Option `json:"options"`
	Items    []Item   `json:"items"`
}

// polishUI is the owner's language on every page (brief §1).
var polishUI = map[string]string{
	"choice":      "Twój wybór",
	"recommended": "rekomendacja",
	"cost":        "koszt",
	"otherwise":   "Inaczej",
	"sketch":      "Szkic",
	"yes":         "TAK",
	"no":          "NIE",
	"signature":   "Podpis",
	"note":        "Notatka",
	"ok":          "✓",
	"fail":        "✗",
	"reason":      "Uzasadnienie",
	"try":         "spróbuj",
	"skip":        "pomiń",
	"stop":        "zatrzymaj",
	"problem":     "Problem",
	"for_whom":    "Dla kogo",
	"done":        "Done",
	"out":         "Poza zakresem",
	"question":    "pytanie",
}

// FromQuestion builds the page and its content for one question revision.
func FromQuestion(raw json.RawMessage, pageID, questionID string, questionRev int, device Device, at time.Time) (Page, Content, error) {
	var q question
	if err := json.Unmarshal(raw, &q); err != nil {
		return Page{}, Content{}, fmt.Errorf("pages: question: %w", err)
	}

	fields, err := fieldsFor(q)
	if err != nil {
		return Page{}, Content{}, err
	}

	p := Page{
		ID: pageID, Rev: 1, QuestionID: questionID, QuestionRev: questionRev, Kind: q.Kind, Title: q.Title,
		Device:     DeviceSpec{Profile: device.Name, WidthPx: device.WidthPx, HeightPx: device.HeightPx, Format: device.Format},
		Fields:     fields,
		QR:         QR{Payload: fmt.Sprintf("INK:%s:%d", pageID, 1)},
		RenderedAt: at.UTC().Format(atLayout),
	}

	return p, Content{Body: q.Body, Items: q.Items, Critical: q.Critical, UI: polishUI}, nil
}

func fieldsFor(q question) ([]Field, error) {
	switch q.Kind {
	case KindDecision:
		return []Field{
			{ID: "choice", Kind: FieldChoice, Label: polishUI["choice"], Options: q.Options, Required: true},
			{ID: "otherwise", Kind: FieldText, Label: polishUI["otherwise"]},
			{ID: "sketch", Kind: FieldSketch, Label: polishUI["sketch"]},
		}, nil
	case KindStamp:
		return []Field{
			{ID: "stamp", Kind: FieldStamp, Label: q.Title, Options: []Option{{Key: "yes", Label: polishUI["yes"]}, {Key: "no", Label: polishUI["no"]}}, Required: true},
			{ID: "signature", Kind: FieldText, Label: polishUI["signature"], Required: true},
			{ID: "note", Kind: FieldText, Label: polishUI["note"]},
		}, nil
	case KindEscalation:
		return []Field{
			{ID: "choice", Kind: FieldChoice, Label: q.Title, Options: []Option{{Key: "try", Label: polishUI["try"]}, {Key: "skip", Label: polishUI["skip"]}, {Key: "stop", Label: polishUI["stop"]}}, Required: true},
			{ID: "note", Kind: FieldText, Label: polishUI["note"]},
		}, nil
	case KindCritique, KindProposals:
		fields := make([]Field, 0, len(q.Items)+1)
		for _, it := range q.Items {
			fields = append(fields, Field{ID: "item-" + it.ID, Kind: FieldCheck, Label: it.Text, Options: []Option{{Key: "ok", Label: polishUI["ok"]}, {Key: "fail", Label: polishUI["fail"]}}})
		}

		return append(fields, Field{ID: "reason", Kind: FieldText, Label: polishUI["reason"]}), nil
	case KindBrief:
		return []Field{
			{ID: "problem", Kind: FieldText, Label: polishUI["problem"]},
			{ID: "for_whom", Kind: FieldText, Label: polishUI["for_whom"]},
			{ID: "done", Kind: FieldText, Label: polishUI["done"]},
			{ID: "out", Kind: FieldText, Label: polishUI["out"]},
			{ID: "sketch", Kind: FieldSketch, Label: polishUI["sketch"]},
		}, nil
	case KindSummary:
		return []Field{{ID: "note", Kind: FieldSketch, Label: polishUI["note"]}}, nil
	default:
		return nil, fmt.Errorf("pages: no page for question kind %q", q.Kind)
	}
}
