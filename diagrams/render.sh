#!/usr/bin/env bash
# Usage: ./diagrams/render.sh   — regenerate all SVGs from the Mermaid sources embedded in diagrams/*.md
# Run from the repo root or from the diagrams/ directory; the script finds its own location.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SVG_DIR="${SCRIPT_DIR}/svg"
PUPPET_CFG="${SCRIPT_DIR}/.puppeteer.json"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

mkdir -p "$SVG_DIR"

# Ensure puppeteer no-sandbox config exists (required on macOS with system Chrome).
if [ ! -f "$PUPPET_CFG" ]; then
    printf '{"args":["--no-sandbox"]}' > "$PUPPET_CFG"
fi

# On macOS, point mmdc at the system Chrome.  On Linux/CI the default
# Chromium bundled with puppeteer is used (PUPPETEER_EXECUTABLE_PATH unset).
if [ "$(uname)" = "Darwin" ] && [ -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then
    export PUPPETEER_EXECUTABLE_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
fi

MMDC=/usr/local/bin/npx
MMDC_ARGS=(-y @mermaid-js/mermaid-cli -p "$PUPPET_CFG" -b transparent)

# Extract every ```mermaid … ``` block from each .md file and render it.
# Output SVGs are named <stem>-<n>.svg where n is 1-based block index per file.
extract_and_render() {
    local md_file="$1"
    local stem
    stem="$(basename "$md_file" .md)"

    python3 - "$md_file" "$stem" "$TMP_DIR" <<'PYEOF'
import sys, re, os

md_path, stem, tmp_dir = sys.argv[1], sys.argv[2], sys.argv[3]
with open(md_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Extract from both raw ```mermaid blocks AND from inside <details> wrappers
blocks = re.findall(r'```mermaid\n(.*?)```', content, re.DOTALL)
for i, block in enumerate(blocks, 1):
    out = os.path.join(tmp_dir, f"{stem}-{i}.mmd")
    with open(out, 'w', encoding='utf-8') as f:
        f.write(block.strip())
    print(f"{i} {out}")
PYEOF
}

RENDERED=0
FAILED=0

for md in "${SCRIPT_DIR}"/README.md \
           "${SCRIPT_DIR}"/architecture.md \
           "${SCRIPT_DIR}"/wire-format.md \
           "${SCRIPT_DIR}"/options-and-modes.md \
           "${SCRIPT_DIR}"/qpack-codecs.md \
           "${SCRIPT_DIR}"/dense-interning.md \
           "${SCRIPT_DIR}"/columnar-and-selective-decode.md \
           "${SCRIPT_DIR}"/performance.md; do

    stem="$(basename "$md" .md)"
    echo "=== $stem ==="

    while read -r idx mmd_path; do
        svg_out="${SVG_DIR}/${stem}-${idx}.svg"
        echo "  rendering ${stem}-${idx}..."
        if "$MMDC" "${MMDC_ARGS[@]}" -i "$mmd_path" -o "$svg_out" 2>&1 | grep -v 'command not found: _encode'; then
            if [ -s "$svg_out" ]; then
                echo "    OK ($(wc -c < "$svg_out") bytes)"
                RENDERED=$((RENDERED + 1))
            else
                echo "    FAILED: empty output"
                FAILED=$((FAILED + 1))
                rm -f "$svg_out"
            fi
        else
            echo "    FAILED: mmdc exited non-zero"
            FAILED=$((FAILED + 1))
            rm -f "$svg_out"
        fi
    done < <(extract_and_render "$md")

done

echo ""
echo "Done: ${RENDERED} SVGs rendered, ${FAILED} failed."
if [ "$FAILED" -gt 0 ]; then
    echo "Failed diagrams keep their Mermaid source as the fallback (no broken <img> tags introduced)."
    exit 1
fi
