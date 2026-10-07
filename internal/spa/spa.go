// Package spa serves the built frontend, falling back to index.html for
// client-side routes.
package spa

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Handler serves static files from dist. Hashed assets get long cache headers;
// unknown paths get index.html so the React router can handle them.
func Handler(dist fs.FS) http.Handler {
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p != "" && p != "index.html" {
			if f, err := fs.Stat(dist, p); err == nil && !f.IsDir() {
				if strings.HasPrefix(p, "assets/") || strings.HasPrefix(p, "fonts/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "frontend not built: run `make build`", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	})
}
