package admin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Google Maps share links (maps.app.goo.gl/…) carry no coordinates, so the
// server follows their redirects to the long URL, which does.

var mapsClient = &http.Client{
	Timeout: 5 * time.Second,
	// Redirects are followed by hand so each hop's host can be checked.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

var googleHost = regexp.MustCompile(`^((www|maps|consent)\.)?google\.[a-z]{2,3}(\.[a-z]{2})?$`)

// isMapsURL reports whether u is a link worth fetching: an https Google Maps
// link or share link. Nothing else is fetched, so admins can't make the
// server request arbitrary hosts.
func isMapsURL(u *url.URL) bool {
	if u.Scheme != "https" {
		return false
	}
	switch host := strings.ToLower(u.Hostname()); {
	case host == "maps.app.goo.gl":
		return true
	case host == "goo.gl":
		return strings.HasPrefix(u.Path, "/maps")
	default:
		return googleHost.MatchString(host)
	}
}

var coordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`!3d(-?\d+\.\d+)!4d(-?\d+\.\d+)`), // the place itself
	regexp.MustCompile(`@(-?\d+\.\d+),(-?\d+\.\d+)`),     // the map's centre
	regexp.MustCompile(`[?&](?:q|query|ll)=(-?\d+\.\d+)(?:,|%2C)\s*(-?\d+\.\d+)`),
}

// coordsFromMapsURL mirrors web/src/admin/mapsUrl.ts.
func coordsFromMapsURL(s string) (lat, lng float64, ok bool) {
	for _, re := range coordPatterns {
		if m := re.FindStringSubmatch(s); m != nil {
			lat, _ = strconv.ParseFloat(m[1], 64)
			lng, _ = strconv.ParseFloat(m[2], 64)
			return lat, lng, true
		}
	}
	return 0, 0, false
}

var errNoCoords = errors.New("no coordinates in link")

// resolveMapsURL follows start's redirects until it reaches a URL with
// coordinates, and returns that URL.
func resolveMapsURL(ctx context.Context, client *http.Client, start *url.URL, allowed func(*url.URL) bool) (string, error) {
	u := start
	for hop := 0; hop <= 10; hop++ {
		// The EU cookie wall carries the real destination in ?continue=.
		if strings.EqualFold(u.Hostname(), "consent.google.com") {
			if next, err := url.Parse(u.Query().Get("continue")); err == nil && allowed(next) {
				u = next
			}
		}
		if _, _, ok := coordsFromMapsURL(u.String()); ok {
			return u.String(), nil
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bkkguide)")
		res, err := client.Do(req)
		if err != nil {
			return "", err
		}
		res.Body.Close()
		loc, err := res.Location()
		if err != nil {
			return "", errNoCoords // not a redirect, and no coordinates so far
		}
		if !allowed(loc) {
			return "", errNoCoords
		}
		u = loc
	}
	return "", errNoCoords
}

func (h *Handler) resolveMaps(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(strings.TrimSpace(r.URL.Query().Get("url")))
	if err != nil || !isMapsURL(u) || u.Hostname() == "consent.google.com" {
		writeError(w, http.StatusBadRequest, "That isn't a Google Maps link.")
		return
	}
	long, err := resolveMapsURL(r.Context(), mapsClient, u, isMapsURL)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Couldn't find coordinates in that link. Open it in a browser and paste the address bar URL instead, or drop the pin on the map.")
		return
	}
	lat, lng, _ := coordsFromMapsURL(long)
	writeJSON(w, http.StatusOK, map[string]any{"url": long, "lat": lat, "lng": lng})
}
