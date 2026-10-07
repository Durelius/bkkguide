// Package api serves the public JSON API under /api.
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"bkkguide/internal/hours"
	"bkkguide/web"
)

type API struct {
	DB   *sql.DB
	Site web.Site
	Now  func() time.Time
}

func (a *API) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/config", a.config)
	r.Get("/categories", a.categories)
	r.Get("/places", a.places)
	r.Get("/places/{slug}", a.place)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return r
}

type Category struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	SortOrder int    `json:"sortOrder"`
}

type Place struct {
	ID             int64            `json:"id"`
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
	Featured       bool             `json:"featured"`
	Hours          []hours.Interval `json:"hours"`
	OpenNow        bool             `json:"openNow"`
	ClosesAt       string           `json:"closesAt,omitempty"`
}

func (a *API) config(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.Site)
}

func (a *API) categories(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id, slug, name, icon, color, sort_order FROM categories ORDER BY sort_order, name`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.Icon, &c.Color, &c.SortOrder); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, out)
}

const placeColumns = `p.id, p.slug, c.slug, p.name, p.name_th, p.summary, p.description_md, p.address, p.lat, p.lng,
	p.nearest_station, p.station_line, p.walk_minutes, p.price_level, p.dress_code, p.etiquette_tips, p.must_try,
	p.website, p.phone, p.google_maps_url, p.featured`

func scanPlace(s interface{ Scan(...any) error }) (Place, error) {
	var p Place
	err := s.Scan(&p.ID, &p.Slug, &p.Category, &p.Name, &p.NameTH, &p.Summary, &p.DescriptionMD, &p.Address, &p.Lat, &p.Lng,
		&p.NearestStation, &p.StationLine, &p.WalkMinutes, &p.PriceLevel, &p.DressCode, &p.EtiquetteTips, &p.MustTry,
		&p.Website, &p.Phone, &p.GoogleMapsURL, &p.Featured)
	return p, err
}

// places lists published places, optionally filtered by ?category=slug.
func (a *API) places(w http.ResponseWriter, r *http.Request) {
	q := `SELECT ` + placeColumns + ` FROM places p JOIN categories c ON c.id = p.category_id
		WHERE p.status = 'published'`
	args := []any{}
	if cat := r.URL.Query().Get("category"); cat != "" {
		q += ` AND c.slug = ?`
		args = append(args, cat)
	}
	q += ` ORDER BY p.featured DESC, p.name`

	rows, err := a.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []Place{}
	for rows.Next() {
		p, err := scanPlace(rows)
		if err != nil {
			serverError(w, err)
			return
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	if err := a.attachHours(r, out); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, out)
}

func (a *API) place(w http.ResponseWriter, r *http.Request) {
	row := a.DB.QueryRowContext(r.Context(), `SELECT `+placeColumns+` FROM places p JOIN categories c ON c.id = p.category_id
		WHERE p.status = 'published' AND p.slug = ?`, chi.URLParam(r, "slug"))
	p, err := scanPlace(row)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "place not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	list := []Place{p}
	if err := a.attachHours(r, list); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, list[0])
}

// attachHours loads hours for all given places in one query and sets OpenNow.
func (a *API) attachHours(r *http.Request, places []Place) error {
	if len(places) == 0 {
		return nil
	}
	idx := make(map[int64]int, len(places))
	for i := range places {
		idx[places[i].ID] = i
		places[i].Hours = []hours.Interval{}
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT place_id, weekday, opens, closes FROM place_hours ORDER BY weekday, opens`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var iv hours.Interval
		if err := rows.Scan(&id, &iv.Weekday, &iv.Opens, &iv.Closes); err != nil {
			return err
		}
		if i, ok := idx[id]; ok {
			places[i].Hours = append(places[i].Hours, iv)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := a.Now()
	for i := range places {
		places[i].OpenNow, places[i].ClosesAt = hours.Status(places[i].Hours, now)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("api error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
