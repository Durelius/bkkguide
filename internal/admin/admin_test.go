package admin

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"bkkguide/internal/auth"
	"bkkguide/internal/db"
)

type env struct {
	t   *testing.T
	db  *sql.DB
	srv http.Handler
}

func newEnv(t *testing.T) *env {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	hash, _ := auth.HashPassword("first-admin-pw")
	if _, err := conn.Exec(`INSERT INTO admins (student_id, full_name, password_hash) VALUES ('s100', 'First Admin', ?)`, hash); err != nil {
		t.Fatal(err)
	}
	h := &Handler{DB: conn, Auth: &auth.Service{DB: conn, Now: time.Now}, Limiter: auth.NewLoginLimiter()}
	return &env{t: t, db: conn, srv: h.Routes()}
}

// do sends a JSON request with the CSRF header and an optional session cookie.
func (e *env) do(method, path, cookie string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(auth.CSRFHeader, "1")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec
}

func (e *env) login(id, pw string) string {
	rec := e.do("POST", "/login", "", map[string]string{"studentId": id, "password": pw})
	if rec.Code != http.StatusOK {
		e.t.Fatalf("login %s: %d %s", id, rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c.Value
		}
	}
	e.t.Fatal("no session cookie")
	return ""
}

func (e *env) expect(rec *httptest.ResponseRecorder, code int) {
	e.t.Helper()
	if rec.Code != code {
		e.t.Fatalf("got %d, want %d: %s", rec.Code, code, rec.Body)
	}
}

func TestLoginAndSessions(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("POST", "/login", "", map[string]string{"studentId": "s100", "password": "wrong-password"}), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized)

	cookie := e.login("S100", "first-admin-pw") // student IDs are case-insensitive
	e.expect(e.do("GET", "/me", cookie, nil), http.StatusOK)

	// Mutations need the CSRF header.
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	e.expect(rec, http.StatusForbidden)

	e.expect(e.do("POST", "/logout", cookie, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/me", cookie, nil), http.StatusUnauthorized)
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 5; i++ {
		e.do("POST", "/login", "", map[string]string{"studentId": "s100", "password": "wrong-password"})
	}
	e.expect(e.do("POST", "/login", "", map[string]string{"studentId": "s100", "password": "first-admin-pw"}), http.StatusTooManyRequests)
}

func TestAdminManagementRules(t *testing.T) {
	e := newEnv(t)
	me := e.login("s100", "first-admin-pw")

	e.expect(e.do("POST", "/admins", me, map[string]string{"studentId": "s200", "fullName": "Second", "password": "short"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/admins", me, map[string]string{"studentId": "s200", "fullName": "Second", "password": "second-admin-pw"}), http.StatusCreated)
	e.expect(e.do("POST", "/admins", me, map[string]string{"studentId": "S200", "fullName": "Dupe", "password": "second-admin-pw"}), http.StatusConflict)

	// No self-removal or self-deactivation.
	e.expect(e.do("DELETE", "/admins/s100", me, nil), http.StatusBadRequest)
	e.expect(e.do("PATCH", "/admins/s100", me, map[string]bool{"active": false}), http.StatusBadRequest)

	// Deactivating ends the other admin's sessions and blocks login.
	other := e.login("s200", "second-admin-pw")
	e.expect(e.do("PATCH", "/admins/s200", me, map[string]bool{"active": false}), http.StatusNoContent)
	e.expect(e.do("GET", "/me", other, nil), http.StatusUnauthorized)
	e.expect(e.do("POST", "/login", "", map[string]string{"studentId": "s200", "password": "second-admin-pw"}), http.StatusUnauthorized)

	e.expect(e.do("DELETE", "/admins/s200", me, nil), http.StatusNoContent)
	e.expect(e.do("DELETE", "/admins/s200", me, nil), http.StatusNotFound)

	// Every action is in the log, attributed to s100, and survives the removal.
	var actions []string
	rows, _ := e.db.Query(`SELECT action || ':' || entity || ':' || entity_id FROM audit_log WHERE admin_id = 's100' ORDER BY id`)
	for rows.Next() {
		var a string
		rows.Scan(&a)
		actions = append(actions, a)
	}
	want := []string{"login:session:", "create:admin:s200", "update:admin:s200", "delete:admin:s200"}
	if len(actions) != len(want) {
		t.Fatalf("audit = %v, want %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("audit = %v, want %v", actions, want)
		}
	}
}
