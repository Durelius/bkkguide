package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCoordsFromMapsURL(t *testing.T) {
	cases := []struct {
		url      string
		lat, lng float64
		ok       bool
	}{
		{"https://www.google.com/maps/place/Jay+Fai/@13.7525,100.5040,17z/data=!3m1!4b1!4m6!3m5!1s0x0:0x0!8m2!3d13.7524!4d100.5045", 13.7524, 100.5045, true},
		{"https://www.google.com/maps/@13.75,100.53,15z", 13.75, 100.53, true},
		{"https://maps.google.com/?q=13.7,100.5", 13.7, 100.5, true},
		{"https://www.google.com/maps?q=13.7%2C100.5", 13.7, 100.5, true},
		{"https://www.google.com/maps?q=Jay+Fai", 0, 0, false},
	}
	for _, c := range cases {
		lat, lng, ok := coordsFromMapsURL(c.url)
		if ok != c.ok || lat != c.lat || lng != c.lng {
			t.Errorf("%s: got %v,%v,%v want %v,%v,%v", c.url, lat, lng, ok, c.lat, c.lng, c.ok)
		}
	}
}

func TestIsMapsURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://maps.app.goo.gl/abc":          true,
		"https://goo.gl/maps/abc":              true,
		"https://goo.gl/other":                 false,
		"https://www.google.com/maps/place/x":  true,
		"https://www.google.co.th/maps":        true,
		"http://maps.app.goo.gl/abc":           false,
		"https://evil.com/maps":                false,
		"https://google.com.evil.com/maps":     false,
		"https://localhost/maps":               false,
		"https://maps.app.goo.gl.evil.com/abc": false,
	} {
		u, _ := url.Parse(raw)
		if got := isMapsURL(u); got != want {
			t.Errorf("%s: got %v want %v", raw, got, want)
		}
	}
}

func TestResolveMapsURL(t *testing.T) {
	long := "/maps/place/Jay+Fai/data=!3d13.7524!4d100.5045"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/short":
			http.Redirect(w, r, "/hop", http.StatusFound)
		case "/hop":
			http.Redirect(w, r, long, http.StatusMovedPermanently)
		case "/offsite":
			http.Redirect(w, r, "https://evil.com/maps/@1.0,2.0", http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	base, _ := url.Parse(srv.URL)
	sameHost := func(u *url.URL) bool { return u.Host == base.Host }
	start := func(p string) *url.URL { return base.ResolveReference(&url.URL{Path: p}) }

	got, err := resolveMapsURL(context.Background(), mapsClient, start("/short"), sameHost)
	if err != nil || got != srv.URL+long {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := resolveMapsURL(context.Background(), mapsClient, start("/offsite"), sameHost); err == nil {
		t.Fatal("followed a redirect to a disallowed host")
	}
	if _, err := resolveMapsURL(context.Background(), mapsClient, start("/nowhere"), sameHost); err == nil {
		t.Fatal("expected an error for a page without coordinates")
	}
}
