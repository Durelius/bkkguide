import { Link } from "react-router-dom";
import { useDocumentTitle } from "../useDocumentTitle";

/** Placeholder for routes built in later milestones. */
export function ComingSoon({ title }: { title: string }) {
  useDocumentTitle(title);
  return (
    <div style={{ padding: "var(--s6) var(--s4)", maxWidth: "60ch" }}>
      <h1 style={{ fontSize: "var(--fs-xl)", fontWeight: 800, marginBottom: "var(--s3)" }}>{title}</h1>
      <p style={{ color: "var(--ink-muted)", marginBottom: "var(--s4)" }}>This page is next on the build list.</p>
      <Link to="/" className="btn btn--ghost">
        Back to the map
      </Link>
    </div>
  );
}
