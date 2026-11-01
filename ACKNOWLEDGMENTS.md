# Acknowledgments

Ink Hub is an independent work. Its ideas were shaped by two projects whose
code is not used here (docs/adr/0003):

- **AI Workflow Engine** by Michal Franc and its fork **dark-factory** by
  Kacper Werema — the clarify round with one recommended answer, the "Needs
  you" list, the gate as a question that needs a stamp, the fresh critic's
  findings, the red-round cap and the post-mortem's proposed actions. Kacper
  Werema agreed to a rebuild of his fork; this repository takes only the
  ideas.
- **The spec flow** (analyst → researcher → architect → developer → qa) by the
  owner of this repository, whose formats `specflow` reads and writes.

Third-party software, with licences:

- `samber/cc-skills-golang` — the Go style skills and the `.golangci.yml`
  in this repository — MIT, Copyright 2026 Samuel Berthe.
- `github.com/oklog/ulid/v2` — ULIDs for events, questions and pages — Apache-2.0.
- `github.com/santhosh-tekuri/jsonschema/v6` — JSON Schema 2020-12 validation in Go — Apache-2.0.
- `jsonschema` (Python) — the same validation in the worker — MIT.
- Go modules and Python packages are listed in `go.mod` and
  `worker/pyproject.toml`; each new one is recorded here with its licence
  when it lands.
