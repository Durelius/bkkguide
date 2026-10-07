// Package web embeds the built frontend and the shared site config.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

//go:embed site.json
var siteJSON []byte

// Site is the shared site config. site.json is the single place to change the
// site name; the frontend imports the same file.
type Site struct {
	Name    string `json:"name"`
	Tagline string `json:"tagline"`
	BaseURL string `json:"baseUrl"`
}

// LoadSite parses the embedded site.json.
func LoadSite() (Site, error) {
	var s Site
	err := json.Unmarshal(siteJSON, &s)
	return s, err
}

// Dist returns the built frontend (web/dist).
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
