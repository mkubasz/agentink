# 0002 — Monorepo layout

- **Status:** proposed
- **Date:** 2026-11-01 (rewritten the same day after Q0 = clean room)
- **Spec:** docs/specs/INK-1-agentink-foundation.md · **Decision points:** D0 (owner), D2 (settled by D0 and the owner's "mono repo")

## Context

AgentInk is written from scratch (ADR 0003). The owner wants one repository
("mono repo") for the Go hub and CLI, the Python OCR worker, the Typst
templates, the contracts and the skill. The Go style guardrail is the
`samber/cc-skills-golang` skill set (`docs/specs/README.md`), whose
`golang-project-layout` rule is: `main` packages only in `cmd/`, logic in
`internal/`, `pkg/` only for code an external consumer imports, the module
path equal to the repository URL.

## Decision

```
go.mod  go.sum  justfile  .golangci.yml  .gitignore  README.md  CLAUDE.md  LICENSE
cmd/inkd/              # the hub daemon: flags, wiring, Run()
cmd/ink/               # the CLI for agents and the owner
internal/platform/     # eventlog, bus, lock, config, webdav, auth, mdfile, cli
internal/projects/     # each context: domain/ + one folder per slice
internal/inbox/
internal/pages/        # incl. pages/domain/templates/*.typ (go:embed) and pages/domain/fonts/
internal/ink/
internal/agents/
internal/specflow/
contracts/             # JSON Schema (event, question, page, zones, ocr, answer) + the Go package that embeds and validates them
worker/                # Python (uv): pyproject.toml, inkworker/, tests/, samples/ (private)
skills/ink/SKILL.md    # the skill for Claude Code and pi
docs/adr/  docs/architecture/  docs/specs/
```

- **One Go module at the root**, one Python project under `worker/`. No
  `go.work`: there is one module. The justfile is the one entry point for
  both (`just test` runs `go test` and `uv run pytest`; ADR 0009).
- **No `pkg/`** until a consumer outside the module exists. The one
  importable package is `contracts/` (the schemas and their validator, ADR
  0008), which sits at the root because `go:embed` cannot reach a parent
  directory and because it is the contract other tools may import.
- **Typst templates are embedded** in `internal/pages/domain/templates/` with
  `//go:embed`; golden PDFs and zone files in
  `internal/pages/renderpage/testdata/`.
- **The worker** has its own `pyproject.toml` (`uv`), `inkworker/` as the
  package (a local HTTP service, ADR 0007), `tests/`, and `samples/` for the
  owner's handwriting — **private
  by default**, gitignored, nothing published unless the owner decides, kept
  until the owner deletes it. The same holds for `datasets/handwriting/` in
  the data directory.
- **Data at run time** is outside the repository, in the hub's data directory
  (`$XDG_DATA_HOME/agentink` by default, `--data` / `AGENTINK_DATA` to change):
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

- `just build` builds `inkd` and `ink`; `just worker` syncs the Python
  environment; `just contracts` runs the Go and Python contract tests over
  the shared examples (ADR 0008).
- The repository root is AgentInk's shape from the first commit; nothing from
  dark-factory is in it (ADR 0003).
