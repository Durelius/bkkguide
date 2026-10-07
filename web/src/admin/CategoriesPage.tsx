import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Glyph, Icon } from "../components/Icon";
import { glyph, iconNames } from "../icons";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi, type AdminCategory, type CategoryInput } from "./api";

const PRESETS = ["#BE3570", "#C4532F", "#8A3F86", "#2F6F8F", "#2E7D5B", "#9A6A12", "#5B4B9A"];

export function CategoriesPage() {
  useDocumentTitle("Categories");
  const qc = useQueryClient();
  const categories = useQuery({ queryKey: ["admin", "categories"], queryFn: adminApi.categories });
  const [editing, setEditing] = useState<number | "new" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["admin", "categories"] });
    qc.invalidateQueries({ queryKey: ["categories"] });
    qc.invalidateQueries({ queryKey: ["admin", "audit"] });
  };

  const move = async (index: number, delta: number) => {
    const list = [...(categories.data ?? [])];
    const [c] = list.splice(index, 1);
    list.splice(index + delta, 0, c);
    qc.setQueryData(["admin", "categories"], list);
    try {
      await adminApi.reorderCategories(list.map((x) => x.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't reorder.");
    }
    refresh();
  };

  return (
    <div className="admin-page">
      <header className="admin-page__head">
        <div>
          <h1>Categories</h1>
          <p className="admin-page__sub">Categories appear in the navigation in this order.</p>
        </div>
        {editing !== "new" && (
          <button type="button" className="btn btn--primary" onClick={() => setEditing("new")}>
            <Icon name="plus" /> Add category
          </button>
        )}
      </header>
      {error && <p className="form-error">{error}</p>}
      {editing === "new" && (
        <CategoryForm
          onCancel={() => setEditing(null)}
          onSave={async (input) => {
            await adminApi.createCategory(input);
            setEditing(null);
            refresh();
          }}
        />
      )}
      <ul className="admin-list">
        {categories.data?.map((c, i) =>
          editing === c.id ? (
            <li key={c.id}>
              <CategoryForm
                initial={c}
                onCancel={() => setEditing(null)}
                onSave={async (input) => {
                  await adminApi.updateCategory(c.id, input);
                  setEditing(null);
                  refresh();
                }}
              />
            </li>
          ) : (
            <CategoryRow
              key={c.id}
              category={c}
              first={i === 0}
              last={i === (categories.data?.length ?? 0) - 1}
              onEdit={() => setEditing(c.id)}
              onMove={(d) => move(i, d)}
              onDeleted={refresh}
            />
          ),
        )}
      </ul>
    </div>
  );
}

function CategoryRow({
  category: c,
  first,
  last,
  onEdit,
  onMove,
  onDeleted,
}: {
  category: AdminCategory;
  first: boolean;
  last: boolean;
  onEdit: () => void;
  onMove: (delta: number) => void;
  onDeleted: () => void;
}) {
  const [confirm, setConfirm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <li className="admin-row">
      <div className="admin-row__who cat-row">
        <span className="cat-swatch" style={{ background: c.color }}>
          <Glyph char={c.icon} />
        </span>
        <span>
          <span className="admin-row__name">{c.name}</span>
          <span className="admin-row__meta">
            <code>/c/{c.slug}</code>
            <span>
              {c.placeCount} {c.placeCount === 1 ? "place" : "places"}
            </span>
          </span>
        </span>
      </div>
      <div className="admin-row__actions">
        <button type="button" className="btn-small" disabled={first} aria-label={`Move ${c.name} up`} onClick={() => onMove(-1)}>
          <Icon name="chevron-up" />
        </button>
        <button type="button" className="btn-small" disabled={last} aria-label={`Move ${c.name} down`} onClick={() => onMove(1)}>
          <Icon name="chevron-down" />
        </button>
        <button type="button" className="btn-small" onClick={onEdit}>
          Edit
        </button>
        {confirm ? (
          <>
            <button
              type="button"
              className="btn-small btn-small--danger"
              autoFocus
              onClick={async () => {
                try {
                  await adminApi.deleteCategory(c.id);
                  onDeleted();
                } catch (e) {
                  setError(e instanceof Error ? e.message : "Couldn't delete.");
                  setConfirm(false);
                }
              }}
            >
              Delete {c.name}
            </button>
            <button type="button" className="btn-small" onClick={() => setConfirm(false)}>
              Cancel
            </button>
          </>
        ) : (
          <button
            type="button"
            className="btn-small btn-small--danger"
            disabled={c.placeCount > 0}
            title={c.placeCount > 0 ? "Move or delete its places first" : undefined}
            onClick={() => setConfirm(true)}
          >
            Delete
          </button>
        )}
      </div>
      {error && <p className="form-error admin-row__error">{error}</p>}
    </li>
  );
}

function CategoryForm({ initial, onSave, onCancel }: { initial?: AdminCategory; onSave: (c: CategoryInput) => Promise<void>; onCancel: () => void }) {
  const [form, setForm] = useState<CategoryInput>({
    name: initial?.name ?? "",
    slug: initial?.slug ?? "",
    icon: initial?.icon ?? glyph("star"),
    color: initial?.color ?? PRESETS[3],
  });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  return (
    <form
      className="admin-card admin-form"
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        setError(null);
        try {
          await onSave(form);
        } catch (err) {
          setError(err instanceof Error ? err.message : "Saving failed.");
        } finally {
          setBusy(false);
        }
      }}
    >
      <h2>{initial ? `Edit ${initial.name}` : "New category"}</h2>
      <div className="admin-form__grid">
        <label className="field">
          <span>Name</span>
          <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required maxLength={40} autoFocus />
        </label>
        <label className="field">
          <span>Link name</span>
          <span className="field__prefixed">
            <span>/c/</span>
            <input value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase() })} placeholder="made from the name" />
          </span>
        </label>
      </div>
      <fieldset className="field">
        <legend>Colour</legend>
        <div className="swatches">
          {PRESETS.map((c) => (
            <button
              key={c}
              type="button"
              className="swatch"
              style={{ background: c }}
              aria-label={c}
              aria-pressed={form.color.toUpperCase() === c}
              onClick={() => setForm({ ...form, color: c })}
            />
          ))}
          <input type="color" aria-label="Custom colour" value={form.color} onChange={(e) => setForm({ ...form, color: e.target.value.toUpperCase() })} />
        </div>
        <small>Pick a dark enough colour: icons are drawn in white on it.</small>
      </fieldset>
      <fieldset className="field">
        <legend>Icon</legend>
        <div className="icon-picker" role="radiogroup">
          {iconNames.map((name) => {
            const g = glyph(name);
            return (
              <button
                key={name}
                type="button"
                role="radio"
                aria-checked={form.icon === g}
                aria-label={name}
                title={name}
                className="icon-picker__item"
                style={form.icon === g ? { background: form.color, color: "#fff" } : undefined}
                onClick={() => setForm({ ...form, icon: g })}
              >
                <span className="nf" aria-hidden>
                  {g}
                </span>
              </button>
            );
          })}
        </div>
      </fieldset>
      {error && <p className="form-error">{error}</p>}
      <div className="admin-form__actions">
        <button type="submit" className="btn btn--primary" disabled={busy}>
          {busy ? "Saving…" : "Save category"}
        </button>
        <button type="button" className="btn btn--ghost" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}
