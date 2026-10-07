package media

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ServePhoto writes one size of a photo from the database. Routes must
// provide {photoId} and {size}. With publishedOnly, photos of draft places
// are 404s. Photo bytes never change for an ID, so they cache forever.
func ServePhoto(w http.ResponseWriter, r *http.Request, db *sql.DB, publishedOnly bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "photoId"), 10, 64)
	size := chi.URLParam(r, "size")
	if err != nil || (size != "thumb" && size != "large") {
		http.NotFound(w, r)
		return
	}
	q := `SELECT ` + size + ` FROM photos ph JOIN places p ON p.id = ph.place_id WHERE ph.id = ?`
	if publishedOnly {
		q += ` AND p.status = 'published'`
	}
	var data []byte
	err = db.QueryRowContext(r.Context(), q, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if publishedOnly {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	w.Write(data)
}
