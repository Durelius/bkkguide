package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"bkkguide/internal/api"
	"bkkguide/internal/config"
	"bkkguide/internal/db"
	"bkkguide/internal/spa"
	"bkkguide/web"
)

func main() {
	seed := flag.Bool("seed", false, "insert development data into an empty database")
	flag.Parse()

	cfg := config.Load()
	site, err := web.LoadSite()
	if err != nil {
		log.Fatalf("site.json: %v", err)
	}
	if cfg.BaseURL != "" {
		site.BaseURL = cfg.BaseURL
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	if *seed {
		if err := db.Seed(conn); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, middleware.Compress(5))
	r.Mount("/api", (&api.API{DB: conn, Site: site, Now: time.Now}).Routes())
	r.Handle("/media/*", http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.UploadsDir))))
	r.Handle("/*", spa.Handler(web.Dist()))

	srv := &http.Server{Addr: cfg.Addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("%s listening on http://%s", site.Name, cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
