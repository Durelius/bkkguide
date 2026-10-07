import { Link, useLocation } from "react-router-dom";
import type { Category } from "../api";
import { Glyph, Icon } from "./Icon";
import { Logo } from "./Logo";
import "./Nav.css";

type NavProps = { categories: Category[]; active?: string };

function navItems(categories: Category[]) {
  return [
    ...categories.map((c) => ({ key: c.slug, label: c.name, href: `/c/${c.slug}`, glyph: <Glyph char={c.icon} /> })),
    { key: "map", label: "Map", href: "/", glyph: <Icon name="map-outline" /> },
  ];
}

/** The nav key for the current URL: "map" on the home page, the category slug on /c/:slug. */
function useActiveKey(override?: string) {
  const { pathname } = useLocation();
  if (override) return override;
  if (pathname === "/") return "map";
  const m = pathname.match(/^\/c\/([^/]+)/);
  return m ? m[1] : "";
}

/** Mobile: thumb-reach tab bar fixed to the bottom. */
export function TabBar({ categories, active }: NavProps) {
  const current = useActiveKey(active);
  return (
    <nav className="tabbar" aria-label="Main">
      {navItems(categories).map((item) => (
        <Link key={item.key} to={item.href} className="tabbar__item" aria-current={item.key === current ? "page" : undefined}>
          <span className="tabbar__glyph">{item.glyph}</span>
          <span className="tabbar__label">{item.label}</span>
        </Link>
      ))}
    </nav>
  );
}

/** Mobile top bar: logo plus search and saved. */
export function TopBar() {
  return (
    <header className="topbar">
      <Link to="/" className="topbar__home" aria-label="Home">
        <Logo compact />
      </Link>
      <div className="topbar__actions">
        <Link to="/search" className="icon-btn" aria-label="Search">
          <Icon name="search" />
        </Link>
        <Link to="/favourites" className="icon-btn" aria-label="Saved places">
          <Icon name="heart-outline" />
        </Link>
      </div>
    </header>
  );
}

/** Desktop: one bar with logo, the same nav items, and search. */
export function DesktopNav({ categories, active }: NavProps) {
  const current = useActiveKey(active);
  return (
    <header className="desknav">
      <Link to="/" className="topbar__home" aria-label="Home">
        <Logo />
      </Link>
      <nav className="desknav__items" aria-label="Main">
        {navItems(categories).map((item) => (
          <Link key={item.key} to={item.href} className="desknav__item" aria-current={item.key === current ? "page" : undefined}>
            {item.glyph}
            {item.label}
          </Link>
        ))}
      </nav>
      <div className="topbar__actions">
        <SearchField />
        <Link to="/favourites" className="icon-btn" aria-label="Saved places">
          <Icon name="heart-outline" />
        </Link>
      </div>
    </header>
  );
}

export function SearchField() {
  return (
    <label className="search">
      <Icon name="search" />
      <span className="visually-hidden">Search places</span>
      <span className="search__box">
        <input type="search" placeholder=" " />
        <span className="search__hint" aria-hidden>
          search places<span className="search__cursor" />
        </span>
      </span>
    </label>
  );
}

export function ListMapToggle({ value }: { value: "list" | "map" }) {
  return (
    <div className="segmented" role="group" aria-label="View">
      <button type="button" aria-pressed={value === "list"}>
        <Icon name="list" /> List
      </button>
      <button type="button" aria-pressed={value === "map"}>
        <Icon name="map-outline" /> Map
      </button>
    </div>
  );
}
