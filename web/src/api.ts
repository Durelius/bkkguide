export type Category = {
  id: number;
  slug: string;
  name: string;
  icon: string;
  color: string;
  sortOrder: number;
};

export type Interval = { weekday: number; opens: string; closes: string };

export type Photo = { thumb: string; large: string; width: number; height: number; alt: string };

export type Place = {
  id: number;
  slug: string;
  category: string;
  name: string;
  nameTh: string;
  summary: string;
  descriptionMd: string;
  address: string;
  lat: number;
  lng: number;
  nearestStation: string;
  stationLine: string;
  walkMinutes: number | null;
  priceLevel: number | null;
  dressCode: string;
  etiquetteTips: string;
  mustTry: string;
  website: string;
  phone: string;
  googleMapsUrl: string;
  featured: boolean;
  hours: Interval[];
  photos: Photo[];
  openNow: boolean;
  closesAt?: string;
};

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`/api${path}`);
  if (!res.ok) throw new Error(`${res.status} ${path}`);
  return res.json();
}

export const api = {
  categories: () => get<Category[]>("/categories"),
  places: (category?: string) =>
    get<Place[]>(category ? `/places?category=${encodeURIComponent(category)}` : "/places"),
  place: (slug: string) => get<Place>(`/places/${encodeURIComponent(slug)}`),
};
