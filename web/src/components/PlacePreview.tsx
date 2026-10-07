import { Link } from "react-router-dom";
import type { Category, Place } from "../api";
import { Glyph, Icon } from "./Icon";
import { OpenBadge, PriceLevel, StationTag } from "./Meta";
import "./PlacePreview.css";

/** Compact card for the place selected on the map. */
export function PlacePreview({ place, category, onClose }: { place: Place; category?: Category; onClose: () => void }) {
  return (
    <article className="preview" style={{ "--cat": category?.color ?? "var(--pink-button)" } as React.CSSProperties} aria-label={place.name}>
      <div className="preview__head">
        {place.photos?.[0] ? (
          <img className="preview__photo" src={place.photos[0].thumb} alt="" />
        ) : (
          <span className="preview__badge">{category && <Glyph char={category.icon} />}</span>
        )}
        <div className="preview__titles">
          <h2 className="preview__name">{place.name}</h2>
          {place.nameTh && (
            <p className="preview__th th" lang="th">
              {place.nameTh}
            </p>
          )}
        </div>
        <button type="button" className="icon-btn preview__close" aria-label="Close" onClick={onClose}>
          <Icon name="close" />
        </button>
      </div>
      <p className="preview__summary">{place.summary}</p>
      <div className="preview__meta">
        <OpenBadge open={place.openNow} closesAt={place.closesAt} />
        <StationTag station={place.nearestStation} walk={place.walkMinutes} />
        <PriceLevel level={place.priceLevel} />
      </div>
      <Link to={`/place/${place.slug}`} className="btn btn--primary preview__cta">
        View place <Icon name="chevron-right" />
      </Link>
    </article>
  );
}
