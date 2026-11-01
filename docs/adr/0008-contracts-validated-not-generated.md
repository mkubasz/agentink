# 0008 — Contracts: JSON Schema as the source of truth, validated on both sides

- **Status:** proposed
- **Date:** 2026-11-01
- **Spec:** docs/specs/INK-1-agentink-foundation.md · **Decision point:** settled by the architect (brief §9)

## Context

Go (`inkd`, `ink`) and Python (`inkworker`) exchange files and HTTP bodies:
the event envelope, `page.json`, `zones.json`, the answer, `ocr.json`, and
the question an agent hands to `ink ask`. The brief wants them as JSON Schema
in `contracts/`. The open question was whether to generate the Go and Python
types from the schemas or to write them by hand.

## Decision

- **`contracts/*.schema.json` (JSON Schema 2020-12) is the source of truth**
  for six documents: `event`, `question`, `page`, `zones`, `answer`, `ocr`.
  Each schema ships with examples under `contracts/examples/<name>/` (valid)
  and `contracts/examples/<name>/invalid/` (must fail).
- **Types are written by hand on each side and proven by tests, not
  generated.** Go: the `contracts` package at the repository root (next to
  the schemas, so `go:embed` can reach them) exposes `Validate(name, doc)`;
  it is the one package meant to be importable from outside the module,
  because it *is* the contract; a test validates every example and every
  Go type's marshalled output. Python: the worker validates its inputs and
  its `ocr.json` with the `jsonschema` package against the same files; a test
  runs the same examples. `make contracts` runs both test sets.
- Why not generate: at six small schemas, generators (Go: `go-jsonschema`;
  Python: `datamodel-code-generator`) add a build step, a style the linters
  fight, and types that still need hand-written methods. The tests catch the
  drift a generator would prevent. Revisit when a schema passes ~30 fields.
- **Libraries:** Go `github.com/santhosh-tekuri/jsonschema/v6` (Apache-2.0,
  2020-12 support); Python `jsonschema` (MIT).

## Alternatives

- **Generate Go types from the schemas.** Rejected for now (above); the
  decision is reversible because the schemas stay the source of truth.
- **Protobuf / gRPC.** A second IDL for six JSON files; rejected.
- **No schemas, types only.** Rejected by the brief.

## Consequences

- Two new dependencies, recorded in `ACKNOWLEDGMENTS.md`.
- A schema change is one commit touching the schema, its examples, and the
  types on the side(s) it affects; `make contracts` fails until all agree.
