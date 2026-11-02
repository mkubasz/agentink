# 0009 — `just` is the task runner

- **Status:** accepted
- **Date:** 2026-11-02
- **Spec:** docs/specs/INK-1-agentink-foundation.md · **Decided by:** the owner ("zamiast make użyjmy just")

## Context

The monorepo needs one entry point for the Go hub and the Python worker:
build, test, lint, provenance, e2e, contracts (ADR 0002). A `Makefile` did
that since Phase 1. The owner asked for `just` instead.

## Decision

- A `justfile` at the root replaces the `Makefile`, recipe for recipe:
  `build`, `test`, `lint`, `provenance`, `e2e`, `worker`, `contracts`, `clean`.
- `just gate` runs `test`, `lint` and `provenance`; it is the check before
  every commit (CLAUDE.md). `just e2e` closes every phase.
- Recipes run under `bash -euo pipefail`; `GO` and `UV` in the environment
  override the tools, as the Makefile's variables did.
- `lint` also lints the `e2e` package under its build tag, which the
  Makefile skipped.
- `just` is a developer tool, not a runtime dependency: `inkd`, `ink` and
  `inkworker` never call it. Install with `uv tool install rust-just`,
  `cargo install just`, or the system package manager. Licence: CC0-1.0.

## Alternatives

- **Keep make.** Everywhere already, but tab-sensitive, `.PHONY`
  bookkeeping, and `$(shell …)` quoting; the owner prefers `just`.
- **Task (go-task).** YAML recipes; nothing it adds is needed here.
- **Go and uv commands only, written in the README.** No single gate
  command; the CLAUDE.md rule would become a paragraph.

## Consequences

- Contributors and agents need `just` on PATH; `just` with no arguments
  lists the recipes with their descriptions.
- The spec's log keeps the old `make …` names as history; everything that
  describes the present names `just`.
