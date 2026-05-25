---
name: Feature request
about: Proposed addition to qdf's wire format, codec set, or public API
title: '[feat] '
labels: enhancement
---

## Motivation

<!-- What workload are you trying to encode/decode? What is the
     current cost (wire size, CPU, allocations) and what is the
     target? -->

## Proposed change

<!-- Concrete API or wire-format sketch. New tag value, function
     signature, codec name. -->

## Wire-format impact

- [ ] Wire-compatible (no new tag bytes / no semantic change).
- [ ] Adds a new tag, claims a slot from the reserved range in
      `wire.go`.
- [ ] Breaks the wire and needs a version bump.

## Alternatives considered

<!-- Why this design over the obvious other options? -->
