// The decision page: options with a box each (the recommendation first), a
// line for "otherwise" and a sketch area (brief §4).
#import "lib.typ": *

#let (page, content, size) = inputs()
#show: doc => setup(page, size, doc)

// Zones need absolute lengths (their coordinates are read back in mm).
#let cw = size.w * 1mm - 28mm
#let sketch-h = calc.max(40mm, size.h * 1mm - 170mm)

#header(page, content)

#let choice = page.fields.find(f => f.id == "choice")

#text(size: 11pt, fill: luma(80), content.ui.at("choice"))
#v(1mm)
#for opt in choice.options {
  grid(
    columns: (9mm, 1fr), column-gutter: 3mm, row-gutter: 0pt, align: (center + top, left + top),
    zone("choice-" + opt.key, "option", 8mm, 8mm, field: "choice", option: opt.key),
    {
      text(size: 15pt, weight: 700, opt.key + ". ")
      text(size: 15pt, opt.label)
      if opt.at("recommended", default: false) { text(size: 10pt, fill: luma(80), " (" + content.ui.at("recommended") + ")") }
      let why = opt.at("why", default: "")
      let cost = opt.at("cost", default: "")
      if why != "" or cost != "" {
        linebreak()
        text(size: 10.5pt, fill: luma(60), why + if why != "" and cost != "" { " · " } else { "" } + if cost != "" { content.ui.at("cost") + ": " + cost } else { "" })
      }
    },
  )
  v(2.5mm)
}

#v(2mm)
#text(size: 11pt, fill: luma(80), content.ui.at("otherwise") + ":")
#v(1mm)
#zone("otherwise", "field", cw, 28mm, field: "otherwise")

#v(4mm)
#text(size: 11pt, fill: luma(80), content.ui.at("sketch") + ":")
#v(1mm)
#zone("sketch", "field", cw, sketch-h, field: "sketch")
