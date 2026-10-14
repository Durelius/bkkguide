"""Widen each icon glyph's advance to fit its ink, so icons center in their boxes.

Nerd Font icons keep the 600-unit monospace advance but draw up to ~1150 units
wide from x=0, so CSS centers the advance box and the icon lands off to the right.
Usage: python3 scripts/center-icons.py web/icons.json web/public/fonts/*.woff2
"""
import json
import sys

from fontTools.pens.boundsPen import BoundsPen
from fontTools.ttLib import TTFont

codepoints = [int(v, 16) for v in json.load(open(sys.argv[1])).values()]
for path in sys.argv[2:]:
    font = TTFont(path)
    glyphs, cmap, hmtx = font.getGlyphSet(), font.getBestCmap(), font["hmtx"]
    for cp in codepoints:
        name = cmap.get(cp)
        if not name:
            continue
        pen = BoundsPen(glyphs)
        glyphs[name].draw(pen)
        if pen.bounds:
            x_min, _, x_max, _ = pen.bounds
            hmtx[name] = (round(x_min + x_max), hmtx[name][1])
    font.save(path)
