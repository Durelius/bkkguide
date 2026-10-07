package admin

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"unicode"

	"github.com/go-chi/chi/v5"

	"bkkguide/internal/audit"
	"bkkguide/internal/auth"
	"bkkguide/internal/hours"
)

// PlaceInput is everything an editor can set on a place. Field json names are
// also the keys used in audit log diffs.
type PlaceInput struct {
	Slug           string           `json:"slug"`
	Category       string           `json:"category"`
	Name           string           `json:"name"`
	NameTH         string           `json:"nameTh"`
	Summary        string           `json:"summary"`
	DescriptionMD  string           `json:"descriptionMd"`
	Address        string           `json:"address"`
	Lat            float64          `json:"lat"`
	Lng            float64          `json:"lng"`
	NearestStation string           `json:"nearestStation"`
	StationLine    string           `json:"stationLine"`
	WalkMinutes    *int             `json:"walkMinutes"`
	PriceLevel     *int             `json:"priceLevel"`
	DressCode      string           `json:"dressCode"`
	EtiquetteTips  string           `json:"etiquetteTips"`
	MustTry        string           `json:"mustTry"`
	Website        string           `json:"website"`
	Phone          string           `json:"phone"`
	GoogleMapsURL  string           `json:"googleMapsUrl"`
	Status         string           `json:"status"`
	Featured       bool             `json:"featured"`
	Hours          []hours.Interval `json:"hours"`
}

type Photo struct {
	ID        int64  `json:"id"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Alt       string `json:"alt"`
	SortOrder int    `json:"sortOrder"`
	Thumb     string `json:"thumb"`
	Large     string `json:"large"`
}

// adminPhotoURL serves photos of drafts too; the public URL only serves published places.
func adminPhotoURL(id int64, size string) string {
	return fmt.Sprintf("/api/admin/photos/%d/%s.jpg", id, size)
}

type AdminPlace struct {
	ID int64 `json:"id"`
	PlaceInput
	Photos    []Photo `json:"photos"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	CreatedBy *string `json:"createdBy"`
	UpdatedBy *string `json:"updatedBy"`
}

// PlaceRow is the list view.
type PlaceRow struct {
	ID            int64   `json:"id"`
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	NameTH        string  `json:"nameTh"`
	Category      string  `json:"category"`
	Status        string  `json:"status"`
	Featured      bool    `json:"featured"`
	UpdatedAt     string  `json:"updatedAt"`
	UpdatedBy     *string `json:"updatedBy"`
	UpdatedByName *string `json:"updatedByName"`
	PhotoCount    int     `json:"photoCount"`
	Thumb         *string `json:"thumb"`
}

func (h *Handler) placeRoutes(r chi.Router) {
	r.Get("/places", h.listPlaces)
	r.Post("/places", h.createPlace)
	r.Get("/places/{id}", h.getPlace)
	r.Put("/places/{id}", h.updatePlace)
	r.Patch("/places/{id}", h.patchPlace)
	r.Delete("/places/{id}", h.deletePlace)
	r.Post("/places/{id}/photos", h.uploadPhoto)
	r.Put("/places/{id}/photos/order", h.reorderPhotos)
	r.Patch("/places/{id}/photos/{photoId}", h.updatePhoto)
	r.Delete("/places/{id}/photos/{photoId}", h.deletePhoto)
	r.Get("/photos/{photoId}/{size}.jpg", h.servePhoto)
}

// --- reads ---

