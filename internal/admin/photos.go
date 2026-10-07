package admin

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"bkkguide/internal/audit"
	"bkkguide/internal/auth"
	"bkkguide/internal/media"
)

const maxPhotosPerPlace = 12

// uploadPhoto takes multipart field "file" (+ optional "alt") in any format
// ffmpeg reads, HEIC included, and stores compressed JPEGs (see media).
func (h *Handler) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "That photo is too big. The limit is 25 MB.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Choose a photo to upload.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, media.MaxUploadBytes+1))
	if err != nil {
		serverError(w, err)
		return
	}
	if len(data) > media.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "That photo is too big. The limit is 25 MB.")
		return
	}
	alt := strings.TrimSpace(r.FormValue("alt"))

	var name string
	var count int
	err = h.DB.QueryRowContext(r.Context(), `SELECT name, (SELECT count(*) FROM photos WHERE place_id = places.id) FROM places WHERE id = ?`, id).Scan(&name, &count)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "No place with that ID.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if count >= maxPhotosPerPlace {
		writeError(w, http.StatusBadRequest, "A place can have at most 12 photos. Remove one first.")
		return
	}

	saved, err := media.Process(r.Context(), data)
	if err != nil {
		if errors.Is(err, media.ErrNotImage) || strings.Contains(err.Error(), "too large") || strings.Contains(err.Error(), "too long") {
			writeError(w, http.StatusBadRequest, capitalize(err.Error())+".")
			return
		}
		serverError(w, err)
		return
	}

	var photoID int64
	err = h.tx(r.Context(), func(tx *sql.Tx) error {
		res, err := tx.ExecContext(r.Context(), `INSERT INTO photos (place_id, thumb, large, width, height, alt, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, (SELECT coalesce(max(sort_order), -1) + 1 FROM photos WHERE place_id = ?))`,
			id, saved.Thumb, saved.Large, saved.Width, saved.Height, alt, id)
		if err != nil {
			return err
		}
		photoID, _ = res.LastInsertId()
		if err := touchPlace(r, tx, id, me.StudentID); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "add_photo", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": name, "photoId": photoID, "bytes": len(saved.Thumb) + len(saved.Large)})
	})
	if err != nil {
		serverError(w, err)
		return
	}
	photos, err := loadPhotos(r.Context(), h.DB, id)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, photos)
}

func (h *Handler) updatePhoto(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	photoID, ok := pathID(w, r, "photoId")
	if !ok {
		return
	}
	var in struct {
		Alt string `json:"alt"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Alt = strings.TrimSpace(in.Alt)
	notFound := false
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var cur, name string
		err := tx.QueryRowContext(r.Context(), `SELECT ph.alt, p.name FROM photos ph JOIN places p ON p.id = ph.place_id WHERE ph.id = ? AND ph.place_id = ?`, photoID, id).Scan(&cur, &name)
		if errors.Is(err, sql.ErrNoRows) {
			notFound = true
			return errAbort
		}
		if err != nil || cur == in.Alt {
			return err
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE photos SET alt = ? WHERE id = ?`, in.Alt, photoID); err != nil {
			return err
		}
		if err := touchPlace(r, tx, id, me.StudentID); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "update_photo", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": name, "photoId": photoID, "alt": audit.Change{From: cur, To: in.Alt}})
	})
	if notFound {
		writeError(w, http.StatusNotFound, "No such photo.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// reorderPhotos takes the full list of photo IDs in their new order.
func (h *Handler) reorderPhotos(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	bad := false
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		cur, err := loadPhotos(r.Context(), tx, id)
		if err != nil {
			return err
		}
		if len(cur) != len(in.IDs) {
			bad = true
			return errAbort
		}
		have := map[int64]bool{}
		before := make([]int64, len(cur))
		for i, p := range cur {
			have[p.ID] = true
			before[i] = p.ID
		}
		for i, pid := range in.IDs {
			if !have[pid] {
				bad = true
				return errAbort
			}
			if _, err := tx.ExecContext(r.Context(), `UPDATE photos SET sort_order = ? WHERE id = ?`, i, pid); err != nil {
				return err
			}
		}
		var name string
		tx.QueryRowContext(r.Context(), `SELECT name FROM places WHERE id = ?`, id).Scan(&name)
		if err := touchPlace(r, tx, id, me.StudentID); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "reorder_photos", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": name, "order": audit.Change{From: before, To: in.IDs}})
	})
	if bad {
		writeError(w, http.StatusBadRequest, "The photo list is out of date. Reload the page and try again.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletePhoto(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	photoID, ok := pathID(w, r, "photoId")
	if !ok {
		return
	}
	notFound := false
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var name, alt string
		err := tx.QueryRowContext(r.Context(), `SELECT ph.alt, p.name FROM photos ph JOIN places p ON p.id = ph.place_id WHERE ph.id = ? AND ph.place_id = ?`, photoID, id).Scan(&alt, &name)
		if errors.Is(err, sql.ErrNoRows) {
			notFound = true
			return errAbort
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM photos WHERE id = ?`, photoID); err != nil {
			return err
		}
		if err := touchPlace(r, tx, id, me.StudentID); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "delete_photo", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": name, "photoId": photoID, "alt": alt})
	})
	if notFound {
		writeError(w, http.StatusNotFound, "No such photo.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// servePhoto serves any photo, including drafts', to logged-in admins.
func (h *Handler) servePhoto(w http.ResponseWriter, r *http.Request) {
	media.ServePhoto(w, r, h.DB, false)
}

func touchPlace(r *http.Request, tx *sql.Tx, placeID int64, by string) error {
	_, err := tx.ExecContext(r.Context(), `UPDATE places SET updated_by = ?, updated_at = datetime('now') WHERE id = ?`, by, placeID)
	return err
}
