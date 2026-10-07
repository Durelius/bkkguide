import * as maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";
import { useEffect, useRef } from "react";
import { BASE_STYLE, tintStyle } from "../map/style";

maplibregl.setWorkerUrl(workerUrl);

const BANGKOK: [number, number] = [100.53, 13.75];

type Props = {
  lat: number | null;
  lng: number | null;
  color: string;
  onChange: (lat: number, lng: number) => void;
};

/** Click the map or drag the pin to set a place's location. */
export default function MapPicker({ lat, lng, color, onChange }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const marker = useRef<maplibregl.Marker | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    if (!container.current) return;
    const has = lat != null && lng != null;
    const m = new maplibregl.Map({
      container: container.current,
      center: has ? [lng!, lat!] : BANGKOK,
      zoom: has ? 16 : 11,
      attributionControl: { compact: true },
      dragRotate: false,
      pitchWithRotate: false,
    });
    m.setStyle(BASE_STYLE, { transformStyle: (_, next) => tintStyle(next) });
    m.addControl(new maplibregl.NavigationControl({ showCompass: false }), "bottom-right");
    m.on("click", (e) => onChangeRef.current(round(e.lngLat.lat), round(e.lngLat.lng)));
    map.current = m;
    const ro = new ResizeObserver(() => m.resize());
    ro.observe(container.current);
    return () => {
      ro.disconnect();
      m.remove();
      map.current = null;
      marker.current = null;
    };
    // The map is created once; position changes are applied below.
  }, []);

  useEffect(() => {
    const m = map.current;
    if (!m || lat == null || lng == null) return;
    if (!marker.current) {
      marker.current = new maplibregl.Marker({ color, draggable: true }).setLngLat([lng, lat]).addTo(m);
      marker.current.on("dragend", () => {
        const p = marker.current!.getLngLat();
        onChangeRef.current(round(p.lat), round(p.lng));
      });
    } else {
      marker.current.setLngLat([lng, lat]);
    }
    if (!m.getBounds().contains([lng, lat])) m.easeTo({ center: [lng, lat], zoom: Math.max(m.getZoom(), 15) });
  }, [lat, lng, color]);

  return <div ref={container} className="map-picker" />;
}

function round(n: number) {
  return Math.round(n * 1e6) / 1e6;
}
