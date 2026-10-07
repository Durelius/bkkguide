import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi, type AuditEntry } from "./api";
import { formatDateTime } from "./format";

const PAGE = 50;

export function AuditPage() {
  useDocumentTitle("Activity log");
  const [adminFilter, setAdminFilter] = useState("");
  const admins = useQuery({ queryKey: ["admin", "admins"], queryFn: adminApi.admins });
  const log = useInfiniteQuery({
    queryKey: ["admin", "audit", adminFilter],
    queryFn: ({ pageParam }) => adminApi.audit({ before: pageParam, admin: adminFilter || undefined, limit: PAGE }),
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
  return `${e.action} ${e.entity} ${e.entityId}`.trim();
}
