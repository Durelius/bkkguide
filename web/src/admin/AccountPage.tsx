import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { useDocumentTitle } from "../useDocumentTitle";
import { useMe } from "./AdminApp";
import { adminApi } from "./api";

export function AccountPage() {
  useDocumentTitle("Account");
  const me = useMe();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [repeat, setRepeat] = useState("");
  const [mismatch, setMismatch] = useState(false);
  const change = useMutation({
    mutationFn: () => adminApi.changePassword(current, next),
    onSuccess: () => {
      setCurrent("");
      setNext("");
      setRepeat("");
    },
  });

  return (
    <div className="admin-page">
      <header className="admin-page__head">
        <div>
          <h1>Account</h1>
          <p className="admin-page__sub">
            {me.data?.fullName} <code>{me.data?.studentId}</code>
          </p>
        </div>
      </header>
      <form
        className="admin-card admin-form admin-form--narrow"
        onSubmit={(e) => {
          e.preventDefault();
          if (next !== repeat) {
            setMismatch(true);
            return;
          }
          setMismatch(false);
          change.mutate();
        }}
      >
        <h2>Change password</h2>
        <label className="field">
          <span>Current password</span>
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" required />
        </label>
        <label className="field">
          <span>New password</span>
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" minLength={10} required />
          <small>At least 10 characters. Other devices will be logged out.</small>
        </label>
        <label className="field">
          <span>Repeat new password</span>
          <input type="password" value={repeat} onChange={(e) => setRepeat(e.target.value)} autoComplete="new-password" required />
        </label>
        {mismatch && <p className="form-error">The new passwords don't match.</p>}
        {change.error && <p className="form-error">{change.error.message}</p>}
        {change.isSuccess && (
          <p className="admin-notice" role="status">
            Password changed.
          </p>
        )}
        <div className="admin-form__actions">
          <button type="submit" className="btn btn--primary" disabled={change.isPending}>
            {change.isPending ? "Saving…" : "Change password"}
          </button>
        </div>
      </form>
    </div>
  );
}
