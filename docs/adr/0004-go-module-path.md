# 0004 — Go module path

- **Status:** proposed
- **Date:** 2026-11-01 (rewritten the same day after Q0 = clean room)
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision point:** D1

## Context

A new module needs a path, and the `golang-project-layout` rule is that the
module path equals the repository URL. The owner answered D1 ("rename") when
the plan was still a fork; under the clean room there is nothing to rename,
only a name to choose, and the repository does not exist yet.

## Decision

- The module path is the new repository's URL without the scheme. Until the
  owner gives it, `go.mod` carries the placeholder
  `github.com/mkubasz/ink-hub` (the architect's recommendation: the owner's
  account, the product's name, lowercase, hyphenated).
- Changing the placeholder is one `go mod edit -module` plus a `sed` over the
  module's own imports, made before the first push; no third party imports
  the module yet, so nothing else moves.
- Binaries are `inkd` and `ink`; `go install <path>/cmd/ink@latest` is the
  install line in the README.

## Alternatives

- **A vanity import path** (`inkhub.dev/...`). Rejected for now: one more
  thing to host; can be added later with a redirect without renaming.
- **Keep any upstream name.** Nothing upstream exists in this repository.

## Consequences

- The first push waits for the URL (also the one thing the push of this
  history needs).
- Every other name in the repository (`inkd`, `ink`, `inkworker`, `INKHUB_*`
  environment variables, the `inkhub` data directory) is ours from the start.
