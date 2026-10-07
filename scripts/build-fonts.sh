#!/usr/bin/env bash
# Rebuild the self-hosted fonts in web/public/fonts from JetBrains Mono Nerd Font.
# Run again after adding icons to web/icons.json.
# Needs: python3 with `pip install fonttools brotli`, curl, tar.
set -euo pipefail
cd "$(dirname "$0")/.."

PY=${PYTHON:-python3}
OUT=web/public/fonts
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sSLf -o "$TMP/jbm.tar.xz" https://github.com/ryanoasis/nerd-fonts/releases/latest/download/JetBrainsMono.tar.xz
tar -xf "$TMP/jbm.tar.xz" -C "$TMP"
mkdir -p "$OUT"

# Latin text, punctuation, arrows, box drawing, shapes, and the baht sign.
TEXT="U+0020-007E,U+00A0-00FF,U+0131,U+0152-0153,U+02C6,U+02DA,U+02DC,U+2000-206F,U+20AC,U+0E3F,U+2122,U+2190-21FF,U+2500-259F,U+25A0-25FF"
ICONS=$($PY -c 'import json; print(",".join("U+"+v for v in json.load(open("web/icons.json")).values()))')

for w in Regular Bold ExtraBold; do
  $PY -m fontTools.subset "$TMP/JetBrainsMonoNerdFont-$w.ttf" --unicodes="$TEXT,$ICONS" \
    --flavor=woff2 --layout-features='*' --output-file="$OUT/jbm-nf-$(echo $w | tr A-Z a-z).woff2"
done
ls -lh "$OUT"
