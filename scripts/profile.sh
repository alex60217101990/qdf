#!/usr/bin/env bash
# profile.sh — pprof harness for qdf bench
#
# Usage: scripts/profile.sh <BenchRegex> [build-tags]
#
# Examples:
#   scripts/profile.sh BenchmarkLogs_1024
#   scripts/profile.sh 'BenchmarkRTB_1024$'
#   scripts/profile.sh BenchmarkIoT_32x256 ""
#
# Output profiles:
#   /tmp/qdf_cpu.out   — CPU profile
#   /tmp/qdf_mem.out   — heap/alloc profile
#
# After the run the script prints the top-25 CPU and alloc_space nodes.
# To explore interactively:
#   go tool pprof /tmp/qdf_cpu.out      (then: top / web / list FuncName)
#   go tool pprof -sample_index=alloc_space /tmp/qdf_mem.out

set -euo pipefail

BENCH="${1:?Usage: profile.sh <BenchRegex> [tags]}"
TAGS="${2:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/../bench"

echo "==> profiling bench='${BENCH}' tags='${TAGS}'"
echo "    working dir: $(pwd)"

go test \
    ${TAGS:+-tags "${TAGS}"} \
    -run='^$' \
    -bench="${BENCH}" \
    -benchmem \
    -cpuprofile=/tmp/qdf_cpu.out \
    -memprofile=/tmp/qdf_mem.out \
    -benchtime=3s \
    .

echo ""
echo "=== CPU top ==="
go tool pprof -top -nodecount=25 /tmp/qdf_cpu.out

echo ""
echo "=== alloc_space top ==="
go tool pprof -top -nodecount=25 -sample_index=alloc_space /tmp/qdf_mem.out

echo ""
echo "Profiles saved to /tmp/qdf_cpu.out and /tmp/qdf_mem.out"
echo "Interactive exploration:"
echo "  go tool pprof /tmp/qdf_cpu.out"
echo "  go tool pprof -sample_index=alloc_space /tmp/qdf_mem.out"
