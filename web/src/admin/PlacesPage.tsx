import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Glyph, Icon } from "../components/Icon";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi, type AdminCategory, type PlaceRow } from "./api";
import { formatDate } from "./format";

export function PlacesPage() {
  useDocumentTitle("Places");
  const [q, setQ] = useState("");
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState("");
  const categories = useQuery({ queryKey: ["admin", "categories"], queryFn: adminApi.categories });
  const places = useQuery({
    queryKey: ["admin", "places", { q, category, status }],
    queryFn: () => adminApi.places({ q, category, status }),
    placeholderData: (prev) => prev,
  });
  const catBySlug = new Map((categories.data ?? []).map((c) => [c.slug, c]));
  const filtered = q || category || status;

  return (
    <div className="admin-page admin-page--wide">
      <header className="admin-page__head">
        <div>
          <h1>Places</h1>
          <p className="admin-page__sub">Drafts are only visible here. Publish a place to put it on the site.</p>
        </div>
        <Link to="/admin/places/new" className="btn btn--primary">
          <Icon name="plus" /> New place
        </Link>
      </header>

      <div className="filters" role="search">
        <label className="field field--inline filters__search">
          <span className="visually-hidden">Search places</span>
          <input type="search" placeholder="Search by name" value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <label className="field field--inline">
          <span className="visually-hidden">Category</span>
          <select value={category} onChange={(e) => setCategory(e.target.value)}>
            <option value="">All categories</option>
            {categories.data?.map((c) => (
              <option key={c.slug} value={c.slug}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <label className="field field--inline">
          <span className="visually-hidden">Status</span>
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">Drafts and published</option>
            <option value="published">Published</option>
            <option value="draft">Drafts</option>
          </select>
        </label>
      </div>

      {places.isLoading && <p className="admin-muted">Loading places…</p>}
      {places.isError && <p className="form-error">{places.error.message}</p>}
      {places.data?.length === 0 &&
        (filtered ? (
          <p className="admin-muted">No places match these filters.</p>
        ) : (
          <div className="admin-card admin-empty">
            <p>No places yet. Add the first one.</p>
            <Link to="/admin/places/new" className="btn btn--primary">
              <Icon name="plus" /> New place
            </Link>
          </div>
        ))}

      <ul className="place-rows">
        {places.data?.map((p) => (
          <PlaceRowItem key={p.id} place={p} category={catBySlug.get(p.category)} />
        ))}
      </ul>
    </div>
  );
}

function PlaceRowItem({ place, category }: { place: PlaceRow; category?: AdminCategory }) {
  const qc = useQueryClient();
  const patch = useMutation({
    mutationFn: (p: { status?: "draft" | "published"; featured?: boolean }) => adminApi.patchPlace(place.id, p),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["admin", "places"] });
      qc.invalidateQueries({ queryKey: ["places"] });
    },
  });
  const published = place.status === "published";

  return (
    <li className="place-row" style={{ "--cat": category?.color ?? "var(--pink-button)" } as React.CSSProperties}>
      <span className="place-row__thumb">
        {place.thumb ? <img src={place.thumb} alt="" loading="lazy" /> : category && <Glyph char={category.icon} />}
      </span>
      <div className="place-row__main">
        <Link to={`/admin/places/${place.id}`} className="place-row__name">
          {place.name}
        </Link>
        <span className="place-row__meta">
          {category?.name ?? place.category}
          {place.nameTh && <span className="th">{place.nameTh}</span>}
          <span>
            updated {formatDate(place.updatedAt)}
            {place.updatedByName ? ` by ${place.updatedByName}` : ""}
          </span>
          {place.photoCount === 0 && <span className="place-row__warn">no photos</span>}
        </span>
        {patch.error && <span className="form-error">{patch.error.message}</span>}
      </div>
      <div className="place-row__actions">
        <button
          type="button"
          className={`star-toggle${place.featured ? " star-toggle--on" : ""}`}
          aria-pressed={place.featured}
          aria-label={place.featured ? `Remove ${place.name} from editors' picks` : `Make ${place.name} an editors' pick`}
          title="Editors' pick"
          disabled={patch.isPending}
          onClick={() => patch.mutate({ featured: !place.featured })}
        >
          <Icon name="star" />
        </button>
        <span className={`status ${published ? "status--on" : "status--off"}`}>{published ? "Published" : "Draft"}</span>
        <button type="button" className="btn-small" disabled={patch.isPending} onClick={() => patch.mutate({ status: published ? "draft" : "published" })}>
          {published ? "Unpublish" : "Publish"}
        </button>
      </div>
    </li>
  );
}
