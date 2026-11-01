# 0007 — The Python worker is a local service that `inkd` supervises

- **Status:** proposed
- **Date:** 2026-11-01
- **Spec:** docs/specs/INK-1-agentink-foundation.md · **Decision point:** D5 (owner)

## Context

The worker does the pixel and model work (parse PDF/`.mark`/`.note`, align,
crop zones, detect boxes, OCR, the CER harness). The owner decided
(2026-11-01): "golang dostaje info o notatce i wywołuje api pythona do
przerobienia. CLI nie musi wiedzieć o tym." So the worker exposes an API, the
daemon calls it, and the CLI never sees it. OCR models are vision-language
models of 1–9 B parameters: loading one per job would cost tens of seconds, so
a process that stays up and keeps the model warm is the right shape.

## Decision

- **`inkworker serve`** is a long-running HTTP service on a Unix socket
  (`$XDG_RUNTIME_DIR/inkworker.sock`; `--listen` for a loopback TCP address
  when a socket is impractical). It never listens on a routable interface.
- **`inkd` supervises it by default:** `worker.command` (default
  `uv run --project <repo>/worker inkworker serve --socket <path>`) is started
  at daemon start, restarted with backoff when it exits, and probed with
  `GET /healthz`. `worker.url` points at an externally managed worker instead
  (a systemd unit, a GPU box on the tailnet is **not** allowed: the socket or
  loopback only, because the worker reads and writes the hub's data directory).
- **One job, one request.** `POST /v1/recognize` with
  `{"page_dir": "...", "device": "<profile>", "engine": "stub|<name>"}`; the
  worker reads `page.json`, `zones.json` and the ink artifact from the page
  directory, writes into `<page-dir>/.run-<ulid>/`, renames `crops/`, `ink/`
  and `ocr.json` into place on success, and answers `200` with the `ocr.json`
  body (validated against `contracts/ocr.schema.json`). The request is
  synchronous; `inkd` runs it from a goroutine with `worker.timeout` (default
  5 min) and emits `AnswerRecognized`, `ConfirmationRequested` or
  `RecognitionFailed`. The worker processes one job at a time; `inkd` queues.
- **`POST /v1/bench`** runs the CER harness over `worker/samples/` and
  returns the report; `GET /v1/engines` lists the engines and their versions.
- **Contracts, not code, cross the boundary:** both sides validate what they
  write and read against `contracts/*.schema.json` (ADR 0008).
- **The CLI knows nothing of the worker.** `ink` talks to `inkd` only;
  `ink status` shows the worker's health as one line because `inkd` reports it.

## Alternatives

- **A process per job** (`inkworker recognize <page-dir>`), the architect's
  first recommendation. Simpler to test, but it loads the model per page;
  rejected by the owner in favour of an API.
- **The worker tails the event log and writes its own events.** A second
  writer, against ADR 0005; rejected.
- **A worker on another machine over the tailnet.** It would need the page
  directory shared over the network and a second trust boundary; not for the
  MVP.

## Consequences

- Python dependencies for the service: FastAPI and uvicorn (both MIT), chosen
  for typed request models that mirror the contracts; recorded in
  `ACKNOWLEDGMENTS.md` when they land (Phase 4).
- A killed or crashed worker leaves `.run-<ulid>/` behind, never a partial
  `ocr.json`; `inkd` sweeps stale `.run-*` directories at start.
- The e2e skeleton (task 1.7) uses a fake worker that speaks the same API
  with the `stub` engine, so the contract is exercised from Phase 1.
