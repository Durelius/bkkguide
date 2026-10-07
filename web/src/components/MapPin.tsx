import type { Category } from "../api";
import { Glyph } from "./Icon";
import "./MapPin.css";

/** Marker markup; MapLibre mounts the same element as an HTML marker. */
export function MapPin({ category, label, selected = false }: { category: Category; label: string; selected?: boolean }) {
  return (
    <span className={`map-pin${selected ? " map-pin--selected" : ""}`} style={{ "--cat": category.color } as React.CSSProperties}>
      <Glyph char={category.icon} className="map-pin__glyph" />
      {selected && <span className="map-pin__label">{label}</span>}
    </span>
  );
}
