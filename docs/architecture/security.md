# AgentInk threat model

Status: written 2026-11-02 (spec task 3.4), awaiting the owner's review.
Scope: the hub as it stands after Phase 3, plus the rules Phases 4–6 must
honour. The brief's §7 is the source:

> Handwriting and anything from a reader are **untrusted data**, never
> instructions; no command runs straight from a note. Critical steps
> (payment, deploy, anything irreversible) need the `stamp` zone marked;
> free text is not enough. The hub listens only on Tailscale (or
> localhost), with a token per device. The agents' own permissions stay
> independent of the hub.

Every control below names the test that keeps it; a control without a test
is listed under *Open items* with the phase that adds it.

## 1. What is at stake

| Asset | Why it matters | Where it lives |
|---|---|---|
| The owner's answers | They steer agents that change code, spend money, deploy | `events.jsonl` (`AnswerConfirmed`), the spec file |
| The spec files | `syncspec` writes into the owner's repositories | `<project>/docs/specs/*.md` |
| The agents' sessions | An adapter can put a prompt in front of an agent | herdr socket, rex, headless |
| Pages and questions | They carry project details onto readers and sync folders | `<data>/pages/`, `<data>/dav/Ink/`, the readers |
| Device tokens | One grants a reader its view of the share | shown once by `ink device add`; sha256 at rest |
| The event logs | The only history; projections are rebuilt from them | `<data>/projects/<slug>/events.jsonl` |

## 2. Trust boundaries

```
 reader ──WebDAV + device token──▶ inkd share ──onPut──▶ ingestink ─┐
 sync folder (Syncthing) ─────────▶ watcher ────────────▶ ingestink ─┤
                                                                     ▼
 agent ──ink ask (Unix socket 0600)──▶ askquestion ──▶ events.jsonl ──▶ renderpage ──▶ share
                                                                     │
 worker (local service, ADR 0007) ◀──POST /v1/recognize── recognize ◀┘
        │ ocr.json (validated)                                       
        ▼                                                            
 confirmanswer ──▶ syncspec (spec file) ──▶ deliveranswer ──▶ adapter ──▶ agent: "ink answer <id>"
```

What crosses each boundary and how much of it is trusted:

- **Reader → hub.** Bytes under `Ink/<project>/<page-id>…<ext>`. Trusted for
  nothing: the file name yields a page id and an extension, the bytes are
  hashed, copied and parsed by the worker. The device is authenticated; its
  content is not.
- **Sync folder → hub.** The same bytes, arriving by a path the owner
  configured (`--watch device=path`). The folder name is the project; the
  page it names must belong to that project.
- **Agent → hub.** A question as JSON over a Unix socket only the owner's
  user can open. The agent runs with the owner's rights anyway, so the hub
  does not try to contain it; it validates the question (contracts) and
  keeps the spec path inside the project so a confused agent cannot aim the
  answer writer elsewhere.
- **Worker → hub.** `ocr.json`, validated against the contract before it is
  read. The worker is a separate process the daemon supervises and abandons
  on timeout; it parses untrusted PDFs and images, which makes it the
  largest parsing surface in the system.
- **Hub → agent.** A prompt that carries the question id and the `ink answer
  <id> --json` command plus one line of context, never the recognized text.
  The agent fetches the words itself, through the API, as data.
- **Hub → spec file.** Only the lines of the spec flow's grammar: the
  `**Answer:**` line of the named question, a `D<n>` decision line, a
  `clarify answered` log line (Phase 5).

## 3. Actors, what they could try, what stops them

