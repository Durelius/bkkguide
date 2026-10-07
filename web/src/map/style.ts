import type { StyleSpecification } from "maplibre-gl";

// MapLibre 6 silently ignores a pre-fetched style object, so pass the URL and
// tint it in transformStyle: map.setStyle(BASE_STYLE, { transformStyle: (_, next) => tintStyle(next) }).
export const BASE_STYLE = "https://tiles.openfreemap.org/styles/positron";

// Layer id -> paint overrides that tint OpenFreeMap's positron to the site palette.
const FILL: Record<string, string> = {
  background: "#e4d6db",
  park: "#d6d4c8",
  landcover_wood: "#d2d0c3",
  water: "#c3d3d8",
  landuse_residential: "#dfd0d6",
  building: "#d8c7ce",
  road_area_pier: "#e4d6db",
  "aeroway-area": "#efe6ea",
};
const LINE: Record<string, string> = {
  waterway: "#c3d3d8",
  road_pier: "#e4d6db",
  highway_path: "#ece2e6",
  highway_minor: "#efe6ea",
  highway_major_casing: "#cdb9c1",
  highway_major_inner: "#f5eef1",
  highway_major_subtle: "#e9dde2",
  highway_motorway_casing: "#cdb9c1",
  highway_motorway_inner: "#f5eef1",
  highway_motorway_subtle: "#e9dde2",
  highway_motorway_bridge_casing: "#cdb9c1",
  highway_motorway_bridge_inner: "#f5eef1",
  tunnel_motorway_casing: "#d4c2c9",
  tunnel_motorway_inner: "#ebe0e5",
  // Skytrain and metro lines get a muted pink so transit reads at a glance.
  railway_transit: "#c98aa6",
  railway_transit_dashline: "#f5eef1",
  railway_service: "#d4bcc6",
  railway: "#d4bcc6",
  railway_dashline: "#f5eef1",
};

/** Recolour the base style to the site palette. */
export function tintStyle(style: StyleSpecification): StyleSpecification {
  for (const layer of style.layers) {
    // We're always inside Bangkok; its big city label only clutters the pins.
    if (layer.id.startsWith("label_city") || layer.id.startsWith("label_state")) layer.maxzoom = 10;
    const paint = ((layer as { paint?: Record<string, unknown> }).paint ??= {});
    if (layer.type === "background" && FILL[layer.id]) paint["background-color"] = FILL[layer.id];
    if (layer.type === "fill" && FILL[layer.id]) {
      paint["fill-color"] = FILL[layer.id];
      if (layer.id === "building") paint["fill-outline-color"] = "#cdb9c1";
    }
    if (layer.type === "line" && LINE[layer.id]) paint["line-color"] = LINE[layer.id];
    if (layer.type === "symbol" && "text-color" in paint) {
      paint["text-color"] = layer.id.startsWith("water") ? "#5b7280" : "#4a3540";
      paint["text-halo-color"] = "rgba(228, 214, 219, 0.85)";
    }
  }
  return style;
}
