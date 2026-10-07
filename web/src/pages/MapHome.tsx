import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../api";
import { PlacePreview } from "../components/PlacePreview";
import { DESKTOP, PlacesMap } from "../map/PlacesMap";
import { useDocumentTitle } from "../useDocumentTitle";
import "./MapHome.css";

export function MapHome() {
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });
  const places = useQuery({ queryKey: ["places"], queryFn: () => api.places() });
  useDocumentTitle();
  // Links from category pages: /?category=slug shows one category, /?place=slug opens on a place.
  const [params, setParams] = useSearchParams();
  const [focus] = useState(() => params.get("place"));
  const [onlyCategory] = useState(() => params.get("category"));
  const [selected, setSelected] = useState<string | null>(focus);
  const [hidden, setHidden] = useState<Set<string> | null>(null);

  const cats = categories.data ?? [];

  // Resolve the initial filter once categories are known.
  useEffect(() => {
    if (hidden || !categories.data) return;
    setHidden(new Set(onlyCategory ? categories.data.filter((c) => c.slug !== onlyCategory).map((c) => c.slug) : []));
  }, [categories.data, hidden, onlyCategory]);

  // The URL only seeds the view; drop it so reloads and shares start from the full map.
  useEffect(() => {
    if (params.size > 0) setParams({}, { replace: true });
  }, [params, setParams]);
  const catBySlug = useMemo(() => new Map(cats.map((c) => [c.slug, c])), [cats]);
  const visible = useMemo(() => (hidden ? (places.data ?? []).filter((p) => !hidden.has(p.category)) : []), [places.data, hidden]);
  const counts = useMemo(() => {
    const m = new Map<string, number>();
    for (const p of places.data ?? []) m.set(p.category, (m.get(p.category) ?? 0) + 1);
    return m;
  }, [places.data]);
  const selectedPlace = visible.find((p) => p.slug === selected);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setSelected(null);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const toggleCategory = (slug: string) =>
    setHidden((h) => {
      const next = new Set(h ?? []);
      next.has(slug) ? next.delete(slug) : next.add(slug);
      return next;
    });

  return (
    <div className="map-home">
      <h1 className="visually-hidden">Map of all places</h1>
      <PlacesMap
        places={visible}
        categories={cats}
        selected={selectedPlace ? selected : null}
        onSelect={setSelected}
        padding={window.matchMedia(DESKTOP).matches ? { top: 90, bottom: 60, left: 60, right: 60 } : { top: 80, bottom: 40, left: 40, right: 40 }}
        selectOffset={() => (window.matchMedia(DESKTOP).matches ? [180, 0] : [0, -110])}
        zoomControls={window.matchMedia(DESKTOP).matches}
        locate
        initialFocus={focus}
      />

      {cats.length > 0 && (
        <div className="map-home__filters" role="group" aria-label="Show categories">
          {cats.map((c) => (
            <button
              key={c.slug}
              type="button"
              className="map-filter"
              style={{ "--cat": c.color } as React.CSSProperties}
              aria-pressed={!hidden?.has(c.slug)}
              onClick={() => toggleCategory(c.slug)}
            >
              <span className="nf" aria-hidden>
                {c.icon}
              </span>
              {c.name}
              <span className="map-filter__count">{counts.get(c.slug) ?? 0}</span>
            </button>
          ))}
        </div>
      )}

      {places.isError && (
        <p className="map-home__error" role="alert">
          Places couldn't load. Reload the page to try again.
        </p>
      )}

      {selectedPlace && (
        <div className="map-home__preview">
          <PlacePreview place={selectedPlace} category={catBySlug.get(selectedPlace.category)} onClose={() => setSelected(null)} />
        </div>
      )}
    </div>
  );
}
