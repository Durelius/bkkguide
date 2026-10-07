package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"bkkguide/internal/audit"
	"bkkguide/internal/auth"
)

type AdminCategory struct {
	ID         int64  `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Color      string `json:"color"`
	SortOrder  int    `json:"sortOrder"`
	PlaceCount int    `json:"placeCount"`
}

type categoryInput struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (h *Handler) categoryRoutes(r chi.Router) {
	r.Get("/categories", h.listCategories)
	r.Post("/categories", h.createCategory)
	r.Put("/categories/order", h.reorderCategories)
	r.Put("/categories/{id}", h.updateCategory)
	r.Delete("/categories/{id}", h.deleteCategory)
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT c.id, c.slug, c.name, c.icon, c.color, c.sort_order,
		(SELECT count(*) FROM places p WHERE p.category_id = c.id) FROM categories c ORDER BY c.sort_order, c.name`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []AdminCategory{}
	for rows.Next() {
		var c AdminCategory
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.Icon, &c.Color, &c.SortOrder, &c.PlaceCount); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, out)
}

func validateCategory(in *categoryInput) string {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)
	in.Icon = strings.TrimSpace(in.Icon)
	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	}
	switch {
	case in.Name == "" || len([]rune(in.Name)) > 40:
		return "Name is required (at most 40 characters)."
	case !slugPattern.MatchString(in.Slug):
		return "The link name can only use lowercase letters, digits and dashes."
	case in.Icon == "" || len([]rune(in.Icon)) > 2:
		return "Choose an icon."
	case !colorPattern.MatchString(in.Color):
		return "Colour must look like #BE3570."
	}
	in.Color = strings.ToUpper(in.Color)
	return ""
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	var in categoryInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateCategory(&in); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	var id int64
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		res, err := tx.ExecContext(r.Context(), `INSERT INTO categories (slug, name, icon, color, sort_order)
			VALUES (?, ?, ?, ?, (SELECT coalesce(max(sort_order), -1) + 1 FROM categories))`, in.Slug, in.Name, in.Icon, in.Color)
		if err != nil {
			return err
		}
		id, _ = res.LastInsertId()
		return audit.Log(r.Context(), tx, me.StudentID, "create", "category", strconv.FormatInt(id, 10), in)
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		writeError(w, http.StatusConflict, "Another category already uses that link name.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, AdminCategory{ID: id, Slug: in.Slug, Name: in.Name, Icon: in.Icon, Color: in.Color})
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in categoryInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateCategory(&in); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	notFound := false
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var cur categoryInput
		err := tx.QueryRowContext(r.Context(), `SELECT slug, name, icon, color FROM categories WHERE id = ?`, id).Scan(&cur.Slug, &cur.Name, &cur.Icon, &cur.Color)
		if errors.Is(err, sql.ErrNoRows) {
			notFound = true
			return errAbort
		}
		if err != nil {
			return err
		}
		changes := map[string]any{}
		for _, f := range []struct{ key, from, to string }{{"slug", cur.Slug, in.Slug}, {"name", cur.Name, in.Name}, {"icon", cur.Icon, in.Icon}, {"color", cur.Color, in.Color}} {
			if f.from != f.to {
				changes[f.key] = audit.Change{From: f.from, To: f.to}
			}
		}
		if len(changes) == 0 {
			return nil
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE categories SET slug = ?, name = ?, icon = ?, color = ? WHERE id = ?`, in.Slug, in.Name, in.Icon, in.Color, id); err != nil {
			return err
		}
		if _, ok := changes["name"]; !ok {
			changes["name"] = cur.Name
		}
		return audit.Log(r.Context(), tx, me.StudentID, "update", "category", strconv.FormatInt(id, 10), changes)
	})
	if notFound {
		writeError(w, http.StatusNotFound, "No category with that ID.")
		return
	}
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		writeError(w, http.StatusConflict, "Another category already uses that link name.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorderCategories(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	bad := false
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var total int
		if err := tx.QueryRowContext(r.Context(), `SELECT count(*) FROM categories`).Scan(&total); err != nil {
			return err
		}
		if total != len(in.IDs) {
			bad = true
			return errAbort
		}
		for i, cid := range in.IDs {
			res, err := tx.ExecContext(r.Context(), `UPDATE categories SET sort_order = ? WHERE id = ?`, i, cid)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n != 1 {
				bad = true
				return errAbort
			}
		}
		return audit.Log(r.Context(), tx, me.StudentID, "reorder", "category", "", map[string]any{"order": in.IDs})
	})
	if bad {
		writeError(w, http.StatusBadRequest, "The category list is out of date. Reload the page and try again.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var cur categoryInput
		var places int
		err := tx.QueryRowContext(r.Context(), `SELECT slug, name, icon, color, (SELECT count(*) FROM places WHERE category_id = categories.id) FROM categories WHERE id = ?`, id).
			Scan(&cur.Slug, &cur.Name, &cur.Icon, &cur.Color, &places)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No category with that ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		if places > 0 {
			status, msg = http.StatusBadRequest, "Move or delete the places in this category first."
			return errAbort
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM categories WHERE id = ?`, id); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "delete", "category", strconv.FormatInt(id, 10), cur)
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
