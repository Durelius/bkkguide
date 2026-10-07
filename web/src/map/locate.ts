import type { IControl, Map as MaplibreMap } from "maplibre-gl";
import { Marker } from "maplibre-gl";
import { glyph } from "../icons";

type Status = "idle" | "locating" | "found";

/**
 * "My location" button: asks for the device position once per press,
 * flies there and drops a "you are here" dot.
 */
export class LocateControl implements IControl {
  private map?: MaplibreMap;
  private container = document.createElement("div");
  private button = document.createElement("button");
  private dot?: Marker;

  constructor(private onError: (message: string) => void) {}

  onAdd(map: MaplibreMap) {
    this.map = map;
    this.container.className = "maplibregl-ctrl maplibregl-ctrl-group locate-ctrl";
    this.button.type = "button";
    this.button.className = "locate-ctrl__btn nf";
    this.button.textContent = glyph("location-arrow");
    this.button.setAttribute("aria-label", "Show my location");
    this.button.title = "Show my location";
    this.button.addEventListener("click", () => this.locate());
    this.container.append(this.button);
    return this.container;
  }

  onRemove() {
    this.dot?.remove();
    this.container.remove();
    this.map = undefined;
  }

  private setStatus(status: Status) {
    this.button.dataset.status = status;
    this.button.setAttribute("aria-busy", String(status === "locating"));
  }

  private locate() {
    if (!("geolocation" in navigator)) {
      this.onError("This browser can't share your location.");
      return;
    }
    this.setStatus("locating");
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        const map = this.map;
        if (!map) return;
        const at: [number, number] = [coords.longitude, coords.latitude];
        if (!this.dot) {
          const el = document.createElement("div");
          el.className = "you-are-here";
          el.setAttribute("aria-label", "You are here");
          this.dot = new Marker({ element: el }).setLngLat(at).addTo(map);
        } else {
          this.dot.setLngLat(at);
        }
        map.flyTo({ center: at, zoom: Math.max(map.getZoom(), 15), duration: 900 });
        this.setStatus("found");
      },
      (err) => {
        this.setStatus("idle");
        this.onError(
          err.code === err.PERMISSION_DENIED
            ? "Location is blocked for this site. Allow it in your browser settings, then try again."
            : "Couldn't find your location. Try again outside or with better signal.",
        );
      },
      { enableHighAccuracy: true, timeout: 10_000, maximumAge: 30_000 },
    );
  }
}