func (h *Handler) listPlaces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1 = 1"}
	args := []any{}
	if c := q.Get("category"); c != "" {
		where = append(where, "c.slug = ?")
		args = append(args, c)
	}
	if s := q.Get("status"); s != "" {
		where = append(where, "p.status = ?")
		args = append(args, s)
	}
	if term := strings.TrimSpace(q.Get("q")); term != "" {
		where = append(where, "(p.name LIKE ? OR p.name_th LIKE ? OR p.slug LIKE ?)")
		like := "%" + term + "%"
		args = append(args, like, like, like)
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT p.id, p.slug, p.name, p.name_th, c.slug, p.status, p.featured, p.updated_at, p.updated_by, a.full_name,
			(SELECT count(*) FROM photos ph WHERE ph.place_id = p.id),
			(SELECT id FROM photos ph WHERE ph.place_id = p.id ORDER BY sort_order, id LIMIT 1)
		FROM places p JOIN categories c ON c.id = p.category_id LEFT JOIN admins a ON a.student_id = p.updated_by
		WHERE `+strings.Join(where, " AND ")+` ORDER BY p.updated_at DESC, p.id DESC`, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []PlaceRow{}
	for rows.Next() {
		var p PlaceRow
		var first *int64
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.NameTH, &p.Category, &p.Status, &p.Featured, &p.UpdatedAt, &p.UpdatedBy, &p.UpdatedByName, &p.PhotoCount, &first); err != nil {
			serverError(w, err)
			return
		}
		if first != nil {
			u := adminPhotoURL(*first, "thumb")
			p.Thumb = &u
		}
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	p, err := loadPlace(r.Context(), h.DB, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "No place with that ID.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadPlace(ctx context.Context, db querier, id int64) (AdminPlace, error) {
	var p AdminPlace
	err := db.QueryRowContext(ctx, `SELECT p.id, p.slug, c.slug, p.name, p.name_th, p.summary, p.description_md, p.address, p.lat, p.lng,
			p.nearest_station, p.station_line, p.walk_minutes, p.price_level, p.dress_code, p.etiquette_tips, p.must_try,
			p.website, p.phone, p.google_maps_url, p.status, p.featured, p.created_at, p.updated_at, p.created_by, p.updated_by
		FROM places p JOIN categories c ON c.id = p.category_id WHERE p.id = ?`, id).Scan(
		&p.ID, &p.Slug, &p.Category, &p.Name, &p.NameTH, &p.Summary, &p.DescriptionMD, &p.Address, &p.Lat, &p.Lng,
		&p.NearestStation, &p.StationLine, &p.WalkMinutes, &p.PriceLevel, &p.DressCode, &p.EtiquetteTips, &p.MustTry,
		&p.Website, &p.Phone, &p.GoogleMapsURL, &p.Status, &p.Featured, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy)
	if err != nil {
		return p, err
	}
	p.Hours = []hours.Interval{}
	rows, err := db.QueryContext(ctx, `SELECT weekday, opens, closes FROM place_hours WHERE place_id = ? ORDER BY weekday, opens`, id)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var iv hours.Interval
		if err := rows.Scan(&iv.Weekday, &iv.Opens, &iv.Closes); err != nil {
			rows.Close()
			return p, err
		}
		p.Hours = append(p.Hours, iv)
	}
	rows.Close()
	p.Photos, err = loadPhotos(ctx, db, id)
	return p, err
}

func loadPhotos(ctx context.Context, db querier, placeID int64) ([]Photo, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, width, height, alt, sort_order FROM photos WHERE place_id = ? ORDER BY sort_order, id`, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var ph Photo
		if err := rows.Scan(&ph.ID, &ph.Width, &ph.Height, &ph.Alt, &ph.SortOrder); err != nil {
			return nil, err
		}
		ph.Thumb = adminPhotoURL(ph.ID, "thumb")
		ph.Large = adminPhotoURL(ph.ID, "large")
		out = append(out, ph)
	}
	return out, rows.Err()
}

// --- writes ---

