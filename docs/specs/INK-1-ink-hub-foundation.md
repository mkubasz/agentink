---
id: INK-1
title: Ink Hub foundation — inventory, architecture and the clean-room plan
step: architect
status: blocked
mode: session
branch: main
repo: ink-hub
vault: Tickets/INK-1-ink-hub-foundation
created: 2026-11-01
updated: 2026-11-01
tags: [spec/INK-1, step/architect, branch/main, status/blocked]
---
<!-- spec-card -->

# INK-1 — Ink Hub foundation — inventory, architecture and the clean-room plan

## Summary
Build Ink Hub from scratch, taking dark-factory's ideas and none of its code: a local hub that renders coding agents' questions as handwriting-friendly e-ink pages, reads the owner's ink back and delivers the answer to the right agent. Phase 0 mapped the fork, fixed the architecture and the provenance rules, and set up the monorepo; it stops for the last answers before Phase 1.

## Plan
**Delivers:** the owner's brief, phases 0–6 · **Built by:** one implementer, one slice at a time (brief §9) · **Extra file:** none — `docs/adr/0001`–`0006` and `docs/architecture/inventory.md` hold the boundary · **ADR:** [0001](../adr/0001-bounded-contexts-events-and-vertical-slices.md), [0002](../adr/0002-monorepo-layout.md), [0003](../adr/0003-clean-room-rebuild-and-provenance.md), [0004](../adr/0004-go-module-path.md), [0005](../adr/0005-event-log-format-and-identifiers.md), [0006](../adr/0006-page-rendering-with-typst.md), all *proposed*
**Mechanism:** the per-project JSONL event log is the only state that outlives a request; the inbox is a projection held in memory and rebuilt from the log at start; DuckDB queries the same files for analytics (D9). A Question has a revision (its text and options), a Page has a revision (one render of a question revision; a confirmation page is a new page of the same question); the QR carries the page id and revision, an answer names them, and an answer for a stale page is `AnswerRejected` (ADR 0005). `deliveranswer` owns retries: every attempt is an event, three failures become an `escalation` page. Low confidence never delivers: `ConfirmationRequested` loops through a confirmation page, and every correction lands in `datasets/handwriting/`. Handwriting is data: the prompt an adapter delivers carries the question id and the `ink answer` command, never the recognized text (brief §7). Nothing is ported from dark-factory (ADR 0003).

