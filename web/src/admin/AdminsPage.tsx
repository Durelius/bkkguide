import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { useDocumentTitle } from "../useDocumentTitle";
import { useMe } from "./AdminApp";
import { adminApi, type Admin } from "./api";
import { formatDate } from "./format";

type RowMode = "view" | "confirm-remove" | "reset-password";

export function AdminsPage() {
  useDocumentTitle("Admins");
  const me = useMe();
  const qc = useQueryClient();
  const admins = useQuery({ queryKey: ["admin", "admins"], queryFn: adminApi.admins });
  const [adding, setAdding] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const refresh = (message?: string) => {
    qc.invalidateQueries({ queryKey: ["admin", "admins"] });
    qc.invalidateQueries({ queryKey: ["admin", "audit"] });
    if (message) setNotice(message);
  };

  return (
    <div className="admin-page">
      <header className="admin-page__head">
        <div>
          <h1>Admins</h1>
          <p className="admin-page__sub">People who can edit the guide. They log in with their student ID.</p>
        </div>
        {!adding && (
          <button type="button" className="btn btn--primary" onClick={() => setAdding(true)}>
            <Icon name="plus" /> Add admin
          </button>
        )}
      </header>

      {notice && (
        <p className="admin-notice" role="status">
          {notice}
          <button type="button" className="admin-notice__close" aria-label="Dismiss" onClick={() => setNotice(null)}>
            <Icon name="close" />
          </button>
        </p>
      )}

      {adding && (
        <AddAdminForm
          onDone={(created) => {
            setAdding(false);
            if (created) refresh(`Added ${created.fullName}. Share their temporary password with them in person.`);
          }}
        />
      )}

      {admins.isLoading && <p className="admin-muted">Loading admins…</p>}
      {admins.isError && <p className="form-error">{admins.error.message}</p>}
      {admins.data && (
        <ul className="admin-list">
          {admins.data.map((a) => (
            <AdminRow key={a.studentId} admin={a} isMe={a.studentId.toLowerCase() === me.data?.studentId.toLowerCase()} onChanged={refresh} />
          ))}
        </ul>
      )}
    </div>
  );
}

function AdminRow({ admin, isMe, onChanged }: { admin: Admin; isMe: boolean; onChanged: (msg?: string) => void }) {
  const [mode, setMode] = useState<RowMode>("view");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const run = useMutation({
    mutationFn: (fn: () => Promise<void>) => fn(),
    onSuccess: () => setError(null),
    onError: (e: Error) => setError(e.message),
  });

  const toggleActive = () =>
    run.mutate(() => adminApi.updateAdmin(admin.studentId, { active: !admin.active }), {
      onSuccess: () => onChanged(admin.active ? `Deactivated ${admin.fullName}. They've been logged out.` : `Reactivated ${admin.fullName}.`),
    });
  const remove = () =>
    run.mutate(() => adminApi.deleteAdmin(admin.studentId), { onSuccess: () => onChanged(`Removed ${admin.fullName}. Their past actions stay in the activity log.`) });
  const resetPassword = (e: React.FormEvent) => {
    e.preventDefault();
    run.mutate(() => adminApi.updateAdmin(admin.studentId, { password }), {
      onSuccess: () => {
        setMode("view");
        setPassword("");
        onChanged(`Reset the password for ${admin.fullName}. They've been logged out everywhere.`);
      },
    });
  };

  return (
    <li className={`admin-row${admin.active ? "" : " admin-row--inactive"}`}>
      <div className="admin-row__who">
        <span className="admin-row__name">
          {admin.fullName}
          {isMe && <span className="admin-row__you">you</span>}
        </span>
        <span className="admin-row__meta">
          <code>{admin.studentId}</code>
          <span>added {formatDate(admin.createdAt)}</span>
        </span>
      </div>
      <span className={`status ${admin.active ? "status--on" : "status--off"}`}>{admin.active ? "Active" : "Inactive"}</span>

      {mode === "view" && !isMe && (
        <div className="admin-row__actions">
          <button type="button" className="btn-small" onClick={toggleActive} disabled={run.isPending}>
            {admin.active ? "Deactivate" : "Reactivate"}
          </button>
          <button type="button" className="btn-small" onClick={() => setMode("reset-password")}>
            Reset password
          </button>
          <button type="button" className="btn-small btn-small--danger" onClick={() => setMode("confirm-remove")}>
            Remove
          </button>
        </div>
      )}
      {mode === "view" && isMe && (
        <div className="admin-row__actions">
          <Link to="/admin/account" className="btn-small">
            Edit account
          </Link>
        </div>
      )}

      {mode === "confirm-remove" && (
        <div className="admin-row__confirm" role="group" aria-label={`Remove ${admin.fullName}`}>
          <span>Remove {admin.fullName}? This can't be undone.</span>
          <button type="button" className="btn-small btn-small--danger" onClick={remove} disabled={run.isPending} autoFocus>
            Remove
          </button>
          <button type="button" className="btn-small" onClick={() => setMode("view")}>
            Cancel
          </button>
        </div>
      )}

      {mode === "reset-password" && (
        <form className="admin-row__confirm" onSubmit={resetPassword}>
          <label className="field field--inline">
            <span>New password for {admin.fullName}</span>
            <input type="text" value={password} onChange={(e) => setPassword(e.target.value)} minLength={10} required autoFocus autoComplete="off" />
          </label>
          <button type="submit" className="btn-small btn-small--primary" disabled={run.isPending}>
            Set password
          </button>
          <button type="button" className="btn-small" onClick={() => setMode("view")}>
            Cancel
          </button>
        </form>
      )}

      {error && <p className="form-error admin-row__error">{error}</p>}
    </li>
  );
}

function AddAdminForm({ onDone }: { onDone: (created?: Admin) => void }) {
  const [studentId, setStudentId] = useState("");
  const [fullName, setFullName] = useState("");
  const [password, setPassword] = useState("");
  const create = useMutation({ mutationFn: adminApi.createAdmin, onSuccess: (a) => onDone(a) });

  return (
    <form
      className="admin-card admin-form"
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate({ studentId, fullName, password });
      }}
    >
      <h2>New admin</h2>
      <div className="admin-form__grid">
        <label className="field">
          <span>Student ID</span>
          <input value={studentId} onChange={(e) => setStudentId(e.target.value)} required autoFocus autoCapitalize="off" spellCheck={false} />
        </label>
        <label className="field">
          <span>Full name</span>
          <input value={fullName} onChange={(e) => setFullName(e.target.value)} required />
        </label>
        <label className="field">
          <span>Temporary password</span>
          <input type="text" value={password} onChange={(e) => setPassword(e.target.value)} minLength={10} required autoComplete="off" />
          <small>At least 10 characters. They can change it under Account.</small>
        </label>
      </div>
      {create.error && <p className="form-error">{create.error.message}</p>}
      <div className="admin-form__actions">
        <button type="submit" className="btn btn--primary" disabled={create.isPending}>
          {create.isPending ? "Adding…" : "Add admin"}
        </button>
        <button type="button" className="btn btn--ghost" onClick={() => onDone()}>
          Cancel
        </button>
      </div>
    </form>
  );
}
