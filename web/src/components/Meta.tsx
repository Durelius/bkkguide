import { Icon } from "./Icon";
import "./Meta.css";

export function OpenBadge({ open, closesAt }: { open: boolean; closesAt?: string }) {
  if (!open) return <span className="chip chip--closed">closed now</span>;
  return (
    <span className="chip chip--open">
      <span className="chip__dot" aria-hidden />
      open{closesAt ? ` until ${closesAt === "00:00" ? "midnight" : closesAt}` : ""}
    </span>
  );
}

/** ฿ to ฿฿฿฿, with the unused signs faded. */
export function PriceLevel({ level }: { level: number | null }) {
  if (!level) return null;
  return (
    <span className="chip chip--price" aria-label={`Price level ${level} of 4`}>
      <span>{"฿".repeat(level)}</span>
      <span className="chip__faint">{"฿".repeat(4 - level)}</span>
    </span>
  );
}

export function StationTag({ station, walk }: { station: string; walk: number | null }) {
  if (!station) return null;
  return (
    <span className="chip">
      <Icon name="train" />
      {station}
      {walk != null && <span className="chip__sub">{walk} min</span>}
    </span>
  );
}
