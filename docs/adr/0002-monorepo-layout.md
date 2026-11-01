# 0002 — Monorepo layout

- **Status:** proposed
- **Date:** 2026-11-01 (rewritten the same day after Q0 = clean room)
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision points:** D0 (owner), D2 (settled by D0 and the owner's "mono repo")

## Context

Ink Hub is written from scratch (ADR 0003). The owner wants one repository
("mono repo") for the Go hub and CLI, the Python OCR worker, the Typst
templates, the contracts and the skill. The Go style guardrail is the
`samber/cc-skills-golang` skill set (`docs/specs/README.md`), whose
`golang-project-layout` rule is: `main` packages only in `cmd/`, logic in
`internal/`, `pkg/` only for code an external consumer imports, the module
path equal to the repository URL.

## Decision

```
go.mod  go.sum  Makefile  .golangci.yml  .gitignore  README.md  CLAUDE.md  LICENSE
cmd/inkd/              # the hub daemon: flags, wiring, Run()
cmd/ink/               # the CLI for agents and the owner
internal/platform/     # eventlog, bus, lock, config, webdav, auth, mdfile, cli, contracts (schemas embedded, validation)
internal/projects/     # each context: domain/ + one folder per slice
internal/inbox/
internal/pages/        # incl. pages/templates/*.typ (go:embed) and pages/fonts/
internal/ink/
internal/agents/
internal/specflow/
contracts/             # JSON Schema: event, page, zones, answer, ocr (the one source for Go and Python types)
worker/                # Python (uv): pyproject.toml, inkworker/, tests/, samples/ (private)
skills/ink/SKILL.md    # the skill for Claude Code and pi
docs/adr/  docs/architecture/  docs/specs/
```

- **One Go module at the root**, one Python project under `worker/`. No
  `go.work`: there is one module. The Makefile is the one entry point for
  both (`make test` runs `go test` and `uv run pytest`).
- **No `pkg/`** until a consumer outside the module exists; the contract
  validation and the hand-written types (ADR 0008) live in
  `internal/platform/contracts` and move to `pkg/` in their own commit the
  day a mod or a second tool imports them.
- **Typst templates are embedded** in `internal/pages/templates/` with
  `//go:embed`; golden PDFs and zone files in
  `internal/pages/renderpage/testdata/`.
- **The worker** has its own `pyproject.toml` (`uv`), `inkworker/` as the
  package (a local HTTP service, ADR 0007), `tests/`, and `samples/` for the
  owner's handwriting — **private
  by default**, gitignored, nothing published unless the owner decides, kept
  until the owner deletes it. The same holds for `datasets/handwriting/` in
  the data directory.
- **Data at run time** is outside the repository, in the hub's data directory
  (`$XDG_DATA_HOME/inkhub` by default, `--data` / `INKHUB_DATA` to change):
  `projects/<project>/events.jsonl`, `projects/<project>/pages/<page-id>/…`,
  `datasets/handwriting/<id>/`.
- **Lint** is the `.golangci.yml` from the `golang-lint` skill (MIT,
  attributed in the file), `ruff` and `mypy` for the worker.

## Alternatives

- **Two repositories** (hub, worker). Rejected by the owner ("mono repo");
  the contracts in `contracts/` are what both sides compile against, and one
  commit should change a schema and both consumers.
- **`go.work` with a module per context.** Rejected: one small daemon, one
  module; the import-boundary test (ADR 0001) gives the separation.
- **`pkg/` from day one.** Rejected: no external consumer exists yet, and two
  homes for one contract (schemas and types) is one too many.
- **Typst templates in a root `templates/`** (the brief's first sketch).
  Rejected in favour of embedding next to the renderer.

## Consequences

- `make build` builds `inkd` and `ink`; `make worker` syncs the Python
  environment; `make contracts` runs the Go and Python contract tests over
  the shared examples (ADR 0008).
- The repository root is Ink Hub's shape from the first commit; nothing from
  dark-factory is in it (ADR 0003).
