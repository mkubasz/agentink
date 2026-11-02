// Shared pieces of every AgentInk page (ADR 0006). The page document and the
// content come in as JSON through sys.inputs; nothing here concatenates
// agent text into Typst source.

#let inputs() = (
  page: json(bytes(sys.inputs.page)),
  content: json(bytes(sys.inputs.content)),
  size: json(bytes(sys.inputs.size)),
)

#let hand = "Caveat"
#let mono = "DejaVu Sans Mono"

// zone marks an answer area: a box the owner writes in, and a metadata record
// `typst eval` reads back as its coordinates in millimetres (zones.json).
#let zone(id, kind, w, h, field: none, option: none, stroke: 0.8pt, body: none) = box(
  width: w, height: h, stroke: stroke, inset: 0pt, radius: 1.5pt,
  {
    context {
      let pos = here().position()
      let z = (zone: id, kind: kind, page: pos.page, x: pos.x.mm(), y: pos.y.mm(), w: w.mm(), h: h.mm())
      if field != none { z.insert("field", field) }
      if option != none { z.insert("option", option) }
      [#metadata(z) #label("zone-" + id)]
    }
    if body != none { body }
  },
)

// fiducials are four filled squares the worker aligns on, in the paper's
// margins (drawn as the page background, so they never meet the content).
#let fiducials(size: 6mm, inset: 5mm) = {
  let mark(id, ..args) = place(..args, zone(id, "fiducial", size, size, stroke: none, body: rect(width: size, height: size, fill: black, stroke: none)))
  mark("mark-tl", top + left, dx: inset, dy: inset)
  mark("mark-tr", top + right, dx: -inset, dy: inset)
  mark("mark-bl", bottom + left, dx: inset, dy: -inset)
  mark("mark-br", bottom + right, dx: -inset, dy: -inset)
}

// qr draws the module matrix the hub computed (rows of "0"/"1").
#let qr(modules, size: 20mm) = {
  let n = modules.len()
  let m = size / n
  zone("qr", "qr", size, size, stroke: none, body: grid(
    columns: (m,) * n, rows: (m,) * n, gutter: 0pt,
    ..modules.map(row => row.clusters().map(c => rect(width: m, height: m, fill: if c == "1" { black } else { white }, stroke: none))).flatten(),
  ))
}

// setup sets the page to the device's size and the handwriting face; the
// corner marks sit in the background and the payload in the footer margin,
// both outside the content area.
#let setup(pg, size, doc) = {
  set page(
    width: size.w * 1mm, height: size.h * 1mm,
    margin: (x: 14mm, top: 14mm, bottom: 14mm),
    background: fiducials(),
    footer: align(center, text(size: 8pt, font: mono, fill: luma(120))[#pg.qr.payload]),
  )
  set text(font: hand, size: 13pt, lang: "pl")
  set par(leading: 0.55em)
  doc
}

// header: the title, a small id line and the body on the left, the QR on
// the right, in two columns so nothing meets the code.
#let header(page, content) = {
  let body = content.at("body", default: "")
  grid(
    columns: (1fr, 22mm), column-gutter: 4mm, align: (left + top, right + top),
    {
      text(size: 22pt, weight: 700, page.title)
      v(1mm)
      if content.at("critical", default: false) {
        text(size: 10pt, font: mono)[KRYTYCZNE — liczy się tylko stempel]
        v(1mm)
      }
      text(size: 8pt, font: mono, fill: luma(90))[#page.kind · #page.question_id · rev #page.question_rev]
      if body != "" {
        v(3mm)
        text(size: 13pt, body)
      }
    },
    qr(page.qr.modules, size: 22mm),
  )
  v(4mm)
}
