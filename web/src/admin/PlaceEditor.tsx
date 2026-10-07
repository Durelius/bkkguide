import { useQuery, useQueryClient } from "@tanstack/react-query";
import { lazy, Suspense, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Icon } from "../components/Icon";
import { useDocumentTitle } from "../useDocumentTitle";
import { adminApi, type AdminPlace, type Photo, type PlaceInput } from "./api";
import { formatDateTime } from "./format";
import { HoursEditor } from "./HoursEditor";
import { coordsFromMapsUrl } from "./mapsUrl";
import { PhotoManager } from "./PhotoManager";

const MapPicker = lazy(() => import("./MapPicker"));

const LINES = ["BTS Sukhumvit", "BTS Silom", "BTS Gold", "MRT Blue", "MRT Purple", "MRT Yellow", "MRT Pink", "Airport Rail Link", "SRT Red", "Chao Phraya Express Boat"];

const EMPTY: PlaceInput = {
  slug: "",
  category: "",
  name: "",
  nameTh: "",
  summary: "",
  descriptionMd: "",
  address: "",
  lat: NaN,
  lng: NaN,
  nearestStation: "",
  stationLine: "",
  walkMinutes: null,
  priceLevel: null,
  dressCode: "",
  etiquetteTips: "",
  mustTry: "",
  website: "",
  phone: "",
  googleMapsUrl: "",
  status: "draft",
  featured: false,
  hours: [],
};

function toInput(p: AdminPlace): PlaceInput {
  const { id: _id, photos: _p, createdAt: _c, updatedAt: _u, createdBy: _cb, updatedBy: _ub, ...input } = p;
  return input;
}

