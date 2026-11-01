# 0006 — Page rendering with Typst, answer zones from the template

- **Status:** proposed
- **Date:** 2026-11-01
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision point:** settled by the brief; `typst query` deprecation verified

## Context

The brief chooses Typst for rendering and wants the answer zones' coordinates
to come **from the template itself** (one source of truth for the look and for
the OCR crops), read with `typst query` into `zones.json`. The brief also asks
to verify the Typst version and the stability of `typst query`.

Verified on 2026-11-01 with the official `typst 0.15.1 (9dfd3a08)`
linux-musl binary (release 2026-07-17; 0.15.0 is from 2026-06-15):

- `typst query` still works but is **deprecated**: `typst query --help` says
  "(deprecated, use `eval` instead)" and every call prints
  `warning: the 'typst query' subcommand is deprecated` with the hint
  `typst eval 'query(metadata).map(it => it.value)' --in t.typ`.
- `typst eval <expr> --in <file>` returns the same JSON without the warning.
- A zone marked in the template as

  ```typst
  #let zone(id, w, h) = context {
    let pos = here().position()
    [#metadata((zone: id, page: pos.page, x: pos.x.mm(), y: pos.y.mm(),
                w: w.mm(), h: h.mm())) #label("zone-" + id)]
    box(width: w, height: h, stroke: 1pt)
  }
  ```

  yields `{"zone":"A","page":1,"x":13.667125,"y":24.4145583,"w":8.0,"h":8.0}`:
  millimetres from the page's top-left corner, as the OCR pipeline needs.

## Decision

- **Typst 0.15.x, pinned.** `inkd` runs the `typst` binary found on `PATH`
  or at `render.typst_bin`; at start-up it runs `typst --version` and refuses
  a major.minor other than the pinned one (configurable `render.typst_version`,
  default `0.15`). The binary is not vendored into the repository.
- **Render.** `typst compile <page.typ> page.pdf` with the page's data passed
  as `--input page=<json>` (Typst's `sys.inputs`) so a template never embeds
  content by string concatenation.
- **Zones.** `typst eval 'query(metadata).map(it => it.value)' --in page.typ
  --input page=<json>` writes `zones.json`; `typst query` is the fallback
  when `eval` is missing (Typst < 0.15). The result is validated against
  `contracts/zones.schema.json`. Each zone is `{zone, kind, page, x, y, w, h}`
  in mm, plus the QR and the four corner markers as zones of kind `fiducial`.
- **Golden tests.** Each page type has a golden `zones.json` and a golden PDF
  rasterized to PNG at a fixed DPI, compared with a tolerance; the test is
  skipped with a clear message when `typst` is absent, and CI installs it.
- **Fonts.** Chosen in Phase 2: a handwriting-style font with the full Polish
  set (ą ć ę ł ń ó ś ź ż and capitals) under an open licence, shipped in
  `internal/pages/fonts/` and passed with `--font-path`. Stroke synthesis is
  v2.

## Alternatives

- **HTML + headless Chromium** (chromedp is already a dependency of the fork,
  `cmd/playtest`). Rejected: coordinates come from the DOM at render time and
  drift with fonts and viewport; a Chromium per render is heavy on a laptop.
- **A PDF library in Go** (gofpdf/maroto): exact coordinates, but no real
  typesetting of Polish text with hyphenation and no template language.
- **LaTeX.** Heavier toolchain, slower, no clean metadata query.

## Consequences

- `typst` is an external runtime dependency; the install docs and the CI
  image pin 0.15.x. Upgrading Typst is an ADR amendment with the golden
  tests as the proof.
- The deprecation of `typst query` costs nothing today: the `eval` form is
  the primary path.
- `here().position()` is a layout-time value: a zone inside a floating or
  multi-column layout must be checked in the golden; page types keep one
  column.
