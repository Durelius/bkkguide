import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { Icon } from "../components/Icon";
import { adminApi, type Photo } from "./api";

type Upload = { name: string; state: "uploading" | "error"; message?: string };

const MAX_PHOTOS = 12;

export function PhotoManager({ placeId, photos, onPhotos }: { placeId: number; photos: Photo[]; onPhotos: (p: Photo[]) => void }) {
  const qc = useQueryClient();
  const input = useRef<HTMLInputElement>(null);
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [dragging, setDragging] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const changed = (next: Photo[]) => {
    onPhotos(next);
    qc.invalidateQueries({ queryKey: ["admin", "places"] });
  };

  const upload = async (files: FileList | File[]) => {
    setError(null);
    for (const file of Array.from(files)) {
      setUploads((u) => [...u, { name: file.name, state: "uploading" }]);
      try {
        const next = await adminApi.uploadPhoto(placeId, file, "");
        changed(next);
        setUploads((u) => u.filter((x) => x.name !== file.name));
      } catch (e) {
        const message = e instanceof Error ? e.message : "Upload failed.";
        setUploads((u) => u.map((x) => (x.name === file.name ? { ...x, state: "error", message } : x)));
      }
    }
  };

  const move = async (index: number, delta: number) => {
    const next = [...photos];
    const [p] = next.splice(index, 1);
    next.splice(index + delta, 0, p);
    changed(next);
    try {
      await adminApi.reorderPhotos(placeId, next.map((x) => x.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't reorder photos.");
      changed(photos);
    }
  };

  const remove = async (photo: Photo) => {
    try {
      await adminApi.deletePhoto(placeId, photo.id);
      changed(photos.filter((p) => p.id !== photo.id));
      setConfirmDelete(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't remove the photo.");
    }
  };

  const saveAlt = async (photo: Photo, alt: string) => {
    if (alt === photo.alt) return;
    try {
      await adminApi.updatePhoto(placeId, photo.id, alt);
      changed(photos.map((p) => (p.id === photo.id ? { ...p, alt } : p)));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't save the description.");
    }
  };

  const full = photos.length >= MAX_PHOTOS;

  return (
    <div className="photos">
      {photos.length > 0 && (
        <ol className="photos__grid">
          {photos.map((p, i) => (
            <li key={p.id} className="photos__item">
              <div className="photos__img">
                <img src={p.thumb} alt={p.alt} loading="lazy" />
                {i === 0 && <span className="photos__cover">Cover</span>}
              </div>
              <label className="field field--inline">
                <span className="visually-hidden">Description for screen readers</span>
                <input defaultValue={p.alt} placeholder="Describe the photo" onBlur={(e) => saveAlt(p, e.target.value.trim())} />
              </label>
              <div className="photos__tools">
                <button type="button" className="btn-small" disabled={i === 0} aria-label="Move earlier" onClick={() => move(i, -1)}>
                  <Icon name="chevron-left" />
                </button>
                <button type="button" className="btn-small" disabled={i === photos.length - 1} aria-label="Move later" onClick={() => move(i, 1)}>
                  <Icon name="chevron-right" />
                </button>
                {confirmDelete === p.id ? (
                  <>
                    <button type="button" className="btn-small btn-small--danger" onClick={() => remove(p)} autoFocus>
                      Remove
                    </button>
                    <button type="button" className="btn-small" onClick={() => setConfirmDelete(null)}>
                      Keep
                    </button>
                  </>
                ) : (
                  <button type="button" className="btn-small btn-small--danger" aria-label="Remove photo" onClick={() => setConfirmDelete(p.id)}>
                    <Icon name="trash" />
                  </button>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}

      {uploads.length > 0 && (
        <ul className="photos__uploads">
          {uploads.map((u) => (
            <li key={u.name} className={u.state === "error" ? "form-error" : "admin-muted"}>
              {u.state === "uploading" ? `Uploading ${u.name}…` : `${u.name}: ${u.message}`}
              {u.state === "error" && (
                <button type="button" className="btn-link" onClick={() => setUploads((x) => x.filter((y) => y.name !== u.name))}>
                  Dismiss
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      {!full ? (
        <div
          className={`dropzone${dragging ? " dropzone--over" : ""}`}
          onDragOver={(e) => {
            e.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragging(false);
            upload(e.dataTransfer.files);
          }}
        >
          <Icon name="image" className="dropzone__icon" />
          <p>
            Drop photos here or{" "}
            <button type="button" className="btn-link" onClick={() => input.current?.click()}>
              choose files
            </button>
          </p>
          <small className="admin-muted">JPEG, PNG, WebP or iPhone HEIC, up to 25 MB. They're compressed automatically. The first photo is the cover.</small>
          <input
            ref={input}
            type="file"
            accept="image/*,.heic,.heif"
            multiple
            hidden
            onChange={(e) => {
              if (e.target.files) upload(e.target.files);
              e.target.value = "";
            }}
          />
        </div>
      ) : (
        <p className="admin-muted">This place has the maximum of {MAX_PHOTOS} photos. Remove one to add another.</p>
      )}
      {error && <p className="form-error">{error}</p>}
    </div>
  );
}