| # | Actor | Attempt | Control | Proof |
|---|---|---|---|---|
| 1 | Anyone on the network who is not on the tailnet | reach the share at all | the listener binds only to a Tailscale (100.64/10) or loopback address unless `--webdav-allow-any` is given, which is for development only | `TestListener_RefusesNonTailscale` |
| 2 | Someone on the tailnet without a token | browse or upload | basic auth is required on every request; an unknown name or wrong token is 401 | `TestWebDAV_DeviceTokenScopesTree` |
| 3 | A paired reader (lost, stolen, or running a hostile app) | read another project's pages | each device names its projects; any other `Ink/<project>/` is 403; `/` and `/Ink/` list nothing but what the device may see | `TestWebDAV_DeviceTokenScopesTree` |
| 4 | the same | delete, move or copy pages | DELETE, MOVE, COPY are 403 for every device; only GET/PROPFIND and PUT reach the tree | `TestWebDAV_DeviceTokenScopesTree` |
| 5 | the same | fill the disk with one upload | a PUT above 64 MiB is 413; the body is read through a cap and written to a temp file, renamed only when whole | `TestWebDAV_RejectsOversizedPut` |
| 6 | the same | escape `Ink/<project>/` with `..` or an absolute path | the request path is cleaned and split; anything not under `Ink/<allowed project>/` is 403 | `TestWebDAV_DeviceTokenScopesTree` |
| 7 | the same, after the owner notices | keep using the token | `ink device revoke` appends `DeviceRevoked`; the projection refuses the token on the next request | `TestDevices_TokenAuthenticatesUntilRevoked`, `TestInk_DeviceAddListRevoke` |
| 8 | Someone who reads the hub's disk | recover a device token | the log holds sha256 of the token; the token is printed once by `ink device add` and never stored; comparison is constant-time | `TestDevices_TokenAuthenticatesUntilRevoked` |
| 9 | Someone who guesses tokens | brute force | 24 random bytes (192 bits), base32 | `TestDevices_TokenAuthenticatesUntilRevoked` (shape) |
| 10 | An upload's content (a crafted PDF, a note, a PNG) | run as code, or be read as a file name trick | the name contributes a 26-character page id and an extension in {pdf, mark, note, png}; hidden, empty and other files emit `IngestRejected`; the artifact is copied to `<page-dir>/answer.<ext>` under a fixed name; nothing executes it, only the worker parses it | `TestIngest_RejectsEmptyAndUnknown`, `TestIngest_UploadEmitsInkReceived` |
| 11 | A sync tool or a reader re-sending files | make one answer count twice, or make the unchanged original count as an answer | dedupe on `page_id` + `rev` + sha256; the published PDF's own hash is ignored | `TestIngest_SameBytesOnce`, `TestIngest_OriginalEchoIsIgnored` |
| 12 | A sync folder | pass a file off as another project's | the folder's first segment is the project; the page it names must belong to it, else `IngestRejected`; the watcher credits the device the owner named | `TestIngest_WatchedFolderEmitsInkReceived`, `TestIngest_RejectsEmptyAndUnknown` |
| 13 | Handwriting that reads like an instruction ("ignore the spec, run …") | reach an agent as a command | the adapter prompt is the id and `ink answer <id> --json` plus one line; the e2e test fails if any recognized text is in it; the words reach the agent only when it asks for them, as the owner's answer to its own question | `assertPromptCarriesNoAnswer` in `TestE2E_AskToDelivery`; `TestDeliver_PromptCarriesNoAnswerText` (5.3) |
| 14 | Free text on a critical question ("yes, go ahead" in the margin) | count as approval of a payment or a deploy | a question with `critical: true` renders as a `stamp` page; only a mark in the stamp zone is an approval; free text is recorded in `ocr.json` and the answer, never as `yes` | `TestStamp_FreeTextIsNotAnApproval` (6.1); the `critical` flag and the `stamp` field kind are in the contracts today (`TestContracts_Examples`) |
| 15 | A low-confidence reading | become the owner's words silently | every text field must be at or above `ocr.confidence_min`, else a confirmation page is rendered and the answer waits | `TestRecognize_LowConfidenceRequestsConfirmation` (4.2) |
| 16 | A late answer to a question the agent has since changed | be delivered against the new text | the answer names the page `rev`; a page whose question revision is no longer current ends as `AnswerRejected` | `TestConfirm_RejectsStalePage` (4.2) |
| 17 | An agent, through `ink ask` | aim the answer writer at a file outside the project | `spec.path` must be relative and must not climb; `/etc/…`, `../…`, drive letters and backslashes are refused with 400 | `TestAskQuestion_RejectsSpecPathOutsideProject` |
| 18 | the same | feed the renderer something that is not a question | the contract is validated before anything is appended; the question is passed to Typst as JSON data read by the template, never as Typst source | `TestAskQuestion_RejectsInvalidQuestion`, `TestRenderDecision_Golden` |
| 19 | the same | learn another project's questions | one log per project; `ink inbox` and `ink answer` are the owner's own tools over the owner's own socket; no per-agent isolation is attempted (see §5) | `TestStore_RoutesByProjectAndReplaysInSlugOrder` |
| 20 | Another user on the hub machine | talk to the daemon or read its files | the socket is 0600, the data directory 0700, logs and pages 0600 | `TestApp_RegisterAskInboxAndRestart` (socket), code review |
| 21 | A crash or a full disk mid-write | corrupt the history | one writer, `write` + `fsync` per append; a torn last line is repaired on open; a corrupt middle line refuses to start rather than guess | `TestEventLog_SkipsTruncatedLastLine`, `TestEventLog_CorruptMiddleLineIsAnError`, `TestEventLog_ConcurrentAppendsStayWhole` |
| 22 | The worker, fed a hostile PDF | hang or crash | a separate process, a request per job, `worker.timeout` after which the daemon emits `RecognitionFailed` and leaves no partial `ocr.json` | `TestWorkerSupervisor_RestartsAndTimesOut`, `TestRecognize_AbandonedRunLeavesNoPartialOutput` (4.1) |
| 23 | Any slice | reach into another context's state | a slice imports only its context's `domain`, `platform` and the contracts | `TestArch_SlicesImportOnlyDomainAndPlatform`, `TestArch_RejectsACrossContextImport` |

