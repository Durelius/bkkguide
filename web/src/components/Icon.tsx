import { glyph, type IconName } from "../icons";

type Props = { name: IconName; label?: string; className?: string };

/** A Nerd Font glyph. Decorative unless a label is given. */
export function Icon({ name, label, className }: Props) {
  return (
    <span
      className={className ? `nf ${className}` : "nf"}
      aria-hidden={label ? undefined : true}
      role={label ? "img" : undefined}
      aria-label={label}
    >
      {glyph(name)}
    </span>
  );
}

/** Raw glyph text stored on a category (from /admin). */
export function Glyph({ char, className }: { char: string; className?: string }) {
  return (
    <span className={className ? `nf ${className}` : "nf"} aria-hidden>
      {char}
    </span>
  );
}
