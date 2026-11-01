// Package eventlog is the append-only JSONL event log: one file per project,
// one envelope per line, the only state that outlives a request (ADR 0005).
package eventlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidEnvelope is returned when an envelope lacks a required field.
var ErrInvalidEnvelope = errors.New("eventlog: invalid envelope")

// atLayout is RFC 3339 with milliseconds, always UTC.
const atLayout = "2006-01-02T15:04:05.000Z"

// Envelope is one event: what happened, to which subject, in which project,
// and what caused it. Data is the event's own payload, validated against
// contracts/event.schema.json per Type.
type Envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	At            time.Time       `json:"at"`
	Project       string          `json:"project"`
	Subject       string          `json:"subject"`
	Rev           int             `json:"rev"`
	Data          json.RawMessage `json:"data,omitempty"`
	CausationID   string          `json:"causation_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

// wire is the on-disk shape: the same fields with At as a string.
type wire struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	At            string          `json:"at"`
	Project       string          `json:"project"`
	Subject       string          `json:"subject"`
	Rev           int             `json:"rev"`
	Data          json.RawMessage `json:"data,omitempty"`
	CausationID   string          `json:"causation_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

// Validate reports the first missing required field.
func (e Envelope) Validate() error {
	switch {
	case e.ID == "":
		return fmt.Errorf("%w: missing id", ErrInvalidEnvelope)
	case e.Type == "":
		return fmt.Errorf("%w: missing type", ErrInvalidEnvelope)
	case e.At.IsZero():
		return fmt.Errorf("%w: missing at", ErrInvalidEnvelope)
	case e.Project == "":
		return fmt.Errorf("%w: missing project", ErrInvalidEnvelope)
	case e.Subject == "":
		return fmt.Errorf("%w: missing subject", ErrInvalidEnvelope)
	case e.Rev < 0:
		return fmt.Errorf("%w: negative rev", ErrInvalidEnvelope)
	}

	return nil
}

// MarshalJSON writes At as RFC 3339 with milliseconds in UTC.
func (e Envelope) MarshalJSON() ([]byte, error) {
	w := wire{
		ID:            e.ID,
		Type:          e.Type,
		At:            e.At.UTC().Format(atLayout),
		Project:       e.Project,
		Subject:       e.Subject,
		Rev:           e.Rev,
		Data:          e.Data,
		CausationID:   e.CausationID,
		CorrelationID: e.CorrelationID,
	}

	b, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("eventlog: marshal %s: %w", e.ID, err)
	}

	return b, nil
}

// UnmarshalJSON reads the on-disk shape and parses At.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	var w wire
	if err := json.Unmarshal(b, &w); err != nil {
		return fmt.Errorf("eventlog: unmarshal: %w", err)
	}

	at, err := time.Parse(time.RFC3339Nano, w.At)
	if err != nil {
		return fmt.Errorf("eventlog: unmarshal at %q: %w", w.At, err)
	}

	*e = Envelope{
		ID:            w.ID,
		Type:          w.Type,
		At:            at.UTC(),
		Project:       w.Project,
		Subject:       w.Subject,
		Rev:           w.Rev,
		Data:          w.Data,
		CausationID:   w.CausationID,
		CorrelationID: w.CorrelationID,
	}

	return nil
}
