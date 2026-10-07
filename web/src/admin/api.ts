// Admin API client. Session lives in an HttpOnly cookie; mutating requests
// carry the X-BKK-Admin header the server requires as CSRF protection.

export type Admin = { studentId: string; fullName: string; active: boolean; createdAt: string };

export type AuditEntry = {
  id: number;
  at: string;
  adminId: string;
  adminName: string | null;
  action: string;
  entity: string;
  entityId: string;
  details: Record<string, unknown>;
};

export type Interval = { weekday: number; opens: string; closes: string };

export type PlaceInput = {
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
  status: "draft" | "published";
  featured: boolean;
  hours: Interval[];
};

export type Photo = { id: number; width: number; height: number; alt: string; sortOrder: number; thumb: string; large: string };

export type AdminPlace = PlaceInput & {
  id: number;
  photos: Photo[];
  createdAt: string;
  updatedAt: string;
  createdBy: string | null;
  updatedBy: string | null;
};

export type PlaceRow = {
  id: number;
  slug: string;
  name: string;
  nameTh: string;
  category: string;
  status: "draft" | "published";
  featured: boolean;
  updatedAt: string;
  updatedBy: string | null;
  updatedByName: string | null;
  photoCount: number;
  thumb: string | null;
};

export type AdminCategory = { id: number; slug: string; name: string; icon: string; color: string; sortOrder: number; placeCount: number };
export type CategoryInput = { slug: string; name: string; icon: string; color: string };

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api/admin${path}`, {
    method,
    credentials: "same-origin",
    headers: {
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      ...(method !== "GET" ? { "X-BKK-Admin": "1" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, data.error ?? `Request failed (${res.status}).`);
  return data as T;
}

export const adminApi = {
  login: (studentId: string, password: string) => request<Admin>("POST", "/login", { studentId, password }),
  logout: () => request<void>("POST", "/logout"),
  me: () => request<Admin>("GET", "/me"),
  changePassword: (current: string, next: string) => request<void>("PUT", "/me/password", { current, new: next }),
  admins: () => request<Admin[]>("GET", "/admins"),
  createAdmin: (a: { studentId: string; fullName: string; password: string }) => request<Admin>("POST", "/admins", a),
  updateAdmin: (id: string, patch: { fullName?: string; active?: boolean; password?: string }) =>
    request<void>("PATCH", `/admins/${encodeURIComponent(id)}`, patch),
  deleteAdmin: (id: string) => request<void>("DELETE", `/admins/${encodeURIComponent(id)}`),
  places: (filter: { q?: string; category?: string; status?: string }) => {
    const q = new URLSearchParams(Object.entries(filter).filter(([, v]) => v) as [string, string][]);
    return request<PlaceRow[]>("GET", `/places?${q}`);
  },
  place: (id: number) => request<AdminPlace>("GET", `/places/${id}`),
  createPlace: (p: PlaceInput) => request<AdminPlace>("POST", "/places", p),
  updatePlace: (id: number, p: PlaceInput) => request<AdminPlace>("PUT", `/places/${id}`, p),
  patchPlace: (id: number, patch: { status?: "draft" | "published"; featured?: boolean }) => request<void>("PATCH", `/places/${id}`, patch),
  deletePlace: (id: number) => request<void>("DELETE", `/places/${id}`),
  uploadPhoto: async (placeId: number, file: File, alt: string) => {
    const form = new FormData();
    form.append("file", file);
    form.append("alt", alt);
    const res = await fetch(`/api/admin/places/${placeId}/photos`, {
      method: "POST",
      credentials: "same-origin",
      headers: { "X-BKK-Admin": "1" },
      body: form,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new ApiError(res.status, data.error ?? `Upload failed (${res.status}).`);
    return data as Photo[];
  },
  updatePhoto: (placeId: number, photoId: number, alt: string) => request<void>("PATCH", `/places/${placeId}/photos/${photoId}`, { alt }),
  reorderPhotos: (placeId: number, ids: number[]) => request<void>("PUT", `/places/${placeId}/photos/order`, { ids }),
  deletePhoto: (placeId: number, photoId: number) => request<void>("DELETE", `/places/${placeId}/photos/${photoId}`),
  categories: () => request<AdminCategory[]>("GET", "/categories"),
  createCategory: (c: CategoryInput) => request<AdminCategory>("POST", "/categories", c),
  updateCategory: (id: number, c: CategoryInput) => request<void>("PUT", `/categories/${id}`, c),
  deleteCategory: (id: number) => request<void>("DELETE", `/categories/${id}`),
  reorderCategories: (ids: number[]) => request<void>("PUT", "/categories/order", { ids }),
  audit: (params: { before?: number; admin?: string; entity?: string; entityId?: string; limit?: number }) => {
    const q = new URLSearchParams();
    if (params.before) q.set("before", String(params.before));
    if (params.admin) q.set("admin", params.admin);
    if (params.entity) q.set("entity", params.entity);
    if (params.entityId) q.set("entityId", params.entityId);
    q.set("limit", String(params.limit ?? 50));
    return request<AuditEntry[]>("GET", `/audit?${q}`);
  },
};
