#!/usr/bin/env bash
# compare.sh — capture competitive throughput + memory tables for docs
#
# Runs the full benchmark matrix on all five canonical fixtures and
# TestMemoryReport, writing both outputs to /tmp/qdf_compare.txt.
#
# Usage: scripts/compare.sh [bench-count]
#
# Examples:
#   scripts/compare.sh          # default: -count=6
#   scripts/compare.sh 3        # faster smoke run
#
# The markdown tables for docs/BENCHMARKS.md are derived from this output:
#   - Throughput rows come from the -bench run (ns/op, MB/s, allocs/op)
#   - Memory rows come from TestMemoryReport (bytes/cycle, peak-heap)
# This script does NOT auto-format markdown; it captures raw output so the
# docs phase can cherry-pick and format as needed.

set -euo pipefail

COUNT="${1:-6}"
OUT="/tmp/qdf_compare.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/../bench"

echo "==> qdf competitive comparison  (count=${COUNT}, $(date))"
echo "    working dir: $(pwd)"
echo "    output: ${OUT}"
echo ""

{
    echo "# qdf compare run  $(date)"
    echo "# host: $(uname -snrm)"
    echo "# go:   $(go version)"
    echo ""
    echo "## Throughput benchmarks (ns/op, MB/s, allocs/op, wire-B)"
    echo ""
    go test \
        -run='^$' \
        -bench='RTB|IoT|OTLP|Logs|Events' \
        -benchmem \
        -count="${COUNT}" \
        .
    echo ""
    echo "## Memory report (bytes/cycle, peak-heap per codec per fixture)"
    echo "## NOTE: maxRSS is process-wide, not per-codec. Use bytes/cycle for container sizing."
    echo ""
    EMIT_MEM=1 go test \
        -run=TestMemoryReport \
        -count=1 \
        -timeout=600s \
        .
} 2>&1 | tee "${OUT}"

echo ""
echo "==> done. Full output written to ${OUT}"
echo "    Derive docs tables from:"
echo "      - Throughput: grep -A... the bench section"
echo "      - Memory: the markdown tables printed by TestMemoryReport"