func (h *Handler) createPlace(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	var in PlaceInput
	if !decode(w, r, &in) {
		return
	}
	if in.Status == "" {
		in.Status = "draft"
	}
	var id int64
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		catID, problem := normalizePlace(r.Context(), tx, &in, 0)
		if problem != "" {
			status, msg = http.StatusBadRequest, problem
			return errAbort
		}
		res, err := tx.ExecContext(r.Context(), `INSERT INTO places
			(slug, category_id, name, name_th, summary, description_md, address, lat, lng, nearest_station, station_line, walk_minutes,
			 price_level, dress_code, etiquette_tips, must_try, website, phone, google_maps_url, status, featured, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.Slug, catID, in.Name, in.NameTH, in.Summary, in.DescriptionMD, in.Address, in.Lat, in.Lng, in.NearestStation, in.StationLine,
			in.WalkMinutes, in.PriceLevel, in.DressCode, in.EtiquetteTips, in.MustTry, in.Website, in.Phone, in.GoogleMapsURL,
			in.Status, in.Featured, me.StudentID, me.StudentID)
		if err != nil {
			return err
		}
		id, _ = res.LastInsertId()
		if err := writeHours(r.Context(), tx, id, in.Hours); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "create", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": in.Name, "slug": in.Slug, "category": in.Category, "status": in.Status})
	})
	if errors.Is(err, errAbort) {
		writeError(w, status, msg)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	p, err := loadPlace(r.Context(), h.DB, id)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePlace(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in PlaceInput
	if !decode(w, r, &in) {
		return
	}
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		cur, err := loadPlace(r.Context(), tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No place with that ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		catID, problem := normalizePlace(r.Context(), tx, &in, id)
		if problem != "" {
			status, msg = http.StatusBadRequest, problem
			return errAbort
		}
		changes := diffPlace(cur.PlaceInput, in)
		if len(changes) == 0 {
			return nil
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE places SET
			slug = ?, category_id = ?, name = ?, name_th = ?, summary = ?, description_md = ?, address = ?, lat = ?, lng = ?,
			nearest_station = ?, station_line = ?, walk_minutes = ?, price_level = ?, dress_code = ?, etiquette_tips = ?, must_try = ?,
			website = ?, phone = ?, google_maps_url = ?, status = ?, featured = ?, updated_by = ?, updated_at = datetime('now')
			WHERE id = ?`,
			in.Slug, catID, in.Name, in.NameTH, in.Summary, in.DescriptionMD, in.Address, in.Lat, in.Lng, in.NearestStation, in.StationLine,
			in.WalkMinutes, in.PriceLevel, in.DressCode, in.EtiquetteTips, in.MustTry, in.Website, in.Phone, in.GoogleMapsURL,
			in.Status, in.Featured, me.StudentID, id); err != nil {
			return err
		}
		if _, ok := changes["hours"]; ok {
			if _, err := tx.ExecContext(r.Context(), `DELETE FROM place_hours WHERE place_id = ?`, id); err != nil {
				return err
			}
			if err := writeHours(r.Context(), tx, id, in.Hours); err != nil {
				return err
			}
		}
		changes["name"] = changeOrName(changes["name"], cur.Name)
		return audit.Log(r.Context(), tx, me.StudentID, "update", "place", strconv.FormatInt(id, 10), changes)
	})
	if errors.Is(err, errAbort) {
		writeError(w, status, msg)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	p, err := loadPlace(r.Context(), h.DB, id)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// patchPlace handles the quick toggles in the list: status and featured.
func (h *Handler) patchPlace(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Status   *string `json:"status"`
		Featured *bool   `json:"featured"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Status != nil && *in.Status != "draft" && *in.Status != "published" {
		writeError(w, http.StatusBadRequest, "Status must be draft or published.")
		return
	}
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var name, curStatus string
		var curFeatured bool
		err := tx.QueryRowContext(r.Context(), `SELECT name, status, featured FROM places WHERE id = ?`, id).Scan(&name, &curStatus, &curFeatured)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No place with that ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		changes := map[string]any{}
		if in.Status != nil && *in.Status != curStatus {
			changes["status"] = audit.Change{From: curStatus, To: *in.Status}
		} else {
			in.Status = &curStatus
		}
		if in.Featured != nil && *in.Featured != curFeatured {
			changes["featured"] = audit.Change{From: curFeatured, To: *in.Featured}
		} else {
			in.Featured = &curFeatured
		}
		if len(changes) == 0 {
			return nil
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE places SET status = ?, featured = ?, updated_by = ?, updated_at = datetime('now') WHERE id = ?`,
			*in.Status, *in.Featured, me.StudentID, id); err != nil {
			return err
		}
		changes["name"] = name
		return audit.Log(r.Context(), tx, me.StudentID, "update", "place", strconv.FormatInt(id, 10), changes)
	})
	if errors.Is(err, errAbort) {
		writeError(w, status, msg)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletePlace(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		cur, err := loadPlace(r.Context(), tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No place with that ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM places WHERE id = ?`, id); err != nil {
			return err
		}
		// Keep a full snapshot so a deleted place can be reconstructed from the log.
		return audit.Log(r.Context(), tx, me.StudentID, "delete", "place", strconv.FormatInt(id, 10),
			map[string]any{"name": cur.Name, "snapshot": cur.PlaceInput, "photoCount": len(cur.Photos)})
	})
	if errors.Is(err, errAbort) {
		writeError(w, status, msg)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeHours(ctx context.Context, tx *sql.Tx, placeID int64, ivs []hours.Interval) error {
	for _, iv := range ivs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO place_hours (place_id, weekday, opens, closes) VALUES (?, ?, ?, ?)`,
			placeID, iv.Weekday, iv.Opens, iv.Closes); err != nil {
			return err
		}
	}
	return nil
}

// --- validation ---

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

// Greater Bangkok, generously. Catches swapped or mistyped coordinates.
const minLat, maxLat, minLng, maxLng = 13.3, 14.3, 100.0, 101.1

