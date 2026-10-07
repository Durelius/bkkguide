package admin

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"bkkguide/internal/auth"
	"bkkguide/internal/media"
)

func (e *env) seedCategory() {
	e.t.Helper()
	if _, err := e.db.Exec(`INSERT INTO categories (slug, name, icon, color) VALUES ('food', 'Food', 'x', '#BE3570')`); err != nil {
		e.t.Fatal(err)
	}
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return v
}

func validPlace() map[string]any {
	return map[string]any{
		"category": "food", "name": "Jay Fai", "lat": 13.7524, "lng": 100.5045,
		"hours": []map[string]any{{"weekday": 1, "opens": "09:00", "closes": "19:00"}},
	}
}

func TestPlaceLifecycle(t *testing.T) {
	e := newEnv(t)
	e.seedCategory()
	me := e.login("s100", "first-admin-pw")

	// Validation: outside Bangkok, bad times.
	bad := validPlace()
	bad["lat"], bad["lng"] = 100.5, 13.7 // swapped
	e.expect(e.do("POST", "/places", me, bad), http.StatusBadRequest)
	bad = validPlace()
	bad["hours"] = []map[string]any{{"weekday": 1, "opens": "9am", "closes": "19:00"}}
	e.expect(e.do("POST", "/places", me, bad), http.StatusBadRequest)

	rec := e.do("POST", "/places", me, validPlace())
	e.expect(rec, http.StatusCreated)
	p := decodeBody[AdminPlace](t, rec)
	if p.Slug != "jay-fai" || p.Status != "draft" || len(p.Hours) != 1 || *p.CreatedBy != "s100" {
		t.Fatalf("created = %+v", p)
	}
	// Same name gets a unique slug.
	second := decodeBody[AdminPlace](t, e.do("POST", "/places", me, validPlace()))
	if second.Slug != "jay-fai-2" {
		t.Fatalf("second slug = %q", second.Slug)
	}

	// Update records only changed fields, with before/after.
	upd := p.PlaceInput
	upd.Summary = "Crab omelette"
	upd.Hours[0].Closes = "20:00"
	e.expect(e.do("PUT", "/places/"+itoa(p.ID), me, upd), http.StatusOK)
	var details string
	e.db.QueryRow(`SELECT details FROM audit_log WHERE action = 'update' AND entity = 'place' ORDER BY id DESC LIMIT 1`).Scan(&details)
	var d map[string]json.RawMessage
	json.Unmarshal([]byte(details), &d)
	if _, ok := d["summary"]; !ok {
		t.Errorf("audit missing summary change: %s", details)
	}
	if _, ok := d["hours"]; !ok {
		t.Errorf("audit missing hours change: %s", details)
	}
	if _, ok := d["lat"]; ok {
		t.Errorf("audit has unchanged lat: %s", details)
	}

	// Photo: upload, hidden publicly while draft, visible once published.
	if !media.Available() {
		t.Skip("ffmpeg not installed")
	}
	photoID := uploadTestPhoto(t, e, me, p.ID)
	public := chi.NewRouter()
	public.Get("/media/photos/{photoId}/{size}.jpg", func(w http.ResponseWriter, r *http.Request) { media.ServePhoto(w, r, e.db, true) })
	get := func() int {
		rec := httptest.NewRecorder()
		public.ServeHTTP(rec, httptest.NewRequest("GET", "/media/photos/"+itoa(photoID)+"/thumb.jpg", nil))
		return rec.Code
	}
	if code := get(); code != http.StatusNotFound {
		t.Fatalf("draft photo served publicly: %d", code)
	}
	e.expect(e.do("PATCH", "/places/"+itoa(p.ID), me, map[string]string{"status": "published"}), http.StatusNoContent)
	if code := get(); code != http.StatusOK {
		t.Fatalf("published photo: %d", code)
	}

	// Delete removes the place and its photos; the log keeps a snapshot.
	e.expect(e.do("DELETE", "/places/"+itoa(p.ID), me, nil), http.StatusNoContent)
	var photos int
	e.db.QueryRow(`SELECT count(*) FROM photos`).Scan(&photos)
	if photos != 0 {
		t.Errorf("photos left after delete: %d", photos)
	}
	var snap string
	e.db.QueryRow(`SELECT details FROM audit_log WHERE action = 'delete' AND entity = 'place'`).Scan(&snap)
	if !bytes.Contains([]byte(snap), []byte(`"snapshot"`)) {
		t.Errorf("delete log has no snapshot: %s", snap)
	}
}

func TestCategoryInUseCannotBeDeleted(t *testing.T) {
	e := newEnv(t)
	e.seedCategory()
	me := e.login("s100", "first-admin-pw")
	e.expect(e.do("POST", "/places", me, validPlace()), http.StatusCreated)
	var catID int64
	e.db.QueryRow(`SELECT id FROM categories WHERE slug = 'food'`).Scan(&catID)
	e.expect(e.do("DELETE", "/categories/"+itoa(catID), me, nil), http.StatusBadRequest)
	e.expect(e.do("POST", "/categories", me, map[string]string{"name": "Night life", "icon": "y", "color": "#123456"}), http.StatusCreated)
}

func uploadTestPhoto(t *testing.T, e *env, cookie string, placeID int64) int64 {
	t.Helper()
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 800, 600)))
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "photo.png")
	fw.Write(img.Bytes())
	mw.WriteField("alt", "Front of the shop")
	mw.Close()
	req := httptest.NewRequest("POST", "/places/"+itoa(placeID)+"/photos", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.CSRFHeader, "1")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	e.expect(rec, http.StatusCreated)
	photos := decodeBody[[]Photo](t, rec)
	if len(photos) != 1 || photos[0].Width != 800 || photos[0].Alt != "Front of the shop" {
		t.Fatalf("photos = %+v", photos)
	}
	return photos[0].ID
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