- [ ] 0.1 Inventory of the fork, now a map of concepts and lessons — `docs/architecture/inventory.md` · written; proved by the owner's review of this round
- [ ] 0.2 ADRs 0001–0006 · written as *proposed*, 0002–0004 rewritten for the clean room; proved by the owner's review of this round
- [ ] 0.3 The monorepo scaffold: `go.mod` (placeholder path), `Makefile`, `.golangci.yml`, `.gitignore`, `README.md`, `CLAUDE.md`, `ACKNOWLEDGMENTS.md`, `worker/pyproject.toml` · proved by `make lint` on an empty tree and `uv sync --project worker`
- [ ] 0.4 The owner's answers to Q5, the licence, the repository URL and the D9 reading recorded here; `LICENSE` added; `go.mod` path set; the first push
- [ ] 1.1 `internal/platform`: eventlog (append + fsync, truncated-last-line skip, replay), bus (sync fan-out, error isolation), lock, config — D3, D4 (ADR 0005) · proved by `TestEventLog_AppendReplay`, `TestEventLog_SkipsTruncatedLastLine`, `TestBus_FanoutIsolatesErrors`
- [ ] 1.2 `contracts/`: JSON Schemas for event, page, zones, answer, ocr; Go types generated into `internal/platform/contracts`; example validation in Go and Python · proved by `TestContracts_Examples`, `pytest worker/tests/test_contracts.py`
- [ ] 1.3 Slice `askquestion` + `ink ask` + `ink inbox` (`--json`); the inbox projection rebuilt from the log — D3, D9 · proved by `TestAskQuestion_AppendsEventAndProjectsInbox`, `TestInbox_RebuiltFromLog`, `TestInkAsk_CapturesHerdrAndRexEnv`, `TestInkAsk_FailsFastWithoutDaemon`
- [ ] 1.4 `ink answer <id> --json`, `ink show <md>`, `ink brief`, `ink status` · proved by `TestInkAnswer_ReturnsConfirmedAnswerOnly`, `TestInkStatus_WithoutDaemonExit3`
- [ ] 1.5 Import-boundary test — ADR 0001 · proved by `TestArch_SlicesImportOnlyDomainAndPlatform`
- [ ] 1.6 Provenance check — ADR 0003 rule 4 · proved by `make provenance` (a grep that fails on upstream names outside the two allowed files)
- [ ] 1.7 A thin end-to-end skeleton from Phase 1: fake agent → `ink ask` → stub render → stub ingest → stub worker → `syncspec` → fake adapter; every later phase replaces one stub · proved by `make e2e` green at the end of every phase
- [ ] 2.1 `pages`: the `decision` template, zones via `typst eval` (ADR 0006), golden PDF and `zones.json`; the QR payload `INK:<page-ulid>:<rev>` · proved by `TestRenderDecision_Golden`, `TestQR_PayloadIsAlphanumeric`
- [ ] 3.1 `platform/webdav` + `publishpage`: `Ink/<project>/…` per device token, Tailscale-only listener, a PUT cap (64 MiB default, `webdav.max_upload`) · proved by `TestWebDAV_DeviceTokenScopesTree`, `TestWebDAV_RejectsOversizedPut`, `TestListener_RefusesNonTailscale`
- [ ] 3.1b Device pairing and revocation: `ink device add <name>` prints a one-time pairing code, tokens are hashed at rest, `ink device revoke <name>` ends access, a device sees only the roots it is scoped to · proved by `TestDevice_PairRevoke`, `TestDevice_RevokedTokenGets401`
- [ ] 3.2 `ingestink`: an upload becomes `InkReceived` — **must not** emit twice for the same bytes of the same page (dedupe key `page_id` + `rev` + sha256), must ignore 0-byte and non-PDF/`.mark` files with an `IngestRejected` event · proved by `TestIngest_UploadEmitsInkReceived`, `TestIngest_SameBytesOnce`, `TestIngest_RejectsEmptyAndUnknown`
- [ ] 3.2b `ingestink` watches configured sync folders too (Boox returns the annotated PDF through Syncthing, brief §6.3) · proved by `TestIngest_WatchedFolderEmitsInkReceived`
- [ ] 3.3 Latency of the full loop on a Boox and a Supernote → `docs/architecture/latency.md` — owner-started gate row
- [ ] 3.4 Ink Hub's threat model — `docs/architecture/security.md`: devices, tokens, the tailnet, handwriting as untrusted data, what a stamp gates (brief §7) · proved by the owner's review and by the tests it names
- [ ] 4.1 `worker/`: parse PDF/`.mark`/`.note`, align on QR and corner marks, crop zones, deterministic boxes, `stub` + one real engine, CER harness over `worker/samples/` (private) with DuckDB over the results; a run writes into `<page-dir>/.run-<ulid>/` and renames on success; `inkd` kills a run after `worker.timeout` (default 5 min) and emits `RecognitionFailed` — D5, D9 · proved by `uv run --project worker pytest`, `TestRecognize_KilledRunLeavesNoPartialOutput`
- [ ] 4.2 `recognizeanswer` / `confirmanswer`: recognized text becomes the owner's words only when every text field is at or above `ocr.confidence_min` (default 0.9), else after the confirmation page; checkbox-only answers carry the option label and any free text · proved by `TestRecognize_LowConfidenceRequestsConfirmation`, `TestConfirm_RejectsStalePage`, `TestConfirm_RawOcrNeverLeavesOcrJson`
- [ ] 5.1 `agents`: the adapter interface with fakes; `herdr` (socket NDJSON), `rex` (`rex send` + `rex send-key enter`, `rex events --json`), `headless` · proved by `TestAdapter_Herdr_PromptOnBlockedAgent`, `TestAdapter_Rex_SendThenEnter`, `TestAdapter_Headless_Resume`
- [ ] 5.2 `specflow`: the markdown and clarify grammar written from the spec flow's `formats.md`; `syncspec` writes the answer in the owner's words, the `D<n> — … → <letter>, … — owner, <date>` line and the `clarify answered` log line · proved by `TestSyncSpec_WritesConfirmedWordsAndLogLine`, `TestSyncSpec_RoundTripsThisFile` (this spec is the golden)
- [ ] 5.2b `watchspec`: a `## Clarify` section with open answers in a watched `docs/specs/*.md` becomes `QuestionAsked` (brief §4) · proved by `TestWatchSpec_OpenClarifyBecomesQuestion`, `TestWatchSpec_IgnoresAnsweredAndAssumed`
- [ ] 5.3 `deliveranswer` with retries and escalation — **must not** put recognized text into the prompt: the prompt is `ink answer <question-id> --json` plus one line of context · proved by `TestDeliver_RetriesThenEscalates`, `TestDeliver_PromptCarriesNoAnswerText`
- [ ] 6.1 `skills/ink/SKILL.md`; page types `critique`, `stamp`, `brief`, `summary`, `inbox` (`00-inbox.pdf` capped at `inbox.max_pages`, default 12), `escalation`, `proposals`; a question flagged `critical: true` is answered only from a `stamp` zone · proved by a golden test per type, `TestSkill_ForbidsAskUserQuestion`, `TestStamp_FreeTextIsNotAnApproval`
- [ ] 6.2 End-to-end MVP without devices · proved by `make e2e`

