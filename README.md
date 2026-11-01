# AgentInk

A local hub for answering coding agents' questions by hand, on e-ink readers
(Onyx Boox, Supernote), across many projects at once.

An agent asks (`ink ask`), the hub renders the question as a page and
publishes it over WebDAV, the owner writes on the reader, the hub reads the
ink back, confirms what it is not sure of, writes the answer into the spec
file in the owner's words and delivers it to the right agent.

- Architecture: `docs/adr/`, starting with [0001](docs/adr/0001-bounded-contexts-events-and-vertical-slices.md).
- Where the ideas come from, and what was not taken: `ACKNOWLEDGMENTS.md`, [ADR 0003](docs/adr/0003-clean-room-rebuild-and-provenance.md), `docs/architecture/inventory.md`.
- The plan and the open decisions: `docs/specs/INK-1-agentink-foundation.md`.

Licence: all rights reserved (see `LICENSE`).

Status: Phase 1 (platform, contracts). Nothing runs yet.
