// Package api holds the wire types of inkd's local HTTP API, shared by the
// slices that serve them and the CLI client that calls them. It depends on
// nothing but the standard library, so both sides can import it.
package api

import (
	"encoding/json"
	"time"
)

// Project is a registered repository (ADR 0001, projects context).
type Project struct {
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	SpecsDir     string    `json:"specs_dir"`
	Orchestrator string    `json:"orchestrator"`
	RegisteredAt time.Time `json:"registered_at"`
}

// RegisterProjectRequest is POST /v1/projects.
type RegisterProjectRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name,omitempty"`
	Path         string `json:"path"`
	SpecsDir     string `json:"specs_dir,omitempty"`
	Orchestrator string `json:"orchestrator,omitempty"`
}

// AskRequest is POST /v1/questions: a question (contracts/question.schema.json)
// for a project named explicitly or found by the caller's working directory.
type AskRequest struct {
	Project  string          `json:"project,omitempty"`
	Cwd      string          `json:"cwd,omitempty"`
	Question json.RawMessage `json:"question"`
}

// AskResponse is what POST /v1/questions returns.
type AskResponse struct {
	QuestionID string `json:"question_id"`
	Project    string `json:"project"`
	Rev        int    `json:"rev"`
	Status     string `json:"status"`
}

// InboxRow is one line of the owner's inbox.
type InboxRow struct {
	QuestionID string    `json:"question_id"`
	Project    string    `json:"project"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Critical   bool      `json:"critical"`
	Status     string    `json:"status"`
	AskedAt    time.Time `json:"asked_at"`
}

// Question is GET /v1/questions/{id}: where one question stands and, once
// the owner confirmed it, the answer (contracts/answer.schema.json) and the
// owner's words. Answer is empty while Status is before "confirmed".
type Question struct {
	QuestionID  string          `json:"question_id"`
	Project     string          `json:"project"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title"`
	Critical    bool            `json:"critical"`
	Status      string          `json:"status"`
	AskedAt     time.Time       `json:"asked_at"`
	Answer      json.RawMessage `json:"answer,omitempty"`
	Words       string          `json:"words,omitempty"`
	ConfirmedAt *time.Time      `json:"confirmed_at,omitempty"`
}

// Status is GET /v1/status.
type Status struct {
	Version       string `json:"version"`
	DataDir       string `json:"data_dir"`
	Render        string `json:"render"` // "on", or why rendering is off
	Projects      int    `json:"projects"`
	OpenQuestions int    `json:"open_questions"`
}

// Error is every non-2xx body.
type Error struct {
	Error string `json:"error"`
}
