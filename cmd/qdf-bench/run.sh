#!/usr/bin/env bash
# Build and run qdf-bench under every valid qdf build-tag combination.
# Build tags are compile-time, so each combo needs its own binary.
#
# Usage: ./cmd/qdf-bench/run.sh [path-to-adalanche-sampledata] [extra qdf-bench flags]
#   With no path, each binary downloads the sample data to a temp dir itself.
#   Pass a local clone path to run offline / skip re-downloading.
# Env:   GO=/path/to/go   (defaults to `go` on PATH)
set -euo pipefail

# A first arg that is an existing directory is treated as the local datapath;
# anything else (e.g. -iters) is forwarded straight to qdf-bench.
DP=""
if [[ $# -gt 0 && -d "$1" ]]; then
	DP="$1"
	shift
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
GO="${GO:-go}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cd "$REPO_ROOT"

# All valid combinations of qdf's build tags. "" == default (none).
combos=("" "qdf_reflect2" "qdf_simd" "qdf_reflect2 qdf_simd")

for tags in "${combos[@]}"; do
	bin="$TMP/qdf-bench"
	CGO_ENABLED=0 "$GO" build -tags "$tags" -o "$bin" ./cmd/qdf-bench
	if [[ -n "$DP" ]]; then
		"$bin" -datapath "$DP" "$@"
	else
		"$bin" "$@"
	fi
	echo
done
