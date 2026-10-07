import codepoints from "../icons.json";

export type IconName = keyof typeof codepoints;

/** The Nerd Font glyph for a curated icon (see web/icons.json). */
export function glyph(name: IconName): string {
  return String.fromCodePoint(parseInt(codepoints[name], 16));
}

export const iconNames = Object.keys(codepoints) as IconName[];
