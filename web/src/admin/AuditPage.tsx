import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi, type AuditEntry } from "./api";
import { formatDateTime } from "./format";

const PAGE = 50;

export function AuditPage() {
  useDocumentTitle("Activity log");
  const [adminFilter, setAdminFilter] = useState("");
  // Linked from a place's "History": ?entity=place&id=12
  const [params] = useSearchParams();
  const entity = params.get("entity") ?? undefined;
  const entityId = params.get("id") ?? undefined;
  const admins = useQuery({ queryKey: ["admin", "admins"], queryFn: adminApi.admins });
  const log = useInfiniteQuery({
    queryKey: ["admin", "audit", adminFilter, entity, entityId],
    queryFn: ({ pageParam }) => adminApi.audit({ before: pageParam, admin: adminFilter || undefined, entity, entityId, limit: PAGE }),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => (last.length === PAGE ? last[last.length - 1].id : undefined),
  });
  const entries = log.data?.pages.flat() ?? [];

  return (
    <div className="admin-page">
      <header className="admin-page__head">
        <div>
          <h1>Activity log</h1>
          <p className="admin-page__sub">Everything admins have done, newest first. Times are Bangkok time.</p>
        </div>
        <label className="field field--inline">
          <span>Admin</span>
          <select value={adminFilter} onChange={(e) => setAdminFilter(e.target.value)}>
            <option value="">Everyone</option>
            {admins.data?.map((a) => (
              <option key={a.studentId} value={a.studentId}>
                {a.fullName} ({a.studentId})
              </option>
            ))}
            <option value="system">Command line (system)</option>
          </select>
        </label>
      </header>

      {entity && entityId && (
        <p className="admin-notice" role="status">
          Showing the history of one {entity}.
          <Link to="/admin/log">Show everything</Link>
        </p>
      )}
      {log.isLoading && <p className="admin-muted">Loading activity…</p>}
      {log.isError && <p className="form-error">{log.error.message}</p>}
      {log.isSuccess && entries.length === 0 && <p className="admin-muted">No activity yet.</p>}

      <ol className="audit">
        {entries.map((e) => (
          <li key={e.id} className="audit__row">
            <time className="audit__time" dateTime={e.at}>
              {formatDateTime(e.at)}
            </time>
            <span className="audit__who">
              {e.adminId === "system" ? "System" : (e.adminName ?? "Removed admin")} <code>{e.adminId}</code>
            </span>
            <span className="audit__what">{describe(e)}</span>
          </li>
        ))}
      </ol>

      {log.hasNextPage && (
        <button type="button" className="btn btn--ghost" onClick={() => log.fetchNextPage()} disabled={log.isFetchingNextPage}>
          {log.isFetchingNextPage ? "Loading…" : "Load older activity"}
        </button>
      )}
    </div>
  );
}

type Change = { from: unknown; to: unknown };

const FIELD_LABELS: Record<string, string> = {
  nameTh: "Thai name",
  descriptionMd: "description",
  lat: "location",
  lng: "location",
  nearestStation: "station",
  stationLine: "line",
  walkMinutes: "walking time",
  priceLevel: "price level",
  dressCode: "what to wear",
  etiquetteTips: "good to know",
  mustTry: "must try",
  googleMapsUrl: "Google Maps link",
  slug: "link name",
};

/** One readable sentence per entry. Unknown actions fall back to their raw parts. */
function describe(e: AuditEntry): string {
  const d = e.details as Record<string, unknown>;
  if (e.entity === "session") return e.action === "login" ? "Logged in" : "Logged out";
  if (e.entity === "admin") {
    const who = `${(d.fullName as string) ?? ""} (${e.entityId})`.trim();
    switch (e.action) {
      case "create":
        return `Added admin ${who}${d.via ? ` from the ${d.via}` : ""}`;
      case "delete":
        return `Removed admin ${who}`;
      case "change_password":
        return "Changed their own password";
      case "update": {
        const parts: string[] = [];
        const active = d.active as Change | undefined;
        if (active) parts.push(`${active.to ? "Reactivated" : "Deactivated"} admin ${e.entityId}`);
        const name = d.fullName as Change | undefined;
        if (name) parts.push(`Renamed admin ${e.entityId} from “${name.from}” to “${name.to}”`);
        if (d.password) parts.push(`Reset the password for admin ${e.entityId}`);
        return parts.join("; ") || `Updated admin ${e.entityId}`;
      }
    }
  }
  if (e.entity === "place") {
    const nameField = d.name as string | Change | undefined;
    const name = typeof nameField === "object" && nameField ? String(nameField.to) : (nameField ?? `place ${e.entityId}`);
    switch (e.action) {
      case "create":
        return `Created ${name} as ${d.status === "published" ? "published" : "a draft"}`;
      case "delete":
        return `Deleted ${name}`;
      case "add_photo":
        return `Added a photo to ${name}`;
      case "delete_photo":
        return `Removed a photo from ${name}`;
      case "update_photo":
        return `Changed a photo description on ${name}`;
      case "reorder_photos":
        return `Reordered the photos of ${name}`;
      case "update": {
        const status = d.status as Change | undefined;
        const featured = d.featured as Change | undefined;
        const parts: string[] = [];
        if (status) parts.push(status.to === "published" ? `Published ${name}` : `Unpublished ${name}`);
        if (featured) parts.push(featured.to ? `Made ${name} an editors' pick` : `Removed ${name} from editors' picks`);
        const fields = Object.keys(d).filter((k) => !["status", "featured"].includes(k) && typeof d[k] === "object" && d[k] !== null);
        if (fields.length) {
          const labels = [...new Set(fields.map((k) => FIELD_LABELS[k] ?? k))];
          parts.push(`Edited ${labels.join(", ")} of ${name}`);
        }
        return parts.join("; ") || `Updated ${name}`;
      }
    }
  }
  if (e.entity === "category") {
    const nameField = d.name as string | Change | undefined;
    const name = typeof nameField === "object" && nameField ? String(nameField.to) : (nameField ?? "a category");
    switch (e.action) {
      case "create":
        return `Added category ${name}`;
      case "delete":
        return `Deleted category ${name}`;
      case "reorder":
        return "Reordered the categories";
      case "update":
        return `Edited category ${name}`;
    }
  }
  return `${e.action} ${e.entity} ${e.entityId}`.trim();
}
