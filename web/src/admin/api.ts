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
  audit: (params: { before?: number; admin?: string; limit?: number }) => {
    const q = new URLSearchParams();
    if (params.before) q.set("before", String(params.before));
    if (params.admin) q.set("admin", params.admin);
    q.set("limit", String(params.limit ?? 50));
    return request<AuditEntry[]>("GET", `/audit?${q}`);
  },
};
