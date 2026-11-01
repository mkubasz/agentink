# 0003 — Clean-room rebuild and provenance rules

- **Status:** proposed
- **Date:** 2026-11-01 (rewritten the same day: supersedes the strangler plan after Q0 = clean room)
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision point:** D0

## Context

The brief started from a fork of dark-factory. Phase 0 traced its licence
history (no licence v0.1.0–v0.32; FSL-1.1-MIT from v0.33.0; PolyForm
Noncommercial 1.0.0 from 2026-10-23) and found that the owner's goal — no
obligation to the upstream author — cannot be met by refactoring a copy: a
work that contains his code, however moved or edited, stays a derivative work.
The owner decided (2026-11-01): "Piszemy od zera od razu poprawione i zgodne z
dobrymi praktykami." Ideas are not copyrightable; expression is. Ink Hub takes
the ideas.

## Decision

- **A new repository, written from scratch.** No file, function, template,
  stylesheet, fixture or sentence of documentation from dark-factory or AI
  Workflow Engine enters this repository. The fork is not a parent of this
  history.
- **What is taken: concepts.** The clarify round with one recommended answer,
  the "Needs you" list, the gate as a question that needs a stamp, the fresh
  critic's `blocks/should/note` findings, the red-round cap, the post-mortem's
  proposed actions, markdown files with YAML frontmatter as the record, the
  owner's words as the recorded answer. `docs/architecture/inventory.md` is
  the map of those concepts and of the lessons (what failed silently, what
  was tri-located, what had no id); it names dark-factory symbols as
  references to behaviour, not as code to port.
- **Formats come from the spec flow**, not from dark-factory: the clarify
  grammar, the decision line `D<n> — <question> → <letter>, <option> — <owner,
  date>` and the `clarify answered` log line are the gist's `formats.md`, and
  Ink Hub's own `contracts/` define everything else.
- **Provenance rules, kept for the life of the project:**
  1. Nobody writes Ink Hub code with a dark-factory source file open. The
     inventory and the spec flow's documents are the references.
  2. dark-factory may be installed and run on the owner's machine as a black
     box to compare behaviour (its licence permits that use); its output, not
     its code, is what is compared.
  3. Third-party code enters only through `go.mod` / `pyproject.toml` with
     its licence recorded in `ACKNOWLEDGMENTS.md`; no vendored snippets.
  4. A CI check greps the tree for the upstream module path, author names and
     product names and fails on a match outside `docs/architecture/inventory.md`
     and `ACKNOWLEDGMENTS.md`.
  5. The inspiration is acknowledged by name in `ACKNOWLEDGMENTS.md`, with
     no text copied from either project.
- **Licence:** the owner's choice between MIT and Apache-2.0 (asked in the
  spec's round); `LICENSE` is added with that answer, before the first push.
- **Nothing is migrated**, so there is no legacy tier, no baseline allowlist
  and no park/drop ledger: `make test` and `make lint` are the whole gate.

## Alternatives

- **Fork and stay under PolyForm Noncommercial.** Every line available;
  rejected by the owner: the notices and the noncommercial limit would last
  for the life of the project.
- **Fork from the last FSL-1.1-MIT commit.** More permissive, but the upstream
  licence still travels until 2027–2028 and everything after 2026-10-23 is
  lost; rejected with the above.
- **Strangler over the fork** (the first version of this ADR): sound as
  engineering, void as a way out of the licence; superseded.

## Consequences

- The ~7,500 lines the fork's *keep*/*rework* rows named are written anew,
  smaller, against the spec flow's formats and our contracts; the fork's
  tests are not available as an oracle, so each format gets its own golden
  tests in Phase 1–2.
- Phase 0's `NOTICE` draft (the PolyForm notices) is withdrawn; it is replaced
  by `LICENSE` and `ACKNOWLEDGMENTS.md`.
- The fork clone and its `ink-hub/phase-0` branch stay on the research
  machine as the record of Phase 0 and are never pushed into this repository.
