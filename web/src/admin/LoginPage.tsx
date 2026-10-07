import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { SunriseMark } from "../components/Logo";
import { SITE_NAME } from "../config";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi } from "./api";

export function LoginPage() {
  useDocumentTitle("Admin login");
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [studentId, setStudentId] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const next = params.get("next")?.startsWith("/admin") ? params.get("next")! : "/admin";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const me = await adminApi.login(studentId, password);
      qc.setQueryData(["admin", "me"], me);
      navigate(next, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed.");
      setPassword("");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login">
      <form className="login__card" onSubmit={submit}>
        <SunriseMark size={72} />
        <h1>Admin login</h1>
        <p className="login__sub">For the people who keep {SITE_NAME} up to date.</p>
        <label className="field">
          <span>Student ID</span>
          <input value={studentId} onChange={(e) => setStudentId(e.target.value)} autoComplete="username" autoCapitalize="off" spellCheck={false} required autoFocus />
        </label>
        <label className="field">
          <span>Password</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
        </label>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <button type="submit" className="btn btn--primary login__submit" disabled={busy}>
          {busy ? "Logging in…" : "Log in"}
        </button>
        <Link to="/" className="login__back">
          Back to the guide
        </Link>
      </form>
    </div>
  );
}