export function PlaceEditor() {
  const { id } = useParams();
  const isNew = id === "new";
  const placeId = isNew ? null : Number(id);
  const navigate = useNavigate();
  const qc = useQueryClient();

  const categories = useQuery({ queryKey: ["admin", "categories"], queryFn: adminApi.categories });
  const loaded = useQuery({ queryKey: ["admin", "place", placeId], queryFn: () => adminApi.place(placeId!), enabled: placeId != null });

  const [form, setForm] = useState<PlaceInput>(EMPTY);
  const [saved, setSaved] = useState<PlaceInput>(EMPTY);
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [meta, setMeta] = useState<AdminPlace | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  useDocumentTitle(isNew ? "New place" : form.name || "Edit place");

  useEffect(() => {
    if (loaded.data) {
      const input = toInput(loaded.data);
      setForm(input);
      setSaved(input);
      setPhotos(loaded.data.photos);
      setMeta(loaded.data);
    }
  }, [loaded.data]);

  // New places default to the first category.
  useEffect(() => {
    if (isNew && !form.category && categories.data?.[0]) {
      const category = categories.data[0].slug;
      setForm((f) => ({ ...f, category }));
      setSaved((f) => ({ ...f, category }));
    }
  }, [isNew, form.category, categories.data]);

  const dirty = JSON.stringify(form) !== JSON.stringify(saved);
  useEffect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const set = <K extends keyof PlaceInput>(key: K, value: PlaceInput[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setNotice(null);
  };

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    if (Number.isNaN(form.lat) || Number.isNaN(form.lng)) {
      setError("Set the location by clicking the map.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = placeId == null ? await adminApi.createPlace(form) : await adminApi.updatePlace(placeId, form);
      const input = toInput(result);
      setForm(input);
      setSaved(input);
      setMeta(result);
      qc.invalidateQueries({ queryKey: ["admin", "places"] });
      qc.invalidateQueries({ queryKey: ["places"] });
      qc.setQueryData(["admin", "place", result.id], result);
      if (placeId == null) {
        navigate(`/admin/places/${result.id}`, { replace: true });
        setNotice("Saved. You can add photos now.");
      } else {
        setNotice(result.status === "published" ? "Saved and live on the site." : "Saved as a draft.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Saving failed.");
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (placeId == null) return;
    try {
      await adminApi.deletePlace(placeId);
      qc.invalidateQueries({ queryKey: ["admin", "places"] });
      qc.invalidateQueries({ queryKey: ["places"] });
      navigate("/admin/places", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Deleting failed.");
    }
  };

  if (loaded.isLoading) return <p className="admin-loading">Loading place…</p>;
  if (loaded.isError) return <p className="admin-loading form-error">{loaded.error.message}</p>;

  const category = categories.data?.find((c) => c.slug === form.category);
  const hasLocation = !Number.isNaN(form.lat) && !Number.isNaN(form.lng);

  return (
    <form className="admin-page admin-page--wide editor" onSubmit={save}>
      <header className="admin-page__head">
        <div>
          <Link to="/admin/places" className="editor__back">
            <Icon name="chevron-left" /> Places
          </Link>
          <h1>{isNew ? "New place" : saved.name}</h1>
          {meta && (
            <p className="admin-page__sub">
              Last saved {formatDateTime(meta.updatedAt)}
              {meta.updatedBy ? ` by ${meta.updatedBy}` : ""}.{" "}
              <Link to={`/admin/log?entity=place&id=${meta.id}`}>History</Link>
            </p>
          )}
        </div>
        {meta?.status === "published" && (
          <a href={`/place/${meta.slug}`} target="_blank" rel="noreferrer" className="btn btn--ghost">
            <Icon name="external-link" /> View on site
          </a>
        )}
      </header>

      <section className="admin-card editor__section">
        <h2>Basics</h2>
        <div className="admin-form__grid">
          <label className="field">
            <span>Name</span>
            <input value={form.name} onChange={(e) => set("name", e.target.value)} required maxLength={120} />
          </label>
          <label className="field">
            <span>Thai name</span>
            <input className="th" lang="th" value={form.nameTh} onChange={(e) => set("nameTh", e.target.value)} />
            <small>Shown to taxi drivers on the place page.</small>
          </label>
          <label className="field">
            <span>Category</span>
            <select value={form.category} onChange={(e) => set("category", e.target.value)} required>
              {categories.data?.map((c) => (
                <option key={c.slug} value={c.slug}>
                  {c.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label className="field">
          <span>Summary</span>
          <textarea rows={2} value={form.summary} onChange={(e) => set("summary", e.target.value)} maxLength={200} />
          <small>
            One or two sentences for cards and previews. {form.summary.length}/200
          </small>
        </label>
        <label className="field">
          <span>Description</span>
          <textarea rows={6} value={form.descriptionMd} onChange={(e) => set("descriptionMd", e.target.value)} />
          <small>Shown on the place page. Leave a blank line between paragraphs.</small>
        </label>
        <label className="field field--narrow">
          <span>Link name</span>
          <span className="field__prefixed">
            <span>/place/</span>
            <input value={form.slug} onChange={(e) => set("slug", e.target.value.toLowerCase())} placeholder="made from the name" pattern="[a-z0-9]+(-[a-z0-9]+)*" />
          </span>
          <small>Changing it breaks links people have already shared.</small>
        </label>
      </section>

      <section className="admin-card editor__section">
        <h2>Location</h2>
        <p className="admin-muted">Click the map to drop the pin, then drag it to the entrance. Pasting a Google Maps link with coordinates also moves the pin.</p>
        <div className="editor__map">
          <Suspense fallback={<div className="map-picker" />}>
            <MapPicker
              lat={hasLocation ? form.lat : null}
              lng={hasLocation ? form.lng : null}
              color={category?.color ?? "#BE3570"}
              onChange={(lat, lng) => setForm((f) => ({ ...f, lat, lng }))}
            />
          </Suspense>
        </div>
        <p className="admin-muted editor__coords">{hasLocation ? `${form.lat.toFixed(6)}, ${form.lng.toFixed(6)}` : "No location set yet."}</p>
        <div className="admin-form__grid">
          <label className="field">
            <span>Google Maps link</span>
            <input
              type="url"
              value={form.googleMapsUrl}
              onChange={(e) => {
                const url = e.target.value;
                const c = coordsFromMapsUrl(url);
                setForm((f) => ({ ...f, googleMapsUrl: url, ...(c ?? {}) }));
              }}
              placeholder="https://maps.google.com/…"
            />
          </label>
          <label className="field">
            <span>Address</span>
            <input value={form.address} onChange={(e) => set("address", e.target.value)} />
          </label>
        </div>
      </section>

      <section className="admin-card editor__section">
        <h2>Getting there</h2>
        <div className="admin-form__grid">
          <label className="field">
            <span>Nearest station</span>
            <input value={form.nearestStation} onChange={(e) => set("nearestStation", e.target.value)} placeholder="Sam Yot" />
          </label>
          <label className="field">
            <span>Line</span>
            <input value={form.stationLine} onChange={(e) => set("stationLine", e.target.value)} list="lines" placeholder="MRT Blue" />
            <datalist id="lines">
              {LINES.map((l) => (
                <option key={l} value={l} />
              ))}
            </datalist>
          </label>
          <label className="field">
            <span>Walk from the station (minutes)</span>
            <input
              type="number"
              min={0}
              max={120}
              value={form.walkMinutes ?? ""}
              onChange={(e) => set("walkMinutes", e.target.value === "" ? null : Number(e.target.value))}
            />
          </label>
        </div>
      </section>

      <section className="admin-card editor__section">
        <h2>Visiting</h2>
        <fieldset className="field">
          <legend>Price level</legend>
          <div className="segmented" role="radiogroup">
            {[null, 1, 2, 3, 4].map((lvl) => (
              <button key={lvl ?? 0} type="button" role="radio" aria-checked={form.priceLevel === lvl} aria-pressed={form.priceLevel === lvl} onClick={() => set("priceLevel", lvl)}>
                {lvl ? "฿".repeat(lvl) : "Not set"}
              </button>
            ))}
          </div>
        </fieldset>
        <div className="field">
          <span>Opening hours</span>
          <HoursEditor hours={form.hours} onChange={(h) => set("hours", h)} />
        </div>
        <div className="admin-form__grid">
          <label className="field">
            <span>Website</span>
            <input type="url" value={form.website} onChange={(e) => set("website", e.target.value)} placeholder="https://" />
          </label>
          <label className="field">
            <span>Phone</span>
            <input type="tel" value={form.phone} onChange={(e) => set("phone", e.target.value)} />
          </label>
        </div>
      </section>

      <section className="admin-card editor__section">
        <h2>Before you go</h2>
        <p className="admin-muted">Each one shows as a tip on the place page. Leave empty to hide it.</p>
        <div className="admin-form__grid">
          <label className="field">
            <span>What to wear</span>
            <textarea rows={3} value={form.dressCode} onChange={(e) => set("dressCode", e.target.value)} placeholder="Shoulders and knees covered." />
          </label>
          <label className="field">
            <span>Good to know</span>
            <textarea rows={3} value={form.etiquetteTips} onChange={(e) => set("etiquetteTips", e.target.value)} />
          </label>
          <label className="field">
            <span>Must try</span>
            <textarea rows={3} value={form.mustTry} onChange={(e) => set("mustTry", e.target.value)} />
          </label>
        </div>
      </section>

      <section className="admin-card editor__section">
        <h2>Photos</h2>
        {placeId == null ? (
          <p className="admin-muted">Save the place first, then add photos.</p>
        ) : (
          <PhotoManager placeId={placeId} photos={photos} onPhotos={setPhotos} />
        )}
      </section>

      <section className="admin-card editor__section">
        <h2>Publishing</h2>
        <div className="choice-group" role="radiogroup" aria-label="Status">
          <label className="choice">
            <input type="radio" name="status" checked={form.status === "draft"} onChange={() => set("status", "draft")} />
            <span>
              <strong>Draft</strong>
              <small>Only admins can see it.</small>
            </span>
          </label>
          <label className="choice">
            <input type="radio" name="status" checked={form.status === "published"} onChange={() => set("status", "published")} />
            <span>
              <strong>Published</strong>
              <small>Visible on the site.</small>
            </span>
          </label>
        </div>
        <label className="choice">
          <input type="checkbox" checked={form.featured} onChange={(e) => set("featured", e.target.checked)} />
          <span>
            <strong>Editors' pick</strong>
            <small>Listed first and marked with a star.</small>
          </span>
        </label>
      </section>

      {placeId != null && (
        <section className="editor__danger">
          {confirmDelete ? (
            <div className="admin-row__confirm">
              <span>Delete {saved.name} and its photos? This can't be undone.</span>
              <button type="button" className="btn-small btn-small--danger" onClick={remove} autoFocus>
                Delete place
              </button>
              <button type="button" className="btn-small" onClick={() => setConfirmDelete(false)}>
                Cancel
              </button>
            </div>
          ) : (
            <button type="button" className="btn-small btn-small--danger" onClick={() => setConfirmDelete(true)}>
              <Icon name="trash" /> Delete place
            </button>
          )}
        </section>
      )}

      <div className="savebar" role="region" aria-label="Save">
        <span className={`savebar__state${error ? " form-error" : ""}`} role="status">
          {error ?? notice ?? (dirty ? "Unsaved changes" : isNew ? "Not saved yet" : "All changes saved")}
        </span>
        <button type="submit" className="btn btn--primary" disabled={saving || (!dirty && !isNew)}>
          {saving ? "Saving…" : form.status === "published" ? "Save and publish" : "Save draft"}
        </button>
      </div>
    </form>
  );
}
