# Security Policy

## Supported Versions

qdf is in `alpha` status (see README). Only the latest `main` branch
receives security updates. Older commits should be considered
unsupported for security purposes.

## Reporting a Vulnerability

Open a **GitHub Security Advisory** for this repository — that is
the private, embargoed channel.

  https://github.com/alex60217101990/qdf/security/advisories/new

Please include:

  1. A minimal reproducer (Go program + input bytes).
  2. The qdf commit SHA, Go version, and `GOARCH` / `GOOS`.
  3. Impact assessment: panic, OOM, memory corruption, denial of
     service, data leak, or other.

We aim to acknowledge advisories within **3 business days** and
ship a fix or workaround within **14 days** for high-severity issues.

## Hard Constraints the Decoder Honours

The following are treated as security invariants. A reproducer that
breaks any of them is a vulnerability:

  - **Never panic on hostile input.** Every prefix of every byte
    sequence — well-formed or not — must surface as a typed error
    from `Unmarshal` / `Decoder.Skip`. Coverage: `truncation_test.go`
    plus `FuzzDecoder_NeverPanics`.
  - **Length prefixes are validated before any `make`.** A payload
    that claims a billion-element array, map, or string returns
    `ErrShortBuffer` rather than attempting the allocation.
    Coverage: `oom_protection_test.go`.
  - **Bounded depth on recursive structures.** Pointer cycles return
    `ErrCycleDetected` instead of stack-overflowing. Default cap is
    `DefaultMaxDepth = 10 000`; configurable via
    `Encoder.SetMaxDepth`.
  - **No silent type confusion.** Tag-byte mismatches return
    `ErrTypeMismatch`; unknown tags return `ErrBadTag`. A decoder
    that does not recognise a new wire feature fails loud.

## Known Limitations

  - The encoder may stack-overflow when given an interface field that
    holds a self-referential value (in addition to the pointer-cycle
    case above). Mitigated by `Encoder.SetMaxDepth` at the price of
    declaring a maximum struct nesting depth.
  - `qdfgen`-generated code uses `unsafe.Pointer` arithmetic for
    field access. A malformed Go input file is rejected at generation
    time, but qdfgen is a build-time tool — do not run it on
    untrusted source code.

## Threat Model

qdf is intended for trusted-or-validated data in transit (RPC, log
shipping, metric pipelines). The wire format does **not** include
cryptographic authentication; downstream callers must wrap the
payload (TLS, signed envelope, etc.) when integrity matters.
