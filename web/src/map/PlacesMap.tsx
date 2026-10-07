import * as maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";
import { useEffect, useRef, useState } from "react";
import type { Category, Place } from "../api";
import "../components/MapPin.css";
import { LocateControl } from "./locate";
import { pinElement, setPinHovered, setPinSelected } from "./pins";
import { BASE_STYLE, tintStyle } from "./style";
import "./PlacesMap.css";

// MapLibre 6 ships its worker as a separate module; Vite bundles it with its imports.
maplibregl.setWorkerUrl(workerUrl);

const BANGKOK: [number, number] = [100.53, 13.75];
export const DESKTOP = "(min-width: 900px)";

type Padding = { top: number; bottom: number; left: number; right: number };

type Props = {
  places: Place[];
  categories: Category[];
  selected?: string | null;
  hovered?: string | null;
  onSelect?: (slug: string | null) => void;
  /** Padding used when framing all places and when centring the selected one. */
  padding?: Padding;
  /** Pixel offset applied when easing to the selected place (to clear overlays). */
  selectOffset?: () => [number, number];
  interactive?: boolean;
  zoomControls?: boolean;
  /** Show the "my location" button. */
  locate?: boolean;
  /** Open zoomed in on this place instead of framing all of them. */
  initialFocus?: string | null;
  maxZoom?: number;
};

/** A MapLibre map with one category pin per place. */
export function PlacesMap({
  places,
  categories,
  selected = null,
  hovered = null,
  onSelect,
  padding = { top: 60, bottom: 40, left: 40, right: 40 },
  selectOffset,
  interactive = true,
  zoomControls = false,
  locate = false,
  initialFocus = null,
  maxZoom = 15,
}: Props) {
  const container = useRef<HTMLDivElement>(null);
  const [map, setMap] = useState<maplibregl.Map | null>(null);
  const [failed, setFailed] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const pins = useRef(new Map<string, { place: Place; el: HTMLDivElement; marker: maplibregl.Marker }>());
  const framed = useRef(false);
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  useEffect(() => {
    if (!container.current) return;
    const instance = new maplibregl.Map({
      container: container.current,
      center: BANGKOK,
      zoom: 12,
      interactive,
      attributionControl: { compact: true },
      dragRotate: false,
      pitchWithRotate: false,
    });
    // MapLibre 6 silently ignores a pre-fetched style object, so tint the URL's style on load.
    instance.setStyle(BASE_STYLE, { transformStyle: (_, next) => tintStyle(next) });
    instance.touchZoomRotate.disableRotation();
    if (locate) instance.addControl(new LocateControl(setNotice), "bottom-right");
    if (zoomControls) instance.addControl(new maplibregl.NavigationControl({ showCompass: false }), "bottom-right");
    instance.on("click", () => onSelectRef.current?.(null));
    instance.on("error", (e) => {
      if (!instance.getStyle()) setFailed(true);
      console.error(e.error);
    });
    setMap(instance);
    return () => instance.remove();
  }, [interactive, zoomControls, locate]);

  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(null), 5000);
    return () => clearTimeout(t);
  }, [notice]);

  // Markers, re-created when the set of places changes; then frame them.
  useEffect(() => {
    if (!map) return;
    const bySlug = new Map(categories.map((c) => [c.slug, c]));
    const current = pins.current;
    const bounds = new maplibregl.LngLatBounds();
    for (const place of places) {
      const category = bySlug.get(place.category);
      if (!category) continue;
      const el = pinElement(place, category, () => onSelectRef.current?.(place.slug));
      const marker = new maplibregl.Marker({ element: el, anchor: "bottom" }).setLngLat([place.lng, place.lat]).addTo(map);
      current.set(place.slug, { place, el, marker });
      bounds.extend([place.lng, place.lat]);
    }
    // Frame only on first load; later changes (category toggles) keep the user's view.
    if (!framed.current && current.size > 0) {
      const focus = initialFocus ? current.get(initialFocus) : current.size === 1 ? [...current.values()][0] : undefined;
      if (focus) map.jumpTo({ center: [focus.place.lng, focus.place.lat], zoom: Math.min(15, maxZoom) });
      else map.fitBounds(bounds, { padding, maxZoom, duration: 0 });
      framed.current = true;
    }
    return () => {
      current.forEach((p) => p.marker.remove());
      current.clear();
    };
    // padding is a layout constant per caller; re-framing on every render would fight the user.
  }, [map, places, categories, maxZoom]);

  useEffect(() => {
    pins.current.forEach((p, slug) => setPinSelected(p.el, slug === selected));
    const pin = selected ? pins.current.get(selected) : undefined;
    if (!map || !pin) return;
    const bounds = map.getBounds();
    const offset = selectOffset?.() ?? [0, 0];
    if (!bounds.contains([pin.place.lng, pin.place.lat]) || offset[0] !== 0 || offset[1] !== 0) {
      map.easeTo({ center: [pin.place.lng, pin.place.lat], offset, duration: 450 });
    }
  }, [selected, map, places]);

  useEffect(() => {
    pins.current.forEach((p, slug) => setPinHovered(p.el, slug === hovered));
  }, [hovered, places]);

  // The container can change size (List/Map toggle, split view), so keep the canvas in sync.
  useEffect(() => {
    if (!map || !container.current) return;
    const ro = new ResizeObserver(() => map.resize());
    ro.observe(container.current);
    return () => ro.disconnect();
  }, [map]);

  return (
    <div className="places-map">
      <div ref={container} className="places-map__canvas" />
      {notice && (
        <p className="places-map__notice" role="status">
          {notice}
        </p>
      )}
      {failed && (
        <p className="places-map__error" role="alert">
          The map couldn't load. Check your connection and reload the page.
        </p>
      )}
    </div>
  );
}
