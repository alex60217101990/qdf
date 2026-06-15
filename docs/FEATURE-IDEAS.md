# qdf — Differentiating Feature Ideas

Features no mainstream wire format ships natively, that real Go developers
need, and that lean on qdf's existing strengths (columnar transpose, Select /
predicate pushdown + filter materialization, intern table, decode arena,
codegen, streaming).

Status: ideation. Pick one and run it through `brainstorming` (requirements +
design) before writing code. Captured 2026-06-14.

---

## 1. Structural delta / patch encoding ⭐ (top pick)

**Status: Phase 1 SHIPPED** (2026-06-15) — Diff/Apply for structs (incl. nested), scalars/string/[]byte/[N]byte, positional slices/arrays, per-key maps with tombstones; schema+base fingerprints; optional rANS post-pass. See docs/DELTA.md. Phase 2 pending: columnar column-level diff, keyed slice diff, content-addressed baseline registry.

**What.** `Marshal(new, qdf.Baseline(old))` emits a wire blob carrying ONLY the
fields / columns / rows that differ from a baseline. `Unmarshal` applies the
patch onto a base value to reconstruct `new`. Optionally a standalone
`qdf.Diff(old, new) → patch` / `qdf.Apply(base, patch) → new` pair.

**Go-dev pain.** Developers hand-roll structural diffs everywhere:
- Kubernetes-style watch / resource reconciliation
- etcd / config hot-reload (ship the change, not the whole config)
- live dashboards, collaborative editing, CRDTs
- game / realtime netcode (send only moved entities)
- DB change-data-capture, event sourcing
`reflect.DeepEqual` is slow and gives no diff; rolling your own delta is
boilerplate + bug-prone (nil vs zero, nested, slices reordering).

**Why no competitor has it.** protobuf / msgpack / flatbuffers / capnproto /
avro have NO native structural delta. FieldMask (protobuf) is manual and
caller-specified, not computed. This is a genuine category gap.

**qdf leverage.** Columnar layout makes column-level diff natural (1 of 20
columns changed → ship 1). The intern table already tracks repeats. The
Select / predicate infra already "touches a subset" of a value. Wire can reuse
state-ref tags to say "field unchanged → reference baseline."

**Rough shape.**
- Diff granularity tiers: whole-field → column (columnar) → row-set.
- Patch wire: a sparse map of `{fieldID/colID/rowIdx → newValue}` + a
  "deleted" set for slices/maps.
- `Apply` must be deterministic and handle nil/zero/length changes correctly.
- Hardest part: slice/map element identity (positional vs keyed diff). Start
  positional (index-aligned), add keyed later.

**Effort.** Large. High payoff. Build incrementally: field-level first, then
columnar, then keyed-collection.

**Risks / open questions.**
- Element identity for slices (reorder = full reship unless keyed).
- Baseline distribution: caller holds `old`, or content-addressed baseline id?
- Interaction with compression tiers (diff before or after rANS?).

---

## 2. Canonical / deterministic encoding (cheapest differentiator)

**What.** A guarantee (a tier / option, e.g. `OptCanonical`) that the same
logical value always serializes to byte-identical output: fixed map-key order,
normalized float (-0.0 → +0.0 decision, NaN canonical bits), no
padding/length-encoding ambiguity, stable struct field order.

**Go-dev pain.**
- Signing payloads (webhooks, JWT-like, API request signing)
- Content-addressable storage / dedup by hash (cache keys, CAS, blob stores)
- Idempotency keys, request fingerprinting
- Blockchain / audit logs
**protobuf is explicitly NOT canonical** ("do not hash serialized protobufs" —
map iteration order, unknown fields). msgpack same. Developers get burned
hashing serialized output and silently breaking on a different machine / lib
version.

**Why no competitor.** protobuf docs warn against it; msgpack/json need an
external canonicalizer (JCS). A wire format that GUARANTEES canonical bytes is
a real differentiator.

**qdf leverage.** qdf already controls map order via shape/intern and float
codecs. Mostly a matter of pinning the remaining sources of nondeterminism and
documenting the guarantee + a fuzz test (`encode(v) == encode(decode(encode(v)))`
and cross-run stability).

**Effort.** Small–medium. Fast win. Strong marketing line.

**Risks.** Must audit EVERY codec for order/representation freedom (intern id
assignment order, MTF state, columnar probe choices must be value-determined,
not insertion/timing-determined). Canonical tier likely disables adaptive
codecs that pick based on sampling.

---

## 3. Aggregation pushdown over columnar

**What.** `qdf.Aggregate(blob, qdf.Count, qdf.Where(level == "error"))` and
SUM / MIN / MAX / AVG over a single columnar field, computed directly on the
encoded column WITHOUT materializing rows into Go structs.

**Go-dev pain.** Telemetry / metrics / analytics over a batch: "count errors",
"p99 latency where service=X", "sum bytes where region=eu". Today: decode the
whole batch, then loop. Parquet / Arrow do pushdown but are not a wire format
for arbitrary Go structs.

**Why no competitor (in this niche).** No general Go struct wire format does
columnar aggregation pushdown. Arrow/Parquet are columnar analytics files, not
RPC/message formats.

**qdf leverage.** Direct extension of the existing Select / predicate eval —
predicate evaluation over `colVals` already exists; add reducers (count/sum/
min/max) that run on the decoded column buffer. Reuses the 3VL nullable
predicate tree already built.

**Effort.** Medium. Natural next step on top of Select. Lower risk than #1.

---

## 4. Schema fingerprint / wire-compat guard

**What.** Optionally embed a compact hash of the schema (field IDs + types) in
the header. The decoder detects an incompatible-schema blob and returns an
explicit error instead of silently misparsing.

**Go-dev pain.** protobuf trusts the developer with field numbers — reuse a
number or change a type and you get silent garbage, not an error. Teams fear
wire breakage on deploy.

**Why no competitor.** Avro carries a schema (heavy, registry needed); protobuf
carries nothing. A lightweight self-check fingerprint is a middle ground nobody
ships by default.

**qdf leverage.** qdfgen already knows the schema at codegen time → can emit a
constant fingerprint; the reflect path can compute it from the typeDesc. Header
already has flag bits.

**Effort.** Small–medium. Pure safety feature; pairs well with #2 (canonical).

---

## Recommendation

- **#1 Structural delta** — most unique, most needed, leans hardest on the
  columnar+Select strengths. Positions qdf as the **state-sync** format.
- **#2 Canonical encoding** — cheapest, real pain, fast differentiator.
  Positions qdf for **signed / content-addressed payloads**.

Together #1 + #2 cover two niches no competitor closes. #3 is the safest
incremental win (extends Select). #4 is a low-cost safety pairing.

Tomorrow: choose one → `brainstorming` skill → design → TDD implementation.
