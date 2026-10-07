import type { Interval } from "./api";

// Display Monday first; the API counts 0 = Sunday.
const DAYS: { label: string; weekday: number }[] = [
  { label: "Mon", weekday: 1 },
  { label: "Tue", weekday: 2 },
  { label: "Wed", weekday: 3 },
  { label: "Thu", weekday: 4 },
  { label: "Fri", weekday: 5 },
  { label: "Sat", weekday: 6 },
  { label: "Sun", weekday: 0 },
];

type Props = { hours: Interval[]; onChange: (hours: Interval[]) => void };

export function HoursEditor({ hours, onChange }: Props) {
  const forDay = (wd: number) => hours.filter((h) => h.weekday === wd);
  const setDay = (wd: number, slots: Interval[]) => onChange([...hours.filter((h) => h.weekday !== wd), ...slots]);

  const copyFirstOpenDay = () => {
    const source = DAYS.map((d) => forDay(d.weekday)).find((s) => s.length > 0);
    if (!source) return;
    onChange(DAYS.flatMap((d) => source.map((s) => ({ ...s, weekday: d.weekday }))));
  };

  return (
    <div className="hours-editor">
      {DAYS.map(({ label, weekday }) => {
        const slots = forDay(weekday);
        const open = slots.length > 0;
        return (
          <div key={weekday} className="hours-editor__day">
            <label className="hours-editor__toggle">
              <input
                type="checkbox"
                checked={open}
                onChange={(e) => setDay(weekday, e.target.checked ? [{ weekday, opens: "09:00", closes: "18:00" }] : [])}
              />
              <span>{label}</span>
            </label>
            {!open && <span className="admin-muted">Closed</span>}
            <div className="hours-editor__slots">
              {slots.map((slot, i) => (
                <span key={i} className="hours-editor__slot">
                  <input
                    type="time"
                    aria-label={`${label} opens`}
                    value={slot.opens}
                    onChange={(e) => setDay(weekday, slots.map((s, j) => (j === i ? { ...s, opens: e.target.value } : s)))}
                    required
                  />
                  <span aria-hidden>–</span>
                  <input
                    type="time"
                    aria-label={`${label} closes`}
                    value={slot.closes}
                    onChange={(e) => setDay(weekday, slots.map((s, j) => (j === i ? { ...s, closes: e.target.value } : s)))}
                    required
                  />
                  {slots.length > 1 && (
                    <button type="button" className="btn-small" aria-label={`Remove ${label} hours ${i + 1}`} onClick={() => setDay(weekday, slots.filter((_, j) => j !== i))}>
                      Remove
                    </button>
                  )}
                </span>
              ))}
              {open && slots.length < 3 && (
                <button type="button" className="btn-link" onClick={() => setDay(weekday, [...slots, { weekday, opens: "17:00", closes: "22:00" }])}>
                  + add hours
                </button>
              )}
            </div>
          </div>
        );
      })}
      <div className="hours-editor__foot">
        <button type="button" className="btn-small" onClick={copyFirstOpenDay}>
          Copy first open day to every day
        </button>
        <small className="admin-muted">Open past midnight? Enter it as is, for example 18:00–02:00.</small>
      </div>
    </div>
  );
}
