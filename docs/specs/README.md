# Specs

One change is one spec, carried by the spec flow (analyst → researcher →
architect → developer → qa → close). The spec file here is the repository's
record: a line of summary, the plan with its gate while open, the log, and the
final record after close. The long form (the actors' notes, the owner's
summary) lives in the owner's vault, not here.

AgentInk is also a *consumer* of this folder: the `specflow` context reads
`## Clarify` sections and writes answers back (docs/adr/0001). The format is
the spec flow's (`.claude/skills/spec-flow/references/formats.md`);
docs/architecture/inventory.md §7 records how dark-factory's differed, as a
lesson only (ADR 0003).

The flow ships with the repository: the skill, its references and `flow.sh`
under `.claude/skills/spec-flow/`, the `/spec*` commands under
`.claude/commands/`, copied from the owner's gist (revision `a389f4e`). A
clone needs only Claude Code and the Obsidian `Manager` vault
(`.claude/skills/spec-flow/INSTALL.md`, steps 1, 2, 5 and 6). `flow.sh`
checks itself with `bash .claude/skills/spec-flow/scripts/flow.test.sh`.

## Project hook

- **ID prefix:** `INK`
- **Plans:** the phases in the owner's brief, recorded as the tasks of
  `docs/specs/INK-1-agentink-foundation.md` §Plan; the verdict per code area in
  `docs/architecture/inventory.md`.
- **Decision record:** `docs/adr/NNNN-<slug>.md`, template in
  `docs/adr/README.md`. Status *proposed* until the spec closes.
- **Guardrails:** `docs/adr/0001` (import boundaries), `docs/adr/0003`
  (provenance: nothing from dark-factory enters this repository), `CLAUDE.md`
  (the repository's rules for agents), `docs/architecture/security.md` (the
  threat model, a Phase 3 deliverable). Go style: the `samber/cc-skills-golang` skills (MIT), chosen by the owner on
  2026-11-01 — `golang-project-layout` (`main` packages only in `cmd/`, logic
  in `internal/`, `pkg/` only for code external consumers import, the module
  path must match the repository URL), `golang-lint` (one `.golangci.yml` as
  the source of truth, golangci-lint v2, `//nolint:<linter> // reason` only),
  `golang-testing`, `golang-error-handling`, `golang-naming`. Install on the
  machine with `/plugin marketplace add samber/cc` then
  `/plugin install cc-skills-golang@samber`; the repository lists them in its
  `CLAUDE.md` once Phase 1 starts.
- **Verification:** `just test` (Go packages and `worker/`), `just lint`
  (golangci-lint, `ruff` + `mypy`), `just provenance` (no upstream name in the
  tree), `just e2e` (the whole loop with fakes). One Go test: `go test
  ./internal/<ctx>/<slice>/ -run '^TestName$' -count=1`. One worker test:
  `uv run --project worker pytest worker/tests -k <name>`.
- **Gate places:** *local* (`just test`, `just lint`, `just provenance`,
  `just e2e`: the fake agent → WebDAV → stub worker → spec → fake herdr round
  trip, no devices);
  *devices* (a real Boox and a real Supernote on the owner's Tailscale —
  the owner starts these, Phase 3 latency and Phase 4 OCR samples); *paid*
  (an OCR model behind an API — the owner starts it). A row on a device or a
  paid model is never green from a stand-in.
- **Commit:** Conventional Commits (`feat(inbox): …`, `docs(adr): …`,
  `test(platform): …`), one slice per commit series. No hooks are installed
  yet; `git log` is the audit log.
- **CI:** none yet; a GitHub Actions workflow running `just test lint
  provenance` is a Phase 1 deliverable once the repository exists. No branch
  deploys.
- **Contract regeneration:** `just contracts` (Phase 1) regenerates Go types
  from `contracts/*.schema.json` and checks the worker's models against the
  same files; a changed schema is committed with its regenerated output.
- **Vault:** `Manager` (the default). In a cloud session without Obsidian the
  actor notes are written to the session's scratchpad and the summary is
  posted in the reply; the spec file in this folder is the only record.
