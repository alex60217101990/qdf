# Continuous benchmarks

Live dashboard: **<https://alex60217101990.github.io/qdf/dev/bench/>**

Every push to `main` runs the Go benchmark suite in CI and appends the
results to a trend dashboard (built with
[`benchmark-action/github-action-benchmark`](https://github.com/benchmark-action/github-action-benchmark)
and published to the `gh-pages` branch). The dashboard is the
always-current source of performance numbers; the tables in
[`BENCH.md`](BENCH.md) are a point-in-time snapshot.

## What the graphs show

- **One chart per benchmark** (e.g. `BenchmarkProfile_HotPath/decode/qdf`,
  `BenchmarkCorpusCodec_*`). The X axis is the commit timeline; the Y axis
  is the metric — `ns/op` (lower is faster), and where reported `B/op` and
  `allocs/op` (lower is leaner).
- **Hover a point** to see the exact commit, value, and date. Click it to
  jump to the commit on GitHub.

## How to read progress vs regression

- **Line trends down → improvement.** A commit that makes a path faster (or
  smaller) drops the curve at that point.
- **Line trends up → regression.** A sustained step up means a change cost
  time/memory. A one-commit spike that immediately returns is almost always
  **runner noise**, not a real regression (see below).
- **Compare the latest point to the recent plateau**, not to a single prior
  point — that filters out the variance.

## Noise — read trends, not single points

CI runs on shared GitHub-hosted runners, so absolute numbers carry roughly
**±10–15 %** run-to-run variance (CPU model, neighbours, thermal). Rules of
thumb:

- A real win/regression shows as a **sustained shift across several
  commits**, not a lone spike.
- The dashboard's auto-alert fires when a commit is more than ~2× slower
  than its baseline; sub-2× changes are below the cross-run noise floor and
  must be confirmed locally with `benchstat` over ≥10 interleaved runs (see
  the "measure-first" discipline in [`BENCH.md`](BENCH.md)).
- Wire **size** numbers (from `TestCorpusCodec_Sizes`) are deterministic —
  no noise — so any change there is real.

## Running the suite locally

```sh
cd bench
go test -run=^$ -bench='BenchmarkProfile_'  -benchtime=2s -count=10 | tee new.txt
benchstat old.txt new.txt          # keep/revert decisions
go test -run TestCorpusCodec_Sizes -v        # deterministic wire sizes
```
