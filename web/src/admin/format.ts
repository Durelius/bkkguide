// Admin timestamps are UTC from SQLite; show them in Bangkok time.
const BKK = "Asia/Bangkok";

function parse(ts: string): Date {
  // "2026-10-07 10:00:00" (datetime()) or "2026-10-07T10:00:00Z" (strftime)
  return new Date(ts.includes("T") ? ts : ts.replace(" ", "T") + "Z");
}

export function formatDate(ts: string): string {
  return parse(ts).toLocaleDateString("en-GB", { timeZone: BKK, day: "numeric", month: "short", year: "numeric" });
}

export function formatDateTime(ts: string): string {
  return parse(ts).toLocaleString("en-GB", { timeZone: BKK, day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}
