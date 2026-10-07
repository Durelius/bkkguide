import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api, type Interval, type Photo } from "../api";
import { Glyph, Icon } from "../components/Icon";
import { OpenBadge, PriceLevel } from "../components/Meta";
import { useFavourites } from "../favourites";
import { useDocumentTitle } from "../useDocumentTitle";
import "./PlacePage.css";

const PlacesMap = lazy(() => import("../map/PlacesMap").then((m) => ({ default: m.PlacesMap })));

const DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const WEEKDAY_OF = [1, 2, 3, 4, 5, 6, 0]; // DAYS index -> API weekday (0 = Sunday)

export function PlacePage() {
  const { slug = "" } = useParams();
  const place = useQuery({ queryKey: ["place", slug], queryFn: () => api.place(slug), retry: false });
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });
  const { isSaved, toggle } = useFavourites();
  const [thaiOpen, setThaiOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!thaiOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setThaiOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [thaiOpen]);

  const p = place.data;
  const category = categories.data?.find((c) => c.slug === p?.category);
  useDocumentTitle(p?.name);

  if (place.isLoading) return <p className="place-note">Loading…</p>;
  if (!p) {
    return (
      <div className="place-note">
        <h1>This place isn't in the guide</h1>
        <p>It may have been removed, or the link is mistyped.</p>
        <Link to="/" className="btn btn--ghost">
          Back to the map
        </Link>
      </div>
    );
  }

  const mapsUrl = p.googleMapsUrl || `https://www.google.com/maps/search/?api=1&query=${p.lat},${p.lng}`;
  const saved = isSaved(p.slug);
  const tips = [
    { label: "What to wear", text: p.dressCode },
    { label: "Good to know", text: p.etiquetteTips },
    { label: "Must try", text: p.mustTry },
  ].filter((t) => t.text);

  const share = async () => {
    const url = window.location.href;
    if (navigator.share) {
      try {
        await navigator.share({ title: p.name, url });
      } catch {
        // Dismissed by the user.
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard blocked; nothing else to do.
    }
  };

  return (
    <article className="place" style={{ "--cat": category?.color ?? "var(--pink-button)" } as React.CSSProperties}>
      <div className={`place__hero${p.photos.length ? " place__hero--photos" : ""}`}>
        {p.photos.length ? (
          <Gallery photos={p.photos} name={p.name} />
        ) : (
          category && <Glyph char={category.icon} className="place__hero-glyph" />
        )}
        {category && (
          <Link to={`/c/${category.slug}`} className="place__crumb">
            <Icon name="chevron-left" /> {category.name}
          </Link>
        )}
      </div>

      <div className="place__layout">
        <div className="place__main">
          <header className="place__head">
            {p.featured && (
              <span className="place__pick">
                <Icon name="star" /> editors' pick
              </span>
            )}
            <h1 className="place__name">{p.name}</h1>
            {p.nameTh && (
              <button type="button" className="place__th th" lang="th" onClick={() => setThaiOpen(true)}>
                {p.nameTh}
                <span className="place__th-hint" lang="en">
                  show to driver
                </span>
              </button>
            )}
            <div className="place__chips">
              <OpenBadge open={p.openNow} closesAt={p.closesAt} />
              <PriceLevel level={p.priceLevel} />
            </div>
          </header>

          <div className="place__actions">
            <a href={mapsUrl} target="_blank" rel="noopener noreferrer" className="btn btn--primary">
              <Icon name="map-marker" /> Directions
            </a>
            <button type="button" className="btn btn--ghost" aria-pressed={saved} onClick={() => toggle(p.slug)}>
              <Icon name={saved ? "heart" : "heart-outline"} /> {saved ? "Saved" : "Save"}
            </button>
            <button type="button" className="btn btn--ghost" onClick={share}>
              <Icon name="share" /> {copied ? "Link copied" : "Share"}
            </button>
          </div>

          <p className="place__summary">{p.summary}</p>
          {p.descriptionMd && (
            <div className="place__description">
              {p.descriptionMd.split(/\n{2,}/).map((para, i) => (
                <p key={i}>{para}</p>
              ))}
            </div>
          )}

          {tips.length > 0 && (
            <section className="place__section">
              <h2>Before you go</h2>
              <dl className="place__tips">
                {tips.map((t) => (
                  <div key={t.label}>
                    <dt>{t.label}</dt>
                    <dd>{t.text}</dd>
                  </div>
                ))}
              </dl>
            </section>
          )}
        </div>

        <aside className="place__side">
          {p.nearestStation && (
            <section className="place__box">
              <h2>Getting there</h2>
              <p className="place__station">
                <Icon name="train" />
                <span>
                  <strong>{p.nearestStation}</strong>
                  {p.stationLine && <span className="place__muted"> {p.stationLine}</span>}
                </span>
              </p>
              {p.walkMinutes != null && <p className="place__muted">{p.walkMinutes} min walk from the station</p>}
              {p.address && <p className="place__muted">{p.address}</p>}
            </section>
          )}

          <section className="place__box">
            <h2>Opening hours</h2>
            <HoursTable hours={p.hours} />
          </section>

          <div className="place__minimap">
            {categories.data && (
              <Suspense fallback={null}>
                <PlacesMap places={[p]} categories={categories.data} maxZoom={15} padding={{ top: 40, bottom: 20, left: 20, right: 20 }} />
              </Suspense>
            )}
          </div>
        </aside>
      </div>

      {thaiOpen && (
        <div className="thai-card" role="dialog" aria-modal="true" aria-label="Thai name" onClick={() => setThaiOpen(false)}>
          <p className="thai-card__name th" lang="th">
            {p.nameTh}
          </p>
          {p.nearestStation && <p className="thai-card__sub">near {p.nearestStation}</p>}
          <button type="button" className="btn btn--ghost thai-card__close" onClick={() => setThaiOpen(false)} autoFocus>
            Close
          </button>
        </div>
      )}
    </article>
  );
}

/** Swipeable strip of photos with a counter; one photo fills the hero. */
function Gallery({ photos, name }: { photos: Photo[]; name: string }) {
  const [index, setIndex] = useState(0);
  return (
    <>
      <div
        className="gallery"
        tabIndex={0}
        aria-label={`Photos of ${name}`}
        onScroll={(e) => {
          const el = e.currentTarget;
          setIndex(Math.round(el.scrollLeft / el.clientWidth));
        }}
      >
        {photos.map((ph, i) => (
          <img key={ph.large} src={ph.large} alt={ph.alt || `${name}, photo ${i + 1}`} loading={i === 0 ? "eager" : "lazy"} className="gallery__img" />
        ))}
      </div>
      {photos.length > 1 && (
        <span className="gallery__count" aria-hidden>
          {index + 1} / {photos.length}
        </span>
      )}
    </>
  );
}

function HoursTable({ hours }: { hours: Interval[] }) {
  if (hours.length === 0) return <p className="place__muted">Hours not listed yet.</p>;
  const today = new Date(new Date().toLocaleString("en-US", { timeZone: "Asia/Bangkok" })).getDay();
  return (
    <table className="hours">
      <tbody>
        {DAYS.map((day, i) => {
          const wd = WEEKDAY_OF[i];
          const slots = hours.filter((h) => h.weekday === wd);
          return (
            <tr key={day} className={wd === today ? "hours__today" : undefined}>
              <th scope="row">{day.slice(0, 3)}</th>
              <td>{slots.length ? slots.map((h) => `${h.opens}–${h.closes === "00:00" ? "24:00" : h.closes}`).join(", ") : "Closed"}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