| Behaviour | Where | Command | Result |
|---|---|---|---|
| Ink Hub's Go packages and the worker pass their tests and lint; no upstream name in the tree | local | `make test && make lint && make provenance` | — |
| A fake agent's `ink ask decision.json` ends with the answer in the spec file (`clarify answered`) and a `prompt` on the fake herdr that carries no answer text, through a WebDAV upload from `testdata/` and the stub engine | local | `make e2e` | — |
| A page round-trips on a real Boox (Syncthing folder) and a real Supernote (WebDAV); the latency is in `docs/architecture/latency.md` | devices — the owner starts it | `ink ask testdata/decision.json` and the reader | — |
| The chosen OCR engine reads the owner's Polish samples with CER ≤ 0.15 on text fields and 0 errors on boxes (threshold *assumed*; the owner sets the number) | local, or paid when the engine is an API — the owner starts it | `uv run --project worker inkworker bench worker/samples` | — |

## Clarify

### Q0. Fork dark-factory, or a clean-room rewrite that owes no licence to the upstream author?
- A. **Clean room: a new repository and module with no file from dark-factory; the inventory becomes a map of concepts; Ink Hub's licence is the owner's choice** (recommended).
- B. Fork under PolyForm Noncommercial.
- C. Fork from the last FSL-1.1-MIT commit.
**Answer:** A — owner, 2026-11-01: "Piszemy od zera od razu poprawione i zgodne z dobrymi praktykami. Mono repo, python do ocr, zamiast chrome qdrant albo duckdb."

### Q1. Which Go module path does Ink Hub use?
- A. **The new repository's URL** (chosen; placeholder `github.com/mkubasz/ink-hub` until the URL is given).
**Answer:** A — owner, 2026-11-01: "Tam jest dużo michal franc nazw w repo co jest błędne". Path pending the repository URL.

### Q5. How does the hub call the Python worker?
- A. **A process per job with a filesystem contract: `inkworker recognize <page-dir>` reads `page.json`, `zones.json` and the ink file, writes into `.run-<ulid>/` and renames to `crops/`, `ink/`, `ocr.json` on success, prints one JSON status line, exits; `inkd` enforces a timeout** (recommended) — no second daemon, testable from `testdata/`, the stub engine is a flag. Cost: a real engine loads its model per job; unmeasured until Phase 4 (vision-language candidates of 1–9 B parameters: tens of seconds on a laptop CPU is plausible), mitigated later by `inkworker serve` behind the same contract.
- B. **A long-running worker service on localhost (HTTP)** — warm models. Cost: two daemons, a protocol and health checks before any model is chosen.
- C. **The worker tails the event log and writes its own events** — decoupling. Cost: a second writer, against D3.
**Answer:** —

