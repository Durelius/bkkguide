import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { Glyph, Icon } from "../components/Icon";
import { PlaceCard } from "../components/PlaceCard";
import { useFavourites } from "../favourites";
import { useDocumentTitle } from "../useDocumentTitle";
import "./CategoryPage.css";

/** A category is a browsable list; the Map tab is the one place for maps. */
export function CategoryPage() {
  const { slug = "" } = useParams();
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });
  const places = useQuery({ queryKey: ["places", slug], queryFn: () => api.places(slug) });
  const { isSaved, toggle } = useFavourites();

  const category = categories.data?.find((c) => c.slug === slug);
  const list = places.data ?? [];
  useDocumentTitle(category?.name);

  if (categories.isSuccess && !category) {
    return (
      <div className="category-missing">
        <h1>No category called “{slug}”</h1>
        <Link to="/" className="btn btn--ghost">
          Back to the map
        </Link>
      </div>
    );
  }

  return (
    <div className="category" style={{ "--cat": category?.color } as React.CSSProperties}>
      <header className="category__head">
        <h1 className="category__title">
          {category && <Glyph char={category.icon} className="category__glyph" />}
          {category?.name ?? " "}
          {places.isSuccess && <span className="category__count">{list.length}</span>}
        </h1>
        <Link to={`/?category=${encodeURIComponent(slug)}`} className="btn btn--ghost category__maplink">
          <Icon name="map-outline" /> See on map
        </Link>
      </header>

      {places.isLoading && <p className="category__note">Loading places…</p>}
      {places.isError && <p className="category__note">Places couldn't load. Reload the page to try again.</p>}
      {places.isSuccess && list.length === 0 && <p className="category__note">Nothing here yet. Check back soon.</p>}

      <div className="category__list">
        {list.map((p) => (
          <PlaceCard key={p.slug} place={p} category={category} saved={isSaved(p.slug)} onToggleSave={() => toggle(p.slug)} />
        ))}
      </div>
    </div>
  );
}
