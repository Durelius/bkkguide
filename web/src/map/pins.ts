import type { Category, Place } from "../api";

/**
 * DOM for a MapLibre HTML marker, matching components/MapPin.tsx.
 * Built by hand because markers live outside the React tree.
 */
export function pinElement(place: Place, category: Category, onSelect: () => void): HTMLDivElement {
  // MapLibre owns the wrapper's transform; the button inside can animate freely.
  const wrapper = document.createElement("div");
  wrapper.className = "map-pin-anchor";
  const el = document.createElement("button");
  el.type = "button";
  el.className = "map-pin map-pin--button";
  el.style.setProperty("--cat", category.color);
  el.setAttribute("aria-label", `${place.name}, ${category.name}`);

  const glyph = document.createElement("span");
  glyph.className = "nf map-pin__glyph";
  glyph.setAttribute("aria-hidden", "true");
  glyph.textContent = category.icon;

  const label = document.createElement("span");
  label.className = "map-pin__label";
  label.textContent = place.name;

  el.append(glyph, label);
  el.addEventListener("click", (e) => {
    e.stopPropagation();
    onSelect();
  });
  wrapper.append(el);
  return wrapper;
}

export function setPinSelected(wrapper: HTMLElement, selected: boolean) {
  wrapper.classList.toggle("map-pin-anchor--selected", selected);
  const pin = wrapper.firstElementChild as HTMLElement;
  pin.classList.toggle("map-pin--selected", selected);
  pin.setAttribute("aria-pressed", String(selected));
}

export function setPinHovered(wrapper: HTMLElement, hovered: boolean) {
  wrapper.classList.toggle("map-pin-anchor--hovered", hovered);
  (wrapper.firstElementChild as HTMLElement).classList.toggle("map-pin--hovered", hovered);
}
