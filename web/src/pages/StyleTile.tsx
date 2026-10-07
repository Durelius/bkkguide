import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { api, type Category, type Place } from "../api";
import { Logo, SunriseMark } from "../components/Logo";
import { MapPin } from "../components/MapPin";
import { OpenBadge, PriceLevel, StationTag } from "../components/Meta";
import { DesktopNav, ListMapToggle, SearchField, TabBar, TopBar } from "../components/Nav";
import { PlaceCard } from "../components/PlaceCard";
import { Icon } from "../components/Icon";
import "./StyleTile.css";

const SWATCHES = [
  { name: "Rose dust", hex: "#E4D6DB", role: "Page background", ink: true },
  { name: "Petal", hex: "#F5EEF1", role: "Cards and bars", ink: true },
  { name: "Plum ink", hex: "#24161D", role: "Text, 12.4:1 on rose dust" },
  { name: "Hot pink", hex: "#D9447F", role: "Rays, focus, fills only" },
  { name: "Deep pink", hex: "#A3245A", role: "Links and pink text, 5.1:1" },
  { name: "Button pink", hex: "#BE3570", role: "Primary buttons, Shopping" },
  { name: "Chilli", hex: "#C4532F", role: "Restaurants" },
  { name: "Orchid", hex: "#8A3F86", role: "Culture" },
];

const TYPE = [
  { token: "2xl / 800", size: "var(--fs-2xl)", weight: 800, text: "Bangkok, hand-picked" },
  { token: "xl / 800", size: "var(--fs-xl)", weight: 800, text: "Restaurants near Sam Yot" },
  { token: "lg / 800", size: "var(--fs-lg)", weight: 800, text: "Thipsamai Pad Thai" },
  { token: "md / 400", size: "var(--fs-md)", weight: 400, text: "Old-school pad thai wrapped in a thin egg crepe. Fresh orange juice is a must." },
  { token: "sm / 400", size: "var(--fs-sm)", weight: 400, text: "Shoulders and knees covered. Shoes off inside halls." },
];

const PIN_SPOTS = [
  { slug: "wat-pho", x: 18, y: 58 },
  { slug: "grand-palace", x: 22, y: 30 },
  { slug: "jay-fai", x: 40, y: 22 },
  { slug: "jim-thompson-house", x: 58, y: 44 },
  { slug: "siam-paragon", x: 72, y: 52 },
  { slug: "somtam-nua", x: 67, y: 70 },
  { slug: "iconsiam", x: 32, y: 80 },
  { slug: "wattana-panich", x: 88, y: 76 },
];