// normalizePlace trims and validates in, fills in a unique slug, and returns
// the category ID or a message for the editor.
func normalizePlace(ctx context.Context, tx *sql.Tx, in *PlaceInput, selfID int64) (int64, string) {
	for _, s := range []*string{&in.Slug, &in.Category, &in.Name, &in.NameTH, &in.Summary, &in.Address, &in.NearestStation,
		&in.StationLine, &in.DressCode, &in.EtiquetteTips, &in.MustTry, &in.Website, &in.Phone, &in.GoogleMapsURL} {
		*s = strings.TrimSpace(*s)
	}
	in.DescriptionMD = strings.TrimSpace(strings.ReplaceAll(in.DescriptionMD, "\r\n", "\n"))

	switch {
	case in.Name == "":
		return 0, "Name is required."
	case len([]rune(in.Name)) > 120:
		return 0, "Name can be at most 120 characters."
	case len([]rune(in.Summary)) > 200:
		return 0, "Summary can be at most 200 characters."
	case in.Lat < minLat || in.Lat > maxLat || in.Lng < minLng || in.Lng > maxLng:
		return 0, "The location is outside Bangkok. Set it by clicking the map."
	case in.Status != "draft" && in.Status != "published":
		return 0, "Status must be draft or published."
	case in.PriceLevel != nil && (*in.PriceLevel < 1 || *in.PriceLevel > 4):
		return 0, "Price level must be 1 to 4."
	case in.WalkMinutes != nil && (*in.WalkMinutes < 0 || *in.WalkMinutes > 120):
		return 0, "Walking time must be 0 to 120 minutes."
	}
	for label, u := range map[string]string{"Website": in.Website, "Google Maps link": in.GoogleMapsURL} {
		if u == "" {
			continue
		}
		parsed, err := url.Parse(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return 0, label + " must be a full link starting with https://."
		}
	}

	var catID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = ?`, in.Category).Scan(&catID); err != nil {
		return 0, "Choose a category."
	}

	if in.Hours == nil {
		in.Hours = []hours.Interval{}
	}
	for _, iv := range in.Hours {
		if iv.Weekday < 0 || iv.Weekday > 6 || !timePattern.MatchString(iv.Opens) || !timePattern.MatchString(iv.Closes) {
			return 0, "Opening hours need valid times (HH:MM)."
		}
		if iv.Opens == iv.Closes {
			return 0, "An opening time can't be the same as its closing time."
		}
	}
	sort.SliceStable(in.Hours, func(i, j int) bool {
		if in.Hours[i].Weekday != in.Hours[j].Weekday {
			return in.Hours[i].Weekday < in.Hours[j].Weekday
		}
		return in.Hours[i].Opens < in.Hours[j].Opens
	})

	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	} else if !slugPattern.MatchString(in.Slug) {
		return 0, "The link name can only use lowercase letters, digits and dashes."
	}
	slug, err := uniqueSlug(ctx, tx, in.Slug, selfID)
	if err != nil {
		return 0, "Couldn't pick a link name. Try again."
	}
	in.Slug = slug
	return catID, ""
}

func slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		// Names written only in Thai have no ASCII to build from.
		buf := make([]byte, 3)
		rand.Read(buf)
		s = "place-" + hex.EncodeToString(buf)
	}
	return s
}

func uniqueSlug(ctx context.Context, tx *sql.Tx, base string, selfID int64) (string, error) {
	for n := 1; n < 100; n++ {
		candidate := base
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d", base, n)
		}
		var taken int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM places WHERE slug = ? AND id != ?`, candidate, selfID).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			return candidate, nil
		}
	}
	return "", errors.New("no free slug")
}

// diffPlace lists changed fields as {field: {from, to}}, keyed by json name.
func diffPlace(old, new PlaceInput) map[string]any {
	out := map[string]any{}
	ov, nv := reflect.ValueOf(old), reflect.ValueOf(new)
	t := ov.Type()
	for i := 0; i < t.NumField(); i++ {
		key := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		a, b := ov.Field(i).Interface(), nv.Field(i).Interface()
		if key == "hours" {
			if formatHours(old.Hours) != formatHours(new.Hours) {
				out[key] = audit.Change{From: formatHours(old.Hours), To: formatHours(new.Hours)}
			}
			continue
		}
		if !reflect.DeepEqual(deref(a), deref(b)) {
			out[key] = audit.Change{From: deref(a), To: deref(b)}
		}
	}
	return out
}

func deref(v any) any {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return v
}

func formatHours(ivs []hours.Interval) string {
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	parts := make([]string, 0, len(ivs))
	for _, iv := range ivs {
		parts = append(parts, fmt.Sprintf("%s %s–%s", days[iv.Weekday], iv.Opens, iv.Closes))
	}
	return strings.Join(parts, ", ")
}

// changeOrName keeps a name change as-is, or records the current name so
// every log line can say which place it was.
func changeOrName(change any, name string) any {
	if change != nil {
		return change
	}
	return name
}

func pathID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Not found.")
		return 0, false
	}
	return id, true
}
