<!-- Thanks for opening a PR against qdf. -->

## Summary

<!-- 1-3 sentences. What does this change do and why? -->

## Wire-format impact

<!-- Tick exactly one. -->

- [ ] Wire-compatible. Existing buffers still decode; new buffers
      still decode through older readers that ignore unknown tags.
- [ ] Adds a new tag. Old decoders will surface `ErrBadTag` rather
      than mis-decoding. Tag value chosen from the reserved range
      documented in `wire.go`.
- [ ] Breaks the wire. Requires a version bump in `Version1` /
      header byte. Migration plan: <describe>.

## Checklist

- [ ] `go test -race -count=1 ./...` passes locally.
- [ ] Relevant `-tags qdf_simd` and `-tags qdf_reflect2` builds tested
      (skip if change does not touch the affected paths).
- [ ] Golden fixtures regenerated (`go test -run TestGolden -update`)
      if the change is intentional.
- [ ] Bench numbers refreshed in `docs/BENCH.md` if the hot path
      changed measurably.
- [ ] Fuzz corpus + property fuzzers run for ≥30 s if the decoder
      surface area changed.
- [ ] README or `docs/BENCH.md` updated for any user-facing API
      change.

## Linked issues

<!-- Closes #..., Refs #... -->
