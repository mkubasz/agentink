---
id: SHP-12
title: Labels at the till
step: architect
status: blocked
mode: session
branch: feature/labels
repo: shop
created: 2026-11-01
updated: 2026-11-01
tags: [spec/SHP-12, step/architect, branch/feature-labels, status/blocked]
---
<!-- spec-card -->

# SHP-12 — Labels at the till

## Summary
Print a shipping label when an order is packed.

## Plan

## Clarify

### Q1. Which printer does the label go to?
- A. **The thermal printer at the till** (recommended) — it is where labels are used. Cost: one driver call.
- B. **Any system printer** — a print dialog. Cost: a layout per printer.
**Answer:** —

## Decisions

## Log
- 2026-11-01 09:00 — spec opened on `feature/labels`
