import { SITE_NAME } from "../config";
import "./Logo.css";

const RAYS = 11;

/** Rising-sun mark: tapered rays fanning over a half sun. */
export function SunriseMark({ size = 40 }: { size?: number }) {
  const rays = Array.from({ length: RAYS }, (_, i) => {
    const a = Math.PI + (Math.PI * (i + 0.5)) / RAYS;
    const pt = (r: number, da = 0) => `${32 + r * Math.cos(a + da)},${32 + r * Math.sin(a + da)}`;
    const long = i % 2 === 0 ? 31 : 26;
    return <polygon key={i} points={`${pt(15, -0.09)} ${pt(long)} ${pt(15, 0.09)}`} />;
  });
  return (
    <svg className="sunrise" viewBox="0 0 64 34" width={size} height={(size * 34) / 64} aria-hidden>
      <g className="sunrise__rays">{rays}</g>
      <path className="sunrise__sun" d="M20 32 a12 12 0 0 1 24 0 z" />
      <rect className="sunrise__ground" x="10" y="32" width="44" height="2" rx="1" />
    </svg>
  );
}

export function Logo({ compact = false }: { compact?: boolean }) {
  return (
    <span className="logo">
      <SunriseMark size={compact ? 34 : 44} />
      <span className="logo__word">{SITE_NAME}</span>
    </span>
  );
}
