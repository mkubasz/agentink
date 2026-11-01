// Package domain is the inbox context: the lifecycle of a question from the
// moment an agent asks it to the moment its answer is delivered (ADR 0001).
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Event types the inbox folds.
const (
	EventAsked     = "QuestionAsked"
	EventWithdrawn = "QuestionWithdrawn"
	EventRendered  = "PageRendered"
	EventPublished = "PagePublished"
	EventReceived  = "InkReceived"
	EventConfirmed = "AnswerConfirmed"
	EventDelivered = "AnswerDelivered"
)

// Statuses a question moves through; the inbox shows every one but the last two.
const (
	StatusAsked     = "asked"
	StatusRendered  = "rendered"
	StatusPublished = "published"
	StatusReceived  = "received"
	StatusConfirmed = "confirmed"
	StatusDelivered = "delivered"
	StatusWithdrawn = "withdrawn"
)

// Asked is the payload of QuestionAsked: the question as the agent gave it
// (contracts/question.schema.json) and the hash that defines its revision.
type Asked struct {
	Question    json.RawMessage `json:"question"`
	ContentHash string          `json:"content_hash"`
}

// head is the part of a question the inbox needs.
type head struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Critical bool   `json:"critical"`
}

// ContentHash is the hash of what the owner sees: kind, title, body, options
// and items. A change to any of them is a new question revision (ADR 0005).
func ContentHash(question json.RawMessage) (string, error) {
	var q struct {
		Kind    string          `json:"kind"`
		Title   string          `json:"title"`
		Body    string          `json:"body"`
		Options json.RawMessage `json:"options"`
		Items   json.RawMessage `json:"items"`
	}

	if err := json.Unmarshal(question, &q); err != nil {
		return "", fmt.Errorf("inbox: content hash: %w", err)
	}

	canon, err := json.Marshal(q)
	if err != nil {
		return "", fmt.Errorf("inbox: content hash: %w", err)
	}

	sum := sha256.Sum256(canon)

	return hex.EncodeToString(sum[:16]), nil
}

// Inbox is the projection the owner reads: every open question, critical
// ones first, then the oldest. It is rebuilt from the log at start.
type Inbox struct {
	mu   sync.RWMutex
	rows map[string]api.InboxRow
}

// NewInbox returns an empty inbox.
func NewInbox() *Inbox { return &Inbox{rows: map[string]api.InboxRow{}} }

// Apply folds one event; unrelated types are ignored.
func (in *Inbox) Apply(e eventlog.Envelope) error {
	in.mu.Lock()
	defer in.mu.Unlock()

	switch e.Type {
	case EventAsked:
		return in.applyAsked(e)
	case EventRendered, EventPublished, EventReceived, EventConfirmed:
		in.setStatus(questionID(e.Subject), statusFor(e.Type))
	case EventDelivered, EventWithdrawn:
		delete(in.rows, questionID(e.Subject))
	}

	return nil
}

func (in *Inbox) applyAsked(e eventlog.Envelope) error {
	var d Asked
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return fmt.Errorf("inbox: bad %s data in %s: %w", e.Type, e.ID, err)
	}

	var h head
	if err := json.Unmarshal(d.Question, &h); err != nil {
		return fmt.Errorf("inbox: bad question in %s: %w", e.ID, err)
	}

	id := questionID(e.Subject)
	in.rows[id] = api.InboxRow{
		QuestionID: id, Project: e.Project, Kind: h.Kind, Title: h.Title,
		Critical: h.Critical, Status: StatusAsked, AskedAt: e.At,
	}

	return nil
}

func (in *Inbox) setStatus(id, status string) {
	if row, ok := in.rows[id]; ok {
		row.Status = status
		in.rows[id] = row
	}
}

func statusFor(eventType string) string {
	switch eventType {
	case EventRendered:
		return StatusRendered
	case EventPublished:
		return StatusPublished
	case EventReceived:
		return StatusReceived
	case EventConfirmed:
		return StatusConfirmed
	default:
		return StatusAsked
	}
}

func questionID(subject string) string {
	const prefix = "question/"
	if len(subject) > len(prefix) && subject[:len(prefix)] == prefix {
		return subject[len(prefix):]
	}

	return subject
}

// Rows returns the open questions, for one project or all, critical first,
// then oldest first.
func (in *Inbox) Rows(project string) []api.InboxRow {
	in.mu.RLock()
	defer in.mu.RUnlock()

	out := make([]api.InboxRow, 0, len(in.rows))

	for _, r := range in.rows {
		if project == "" || r.Project == project {
			out = append(out, r)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Critical != out[j].Critical {
			return out[i].Critical
		}

		if !out[i].AskedAt.Equal(out[j].AskedAt) {
			return out[i].AskedAt.Before(out[j].AskedAt)
		}

		return out[i].QuestionID < out[j].QuestionID
	})

	return out
}

// Open counts the open questions.
func (in *Inbox) Open() int {
	in.mu.RLock()
	defer in.mu.RUnlock()

	return len(in.rows)
}

// Now is the clock the slices use; tests replace it.
var Now = time.Now
