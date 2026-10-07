// Package admin serves the authenticated admin API under /api/admin.
package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"bkkguide/internal/audit"
	"bkkguide/internal/auth"
)

type Handler struct {
	DB      *sql.DB
	Auth    *auth.Service
	Limiter *auth.LoginLimiter
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/login", h.login)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.Require)
		r.Post("/logout", h.logout)
		r.Get("/me", h.me)
		r.Put("/me/password", h.changeOwnPassword)
		r.Get("/admins", h.listAdmins)
		r.Post("/admins", h.createAdmin)
		r.Patch("/admins/{id}", h.updateAdmin)
		r.Delete("/admins/{id}", h.deleteAdmin)
		r.Get("/audit", h.listAudit)
	})
	return r
}

// --- session ---

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	// Same CSRF header as authenticated requests, so other sites can't log a visitor in.
	if r.Header.Get(auth.CSRFHeader) != "1" {
		writeError(w, http.StatusForbidden, "Missing "+auth.CSRFHeader+" header.")
		return
	}
	var in struct {
		StudentID string `json:"studentId"`
		Password  string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	now := time.Now()
	idKey := "id:" + strings.ToLower(strings.TrimSpace(in.StudentID))
	ipKey := "ip:" + auth.ClientIP(r)
	if !h.Limiter.Allowed(now, idKey, ipKey) {
		writeError(w, http.StatusTooManyRequests, "Too many failed attempts. Wait 15 minutes and try again.")
		return
	}
	a, token, err := h.Auth.Login(r.Context(), in.StudentID, in.Password)
	if errors.Is(err, auth.ErrBadCredentials) {
		h.Limiter.Fail(now, idKey, ipKey)
		writeError(w, http.StatusUnauthorized, "Wrong student ID or password.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	h.Limiter.Reset(idKey)
	audit.Log(r.Context(), h.DB, a.StudentID, "login", "session", "", map[string]string{"ip": auth.ClientIP(r)})
	h.Auth.SetCookie(w, token)
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	a, _ := auth.Current(r.Context())
	if c, err := r.Cookie(auth.CookieName); err == nil {
		h.Auth.Logout(r.Context(), c.Value)
	}
	audit.Log(r.Context(), h.DB, a.StudentID, "logout", "session", "", nil)
	h.Auth.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	a, _ := auth.Current(r.Context())
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) changeOwnPassword(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !decode(w, r, &in) {
		return
	}
	var hash string
	if err := h.DB.QueryRowContext(r.Context(), `SELECT password_hash FROM admins WHERE student_id = ?`, me.StudentID).Scan(&hash); err != nil {
		serverError(w, err)
		return
	}
	if ok, _ := auth.CheckPassword(hash, in.Current); !ok {
		writeError(w, http.StatusBadRequest, "Your current password is wrong.")
		return
	}
	newHash, err := auth.HashPassword(in.New)
	if err != nil {
		writeError(w, http.StatusBadRequest, capitalize(err.Error())+".")
		return
	}
	err = h.tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(r.Context(), `UPDATE admins SET password_hash = ? WHERE student_id = ?`, newHash, me.StudentID); err != nil {
			return err
		}
		// Sign out other devices; keep this session.
		c, _ := r.Cookie(auth.CookieName)
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM sessions WHERE student_id = ? AND token_hash != ?`, me.StudentID, auth.HashToken(c.Value)); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "change_password", "admin", me.StudentID, nil)
	})
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- admins ---

func (h *Handler) listAdmins(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT student_id, full_name, active, created_at FROM admins ORDER BY active DESC, full_name`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []auth.Admin{}
	for rows.Next() {
		var a auth.Admin
		if err := rows.Scan(&a.StudentID, &a.FullName, &a.Active, &a.CreatedAt); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, a)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) createAdmin(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	var in struct {
		StudentID string `json:"studentId"`
		FullName  string `json:"fullName"`
		Password  string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	id, err := auth.NormalizeStudentID(in.StudentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, capitalize(err.Error())+".")
		return
	}
	name := strings.TrimSpace(in.FullName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Full name is required.")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, capitalize(err.Error())+".")
		return
	}
	err = h.tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(r.Context(), `INSERT INTO admins (student_id, full_name, password_hash) VALUES (?, ?, ?)`, id, name, hash); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "create", "admin", id, map[string]string{"fullName": name})
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "PRIMARY KEY") {
			writeError(w, http.StatusConflict, "An admin with that student ID already exists.")
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, auth.Admin{StudentID: id, FullName: name, Active: true})
}

