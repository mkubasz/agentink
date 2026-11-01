// Package contracts holds the JSON Schemas that Go and Python agree on (ADR
// 0008): the event envelope, the question an agent asks, the rendered page,
// its zones, the OCR result and the answer. The schemas are the source of
// truth; this package embeds them and validates documents against them.
package contracts

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Names of the contracts, each backed by <name>.schema.json.
const (
	Event    = "event"
	Question = "question"
	Page     = "page"
	Zones    = "zones"
	OCR      = "ocr"
	Answer   = "answer"
)

// Names lists every contract in a fixed order.
var Names = []string{Event, Question, Page, Zones, OCR, Answer}

// ErrInvalid wraps every validation failure.
var ErrInvalid = errors.New("contracts: document does not match the schema")

// Schemas is the embedded schema set, also served to the Python side by path.
//
//go:embed *.schema.json
var Schemas embed.FS

const baseURL = "https://agentink.local/contracts/"

var (
	compileOnce sync.Once
	compiled    map[string]*jsonschema.Schema
	errCompile  error
)

func compile() {
	c := jsonschema.NewCompiler()
	compiled = make(map[string]*jsonschema.Schema, len(Names))

	for _, name := range Names {
		raw, err := Schemas.ReadFile(name + ".schema.json")
		if err != nil {
			errCompile = fmt.Errorf("contracts: read %s: %w", name, err)

			return
		}

		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			errCompile = fmt.Errorf("contracts: parse %s: %w", name, err)

			return
		}

		url := baseURL + name + ".schema.json"
		if err := c.AddResource(url, doc); err != nil {
			errCompile = fmt.Errorf("contracts: add %s: %w", name, err)

			return
		}
	}

	for _, name := range Names {
		sch, err := c.Compile(baseURL + name + ".schema.json")
		if err != nil {
			errCompile = fmt.Errorf("contracts: compile %s: %w", name, err)

			return
		}

		compiled[name] = sch
	}
}

// Validate checks one JSON document against the named contract.
func Validate(name string, doc []byte) error {
	compileOnce.Do(compile)

	if errCompile != nil {
		return errCompile
	}

	sch, ok := compiled[name]
	if !ok {
		return fmt.Errorf("contracts: unknown contract %q", name)
	}

	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("%w: %s: not JSON: %w", ErrInvalid, name, err)
	}

	if err := sch.Validate(v); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalid, name, err)
	}

	return nil
}
