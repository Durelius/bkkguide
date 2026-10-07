// Package config reads runtime settings from the environment.
package config

import "os"

type Config struct {
	Addr       string // BKK_ADDR
	DBPath     string // BKK_DB
	UploadsDir string // BKK_UPLOADS
	BaseURL    string // BKK_BASE_URL, overrides site.json baseUrl when set
	Dev        bool   // BKK_DEV=1: plain-HTTP cookies for local development
}

func Load() Config {
	return Config{
		Addr:       env("BKK_ADDR", "127.0.0.1:8080"),
		DBPath:     env("BKK_DB", "data/bkkguide.db"),
		UploadsDir: env("BKK_UPLOADS", "data/uploads"),
		BaseURL:    os.Getenv("BKK_BASE_URL"),
		Dev:        os.Getenv("BKK_DEV") == "1",
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