### Q9. The index: is "zamiast chrome qdrant albo duckdb" read right?
The architect reads it as "no SQLite index; DuckDB or Qdrant instead" ("chrome" taken as the embedded store, not the browser, which Ink Hub never used).
- A. **DuckDB over the JSONL files for analytics (the CER harness, the dataset, reports), the daemon's projections in memory rebuilt from the log, Qdrant deferred to Phase 4 (crop similarity, with DuckDB's `vss` extension tried first)** (recommended) — no database inside `inkd`, one analytical engine that reads the log as it is. Cost: `duckdb-go` is CGO if `ink` ever needs it in process; until then DuckDB lives in the worker and the CLI shells out.
- B. **DuckDB inside `inkd` as the index** (`github.com/duckdb/duckdb-go`) — SQL over projections in the daemon. Cost: CGO in the daemon; at kilobytes of events it buys nothing over memory.
- C. **Qdrant as the store** — a vector database for an inbox is the wrong tool; it would hold embeddings of crops only.
**Answer:** —

### Q10. Which licence does Ink Hub carry?
- A. **MIT** (recommended) — the shortest, the one the samber skills and most Go tooling use; no patent clause.
- B. **Apache-2.0** — an explicit patent grant and a NOTICE mechanism; longer.
**Answer:** —

### Operational
- The repository URL (sets the module path; enables the first push): —

### Settled by the architect (brief §9), open to objection
- **D3 single writer**: only `inkd` appends; `ink` talks to it over a Unix socket and fails fast (exit 3) when the daemon is down (ADR 0005).
- **D4 ULID**; QR payload uppercase `INK:<page-ulid>:<rev>` (ADR 0005).
- **D7 gate**: `make test`, `make lint`, `make provenance` on every commit; no legacy tier exists (ADR 0003).
- **Samples and the handwriting dataset are private** by default; retention until the owner deletes (ADR 0002).
- **Spec ID prefix `INK`**; the spec flow is dogfooded in this repository.

## Decisions
- D0 — Fork or clean room? → A, clean room, one monorepo, Python for OCR — owner, 2026-11-01
- D1 — Module path? → A, the repository's URL; placeholder until given — owner, 2026-11-01
- D2 — Layout? → monorepo per ADR 0002 — settled by D0 and the owner's "mono repo", 2026-11-01
- D3 — Who writes the log? → A, only `inkd` — assumed, architect, 2026-11-01
- D4 — Identifiers? → A, ULID; QR payload uppercase — assumed, architect, 2026-11-01
- D7 — Gate scope? → A, test + lint + provenance on every commit — assumed, architect, 2026-11-01
- D9 — Index? → DuckDB or Qdrant instead of SQLite — owner, 2026-11-01 ("zamiast chrome qdrant albo duckdb"); the reading and the split (Q9) pending

## Log
- 2026-11-01 09:50 — spec opened in the dark-factory clone (Phase 0 research)
- 2026-11-01 10:40 — architect: work written (inventory, ADRs 0001–0006, NOTICE draft)
- 2026-11-01 11:20 — owner, ahead of the round: "Tam jest dużo michal franc nazw w repo co jest błędne, poza tym w głównym jest tyle kodu i nie ma pkg" — Q1 answered
- 2026-11-01 11:40 — owner: "Zrobimy poprawy kodu i refaktor by nie zostało potrzeby trzymania licencji do Michała" — Q0 added; the licence history traced
- 2026-11-01 12:05 — critique settled: 3 blocks, 16 should, 7 note; the round cut; D3, D4, D6, D7, D8 taken as assumed
- 2026-11-01 12:30 — clarify answered: Q0 = A — owner: "Piszemy od zera od razu poprawione i zgodne z dobrymi praktykami. Mono repo, python do ocr, zamiast chrome qdrant albo duckdb"; the spec moved to the new monorepo; ADRs 0002–0004 rewritten; the fork-only tasks, Q2, D6 and D8 dropped; Q9 (index reading) and Q10 (licence) added