func (h *Handler) updateAdmin(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id := chi.URLParam(r, "id")
	var in struct {
		FullName *string `json:"fullName"`
		Active   *bool   `json:"active"`
		Password *string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Active != nil && !*in.Active && strings.EqualFold(id, me.StudentID) {
		writeError(w, http.StatusBadRequest, "You can't deactivate yourself.")
		return
	}
	var newHash string
	if in.Password != nil {
		var err error
		if newHash, err = auth.HashPassword(*in.Password); err != nil {
			writeError(w, http.StatusBadRequest, capitalize(err.Error())+".")
			return
		}
	}

	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var cur auth.Admin
		err := tx.QueryRowContext(r.Context(), `SELECT student_id, full_name, active FROM admins WHERE student_id = ?`, id).Scan(&cur.StudentID, &cur.FullName, &cur.Active)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No admin with that student ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		changes := map[string]any{}
		if in.FullName != nil {
			name := strings.TrimSpace(*in.FullName)
			if name == "" {
				status, msg = http.StatusBadRequest, "Full name is required."
				return errAbort
			}
			if name != cur.FullName {
				if _, err := tx.ExecContext(r.Context(), `UPDATE admins SET full_name = ? WHERE student_id = ?`, name, cur.StudentID); err != nil {
					return err
				}
				changes["fullName"] = audit.Change{From: cur.FullName, To: name}
			}
		}
		if in.Active != nil && *in.Active != cur.Active {
			if !*in.Active {
				if last, err := isLastActive(r.Context(), tx, cur.StudentID); err != nil {
					return err
				} else if last {
					status, msg = http.StatusBadRequest, "At least one admin must stay active."
					return errAbort
				}
			}
			if _, err := tx.ExecContext(r.Context(), `UPDATE admins SET active = ? WHERE student_id = ?`, *in.Active, cur.StudentID); err != nil {
				return err
			}
			if !*in.Active {
				if _, err := tx.ExecContext(r.Context(), `DELETE FROM sessions WHERE student_id = ?`, cur.StudentID); err != nil {
					return err
				}
			}
			changes["active"] = audit.Change{From: cur.Active, To: *in.Active}
		}
		if in.Password != nil {
			if _, err := tx.ExecContext(r.Context(), `UPDATE admins SET password_hash = ? WHERE student_id = ?`, newHash, cur.StudentID); err != nil {
				return err
			}
			if !strings.EqualFold(cur.StudentID, me.StudentID) {
				if _, err := tx.ExecContext(r.Context(), `DELETE FROM sessions WHERE student_id = ?`, cur.StudentID); err != nil {
					return err
				}
			}
			changes["password"] = "reset"
		}
		if len(changes) == 0 {
			return nil
		}
		return audit.Log(r.Context(), tx, me.StudentID, "update", "admin", cur.StudentID, changes)
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

func (h *Handler) deleteAdmin(w http.ResponseWriter, r *http.Request) {
	me, _ := auth.Current(r.Context())
	id := chi.URLParam(r, "id")
	if strings.EqualFold(id, me.StudentID) {
		writeError(w, http.StatusBadRequest, "You can't remove yourself.")
		return
	}
	var status int
	var msg string
	err := h.tx(r.Context(), func(tx *sql.Tx) error {
		var cur auth.Admin
		err := tx.QueryRowContext(r.Context(), `SELECT student_id, full_name, active FROM admins WHERE student_id = ?`, id).Scan(&cur.StudentID, &cur.FullName, &cur.Active)
		if errors.Is(err, sql.ErrNoRows) {
			status, msg = http.StatusNotFound, "No admin with that student ID."
			return errAbort
		}
		if err != nil {
			return err
		}
		if cur.Active {
			if last, err := isLastActive(r.Context(), tx, cur.StudentID); err != nil {
				return err
			} else if last {
				status, msg = http.StatusBadRequest, "At least one admin must stay active."
				return errAbort
			}
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM admins WHERE student_id = ?`, cur.StudentID); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, me.StudentID, "delete", "admin", cur.StudentID, map[string]any{"fullName": cur.FullName, "wasActive": cur.Active})
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

// isLastActive guards against two admins deactivating each other at the same
// time; self-deactivation is already refused, so this is a safety net.
func isLastActive(ctx context.Context, tx *sql.Tx, studentID string) (bool, error) {
	var others int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM admins WHERE active = 1 AND student_id != ?`, studentID).Scan(&others)
	return others == 0, err
}

// --- audit log ---

type AuditEntry struct {
	ID        int64           `json:"id"`
	At        string          `json:"at"`
	AdminID   string          `json:"adminId"`
	AdminName *string         `json:"adminName"` // null once the admin is removed
	Action    string          `json:"action"`
	Entity    string          `json:"entity"`
	EntityID  string          `json:"entityId"`
	Details   json.RawMessage `json:"details"`
}

// listAudit pages newest-first: ?before=<id>&limit=<n>&admin=<studentId>&entity=<name>.
func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where := []string{"1 = 1"}
	args := []any{}
	if before, err := strconv.ParseInt(q.Get("before"), 10, 64); err == nil {
		where = append(where, "l.id < ?")
		args = append(args, before)
	}
	if a := q.Get("admin"); a != "" {
		where = append(where, "l.admin_id = ?")
		args = append(args, a)
	}
	if e := q.Get("entity"); e != "" {
		where = append(where, "l.entity = ?")
		args = append(args, e)
	}
	args = append(args, limit)
	rows, err := h.DB.QueryContext(r.Context(), `SELECT l.id, l.at, l.admin_id, a.full_name, l.action, l.entity, l.entity_id, l.details
		FROM audit_log l LEFT JOIN admins a ON a.student_id = l.admin_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY l.id DESC LIMIT ?`, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var details string
		if err := rows.Scan(&e.ID, &e.At, &e.AdminID, &e.AdminName, &e.Action, &e.Entity, &e.EntityID, &details); err != nil {
			serverError(w, err)
			return
		}
		e.Details = json.RawMessage(details)
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- helpers ---

var errAbort = errors.New("abort")

func (h *Handler) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("admin api error: %v", err)
	writeError(w, http.StatusInternalServerError, "Something went wrong on the server.")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