export function StyleTile() {
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });
  const places = useQuery({ queryKey: ["places"], queryFn: () => api.places() });
  const [saved, setSaved] = useState<Set<string>>(new Set(["thipsamai"]));

  const cats = categories.data ?? [];
  const bySlug = new Map(cats.map((c) => [c.slug, c]));
  const all = places.data ?? [];
  const find = (slug: string) => all.find((p) => p.slug === slug);
  const cards = ["jay-fai", "chatuchak", "grand-palace"].map(find).filter((p): p is Place => !!p);
  const toggle = (slug: string) =>
    setSaved((s) => {
      const next = new Set(s);
      next.has(slug) ? next.delete(slug) : next.add(slug);
      return next;
    });

  if (categories.isLoading || places.isLoading) return <p className="tile-loading">Loading…</p>;
  if (categories.isError || places.isError) return <p className="tile-loading">The API isn't running. Start it with `make dev`.</p>;

  return (
    <div className="tile">
      <div className="tile__desknav">
        <DesktopNav categories={cats} active="restaurants" />
      </div>

      <main className="tile__main">
        <header className="tile__intro">
          <h1>Style tile</h1>
          <p>
            Pink and playful, typed like a terminal. One family everywhere: JetBrains Mono Nerd Font, with IBM Plex Sans Thai
            for Thai script. The rising-sun rays are the one decorative motif.
          </p>
        </header>

        <section className="tile__section">
          <h2>Mark</h2>
          <div className="tile__marks">
            <div className="tile__mark tile__mark--bg">
              <Logo />
            </div>
            <div className="tile__mark tile__mark--card">
              <SunriseMark size={120} />
            </div>
            <div className="tile__mark tile__mark--ink">
              <Logo />
            </div>
          </div>
        </section>

        <section className="tile__section">
          <h2>Colour</h2>
          <ul className="tile__swatches">
            {SWATCHES.map((s) => (
              <li key={s.hex}>
                <span className="tile__chip" style={{ background: s.hex }} data-outline={s.ink ? "" : undefined} />
                <strong>{s.name}</strong>
                <code>{s.hex}</code>
                <span className="tile__role">{s.role}</span>
              </li>
            ))}
          </ul>
        </section>

        <section className="tile__section">
          <h2>Type</h2>
          <div className="tile__type">
            {TYPE.map((t) => (
              <div key={t.token} className="tile__type-row">
                <code>{t.token}</code>
                <span style={{ fontSize: t.size, fontWeight: t.weight }}>{t.text}</span>
              </div>
            ))}
            <div className="tile__type-row">
              <code>thai</code>
              <span className="th" lang="th" style={{ fontSize: "var(--fs-lg)" }}>
                วัดโพธิ์ · ตลาดนัดจตุจักร · เจ๊ไฝ
              </span>
            </div>
          </div>
        </section>

        <section className="tile__section">
          <h2>Controls</h2>
          <div className="tile__row">
            <button className="btn btn--primary">
              <Icon name="map-marker" /> Open in Google Maps
            </button>
            <button className="btn btn--ghost">
              <Icon name="heart-outline" /> Save
            </button>
            <ListMapToggle value="list" />
            <SearchField />
          </div>
          <div className="tile__row">
            <OpenBadge open closesAt="19:00" />
            <OpenBadge open closesAt="00:00" />
            <OpenBadge open={false} />
            <StationTag station="Sam Yot" walk={8} />
            <PriceLevel level={1} />
            <PriceLevel level={3} />
          </div>
        </section>

        <section className="tile__section">
          <h2>Place cards</h2>
          <div className="tile__cards">
            {cards.map((p) => (
              <PlaceCard
                key={p.slug}
                place={p}
                category={bySlug.get(p.category)}
                saved={saved.has(p.slug)}
                onToggleSave={() => toggle(p.slug)}
              />
            ))}
          </div>
        </section>

        <section className="tile__section">
          <h2>Map pins</h2>
          <p className="tile__note">
            Pins take the category colour and icon. The selected pin lifts and shows its name. The real map uses a
            MapLibre style tinted to match rose dust.
          </p>
          <div className="tile__map">
            {PIN_SPOTS.map(({ slug, x, y }) => {
              const p = find(slug);
              const c = p && bySlug.get(p.category);
              if (!p || !c) return null;
              return (
                <span key={slug} className="tile__map-pin" style={{ left: `${x}%`, top: `${y}%` }}>
                  <MapPin category={c} label={p.name} selected={slug === "jim-thompson-house"} />
                </span>
              );
            })}
            <div className="tile__map-legend">
              {cats.map((c) => (
                <span key={c.slug} className="tile__legend-chip" style={{ "--cat": c.color } as React.CSSProperties}>
                  <span className="nf">{c.icon}</span>
                  {c.name}
                </span>
              ))}
            </div>
          </div>
        </section>

        <section className="tile__section">
          <h2>Mobile shell</h2>
          <div className="tile__phones">
            <Phone categories={cats} active="restaurants">
              <div className="tile__phone-head">
                <h3>Restaurants</h3>
                <ListMapToggle value="list" />
              </div>
              {all
                .filter((p) => p.category === "restaurants")
                .slice(0, 2)
                .map((p) => (
                  <PlaceCard key={p.slug} place={p} category={bySlug.get("restaurants")} saved={saved.has(p.slug)} onToggleSave={() => toggle(p.slug)} />
                ))}
            </Phone>
            <Phone categories={cats} active="map">
              <div className="tile__map tile__map--phone">
                {PIN_SPOTS.slice(0, 6).map(({ slug, x, y }) => {
                  const p = find(slug);
                  const c = p && bySlug.get(p.category);
                  if (!p || !c) return null;
                  return (
                    <span key={slug} className="tile__map-pin" style={{ left: `${x}%`, top: `${y}%` }}>
                      <MapPin category={c} label={p.name} selected={slug === "wat-pho"} />
                    </span>
                  );
                })}
              </div>
            </Phone>
          </div>
        </section>
      </main>
    </div>
  );
}

function Phone({ categories, active, children }: { categories: Category[]; active: string; children: React.ReactNode }) {
  return (
    <div className="tile__phone">
      <TopBar />
      <div className="tile__phone-body">{children}</div>
      <TabBar categories={categories} active={active} />
    </div>
  );
}