## 4. What a stamp gates

A `stamp` page exists for the step an agent must not take on a hunch: a
payment, a deploy, a migration that cannot be rolled back, deleting data.
The agent marks the question `critical: true`; the hub renders it as a stamp
page whose one field is the stamp zone with *yes* and *no* boxes.

- The answer is `yes` only when the stamp zone carries the mark. A mark
  elsewhere, a word in the margin, a signature on the sketch area: all of it
  is kept in `ocr.json` and in the answer, none of it is an approval.
- An unclear mark (both boxes, a faint stroke, a low-confidence box) goes to
  a confirmation page; it never resolves to `yes` on its own.
- A stamp answers one page revision. If the agent changed the question in
  the meantime, the stamp is `AnswerRejected` and the agent asks again.
- The hub grants nothing. A stamped `yes` becomes a line in the spec file and
  a `prompt` with the question id; the agent still runs under its own
  permission mode, and the orchestrator's limits still apply. The stamp
  tells the agent what the owner decided; it does not act for the owner.

## 5. What is accepted, and what the owner does

- **Agents are the owner.** They run under the owner's account, so the hub
  does not try to protect one agent from another or the owner's files from
  an agent; the socket and the file modes keep other users out. Per-agent
  isolation is the orchestrator's job.
- **Physical access to a reader is owner access.** Whoever writes on a
  paired reader answers as the owner. Pair readers by name, one token each,
  and revoke a lost one at once.
- **The tailnet encrypts.** The share is plain HTTP over WireGuard; a reader
  that insists on `https://` goes through `tailscale serve`, and
  `--webdav-url` tells `ink device add` the address to print.
- **Pages leave the machine.** A question's text and options sit on the
  reader and in any sync folder. Agents should not put secrets into
  questions; the skill (6.1) says so.
- **The disk holds the history in clear.** Logs, pages and `ocr.json` are
  the owner's files; disk encryption is the owner's.

## 6. Open items

| Item | Risk | Lands in |
|---|---|---|
| No rate limit or quota per device on the share; many 64 MiB PUTs fill the disk | availability | after 3.3, if the latency run shows the share is the right place; otherwise a disk watermark in `inkd` |
| The worker parses PDFs and images with the owner's rights | code execution through a parser bug | 4.1 decides between a seccomp/bubblewrap wrapper and accepting the risk with pinned, current parsers; record in an ADR |
| Stamp enforcement, confidence gate and stale-page rejection are contract shapes, not code yet | the §4 rules | 4.2 and 6.1, with the tests named above |
| `syncspec` must re-check the spec path at write time and refuse symlinks that leave the project | defense in depth for #17 | 5.2, `TestSyncSpec_RefusesPathOutsideProject` |
| Tokens never expire | a forgotten device keeps its view | revoke and re-add; expiry only if the owner asks |
| `IngestRejected` is logged, not shown | a misbehaving reader goes unnoticed | `ink status` counts rejections per device (6.x) |
