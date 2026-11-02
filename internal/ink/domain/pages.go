package domain

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Events the ink context folds about pages it published and ink it received.
const (
	EventPageRendered   = "PageRendered"
	EventPagePublished  = "PagePublished"
	EventInkReceived    = "InkReceived"
	EventIngestRejected = "IngestRejected"
)

// PublishedPage is what the hub knows about a page the readers can see.
type PublishedPage struct {
	PageID      string
	Rev         int
	Project     string
	QuestionID  string
	QuestionRev int
	PageDir     string
	// OriginalSHA256 is the PDF as published; an upload with the same bytes
	// is the sync's echo, not an answer.
	OriginalSHA256 string
	File           string
}

// Pages is the projection ingest needs: pages by id, and the uploads already
// received so the same bytes never count twice (spec task 3.2).
type Pages struct {
	mu       sync.RWMutex
	byID     map[string]PublishedPage
	received map[string]bool // page_id + rev + sha256
}

// NewPages returns an empty projection.
func NewPages() *Pages { return &Pages{byID: map[string]PublishedPage{}, received: map[string]bool{}} }

// Apply folds PageRendered, PagePublished and InkReceived.
func (ps *Pages) Apply(e eventlog.Envelope) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	switch e.Type {
	case EventPageRendered:
		var d struct {
			PageID      string `json:"page_id"`
			QuestionID  string `json:"question_id"`
			QuestionRev int    `json:"question_rev"`
			PageDir     string `json:"page_dir"`
		}

		if err := json.Unmarshal(e.Data, &d); err != nil {
			return fmt.Errorf("ink: bad %s in %s: %w", e.Type, e.ID, err)
		}

		p := ps.byID[d.PageID]
		p.PageID, p.Rev, p.Project, p.QuestionID, p.QuestionRev, p.PageDir = d.PageID, e.Rev, e.Project, d.QuestionID, d.QuestionRev, d.PageDir
		ps.byID[d.PageID] = p
	case EventPagePublished:
		var d struct {
			PageID string `json:"page_id"`
			SHA256 string `json:"sha256"`
			File   string `json:"file"`
		}

		if err := json.Unmarshal(e.Data, &d); err != nil {
			return fmt.Errorf("ink: bad %s in %s: %w", e.Type, e.ID, err)
		}

		p := ps.byID[d.PageID]
		p.PageID, p.OriginalSHA256, p.File = d.PageID, d.SHA256, d.File
		ps.byID[d.PageID] = p
	case EventInkReceived:
		var d struct {
			PageID string `json:"page_id"`
			Rev    int    `json:"rev"`
			SHA256 string `json:"sha256"`
		}

		if err := json.Unmarshal(e.Data, &d); err != nil {
			return fmt.Errorf("ink: bad %s in %s: %w", e.Type, e.ID, err)
		}

		ps.received[receivedKey(d.PageID, d.Rev, d.SHA256)] = true
	}

	return nil
}

// Get returns a published page.
func (ps *Pages) Get(pageID string) (PublishedPage, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	p, ok := ps.byID[pageID]

	return p, ok && p.PageDir != ""
}

// Received says whether these bytes for this page were already taken in.
func (ps *Pages) Received(pageID string, rev int, sha256 string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.received[receivedKey(pageID, rev, sha256)]
}

func receivedKey(pageID string, rev int, sha string) string {
	return fmt.Sprintf("%s:%d:%s", pageID, rev, sha)
}
