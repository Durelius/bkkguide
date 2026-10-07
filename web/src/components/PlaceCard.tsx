import { Link } from "react-router-dom";
import type { Category, Place } from "../api";
import { Glyph, Icon } from "./Icon";
import { OpenBadge, PriceLevel, StationTag } from "./Meta";
import "./PlaceCard.css";

type Props = {
  place: Place;
  category?: Category;
  saved?: boolean;
  onToggleSave?: () => void;
  highlighted?: boolean;
  onHover?: (hovering: boolean) => void;
  id?: string;
};

export function PlaceCard({ place, category, saved = false, onToggleSave, highlighted = false, onHover, id }: Props) {
  const color = category?.color ?? "var(--pink-button)";
  const cover = place.photos?.[0];
  return (
    <article
      id={id}
      className={`place-card${highlighted ? " place-card--highlighted" : ""}`}
      style={{ "--cat": color } as React.CSSProperties}
      onMouseEnter={onHover && (() => onHover(true))}
      onMouseLeave={onHover && (() => onHover(false))}
    >
      <div className={`place-card__visual${cover ? " place-card__visual--photo" : ""}`}>
        {cover ? (
          <img src={cover.thumb} alt={cover.alt} loading="lazy" className="place-card__img" />
        ) : (
          category && <Glyph char={category.icon} className="place-card__glyph" />
        )}
        {place.featured && (
          <span className="place-card__featured">
            <Icon name="star" /> editors' pick
          </span>
        )}
        <div className="place-card__tools">
          <Link
            to={`/?place=${encodeURIComponent(place.slug)}`}
            className="place-card__tool"
            aria-label={`Show ${place.name} on the map`}
            title="Show on map"
          >
            <Icon name="map-marker" />
          </Link>
          <button
            type="button"
            className="place-card__tool place-card__save"
            aria-pressed={saved}
            aria-label={saved ? `Remove ${place.name} from saved` : `Save ${place.name}`}
            onClick={onToggleSave}
          >
            <Icon name={saved ? "heart" : "heart-outline"} />
          </button>
        </div>
      </div>
      <div className="place-card__body">
        <h3 className="place-card__name">
          <Link to={`/place/${place.slug}`}>{place.name}</Link>
        </h3>
        {place.nameTh && (
          <p className="place-card__th th" lang="th">
            {place.nameTh}
          </p>
        )}
        <p className="place-card__summary">{place.summary}</p>
        <div className="place-card__meta">
          <OpenBadge open={place.openNow} closesAt={place.closesAt} />
          <StationTag station={place.nearestStation} walk={place.walkMinutes} />
          <PriceLevel level={place.priceLevel} />
        </div>
      </div>
    </article>
  );
}
