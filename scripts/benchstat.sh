#!/usr/bin/env bash
# benchstat.sh — interleaved baseline-vs-head benchmark comparison
#
# Interleaving rounds avoids thermal-drift false regressions: both binaries
# run in the same thermal/clock-frequency window each round.
#
# Usage: scripts/benchstat.sh <BenchRegex> <baseRef> [rounds]
#
# Examples:
#   scripts/benchstat.sh BenchmarkLogs_1024 main
#   scripts/benchstat.sh 'BenchmarkRTB|BenchmarkIoT' main 10
#   scripts/benchstat.sh BenchmarkOTLP_4x512 HEAD~3 5
#
# Requirements: golang.org/x/perf/cmd/benchstat (go install or in PATH)

set -euo pipefail

BENCH="${1:?Usage: benchstat.sh <BenchRegex> <baseRef> [rounds]}"
BASE_REF="${2:?Usage: benchstat.sh <BenchRegex> <baseRef> [rounds]}"
ROUNDS="${3:-10}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SCRIPT_DIR}/.."
BENCH_DIR="${REPO_ROOT}/bench"

# Require benchstat
if ! command -v benchstat &>/dev/null; then
    echo "ERROR: benchstat not found. Install with:"
    echo "  go install golang.org/x/perf/cmd/benchstat@latest"
    exit 1
fi

echo "==> Building head binary (current worktree)"
cd "${BENCH_DIR}"
go test -c -o /tmp/qdf_bench_head.test .
echo "    /tmp/qdf_bench_head.test built"

echo "==> Building base binary from ${BASE_REF}"
# Create a temp dir, check out base ref into it, build, then clean up.
TMPDIR="$(mktemp -d /tmp/qdf_base.XXXXXXXX)"
trap 'rm -rf "${TMPDIR}"' EXIT

git -C "${REPO_ROOT}" archive "${BASE_REF}" | tar -x -C "${TMPDIR}"

# bench/go.mod replaces the root module; keep the local replace pointing at the
# extracted root so the base build uses the same qdf version as its own tree.
cd "${TMPDIR}/bench"
# Rewrite the replace directive to point at the extracted root.
go mod edit -replace "github.com/alex60217101990/qdf=${TMPDIR}"
go test -c -o /tmp/qdf_bench_base.test .
echo "    /tmp/qdf_bench_base.test built"

echo ""
echo "==> Running ${ROUNDS} interleaved rounds, bench='${BENCH}'"
echo "    (base=${BASE_REF}, benchtime=1s per round)"

BASE_OUT=/tmp/qdf_benchstat_base.txt
HEAD_OUT=/tmp/qdf_benchstat_head.txt
: >"${BASE_OUT}"
: >"${HEAD_OUT}"

for i in $(seq 1 "${ROUNDS}"); do
    echo "  round ${i}/${ROUNDS}"
    /tmp/qdf_bench_base.test -test.run='^$' -test.bench="${BENCH}" \
        -test.benchmem -test.benchtime=1s >>"${BASE_OUT}" 2>/dev/null
    /tmp/qdf_bench_head.test -test.run='^$' -test.bench="${BENCH}" \
        -test.benchmem -test.benchtime=1s >>"${HEAD_OUT}" 2>/dev/null
done

echo ""
echo "=== benchstat: base (${BASE_REF}) vs head ==="
benchstat "${BASE_OUT}" "${HEAD_OUT}"

echo ""
echo "Raw results saved to:"
echo "  base: ${BASE_OUT}"
echo "  head: ${HEAD_OUT}"
