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

	"bkkguide/internal/admin"
	"bkkguide/internal/api"
	"bkkguide/internal/auth"
	"bkkguide/internal/config"
	"bkkguide/internal/db"
	"bkkguide/internal/media"
	"bkkguide/internal/spa"
	"bkkguide/web"
)

func main() {
	seed := flag.Bool("seed", false, "insert development data into an empty database")
	newAdmin := flag.Bool("create-admin", false, "create an admin interactively, then exit")
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
	if *newAdmin {
		if err := createAdmin(conn); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *seed {
		if err := db.Seed(conn); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	if !media.Available() {
		log.Printf("warning: ffmpeg not found on PATH; photo uploads will fail")
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, middleware.Compress(5))
	authSvc := &auth.Service{DB: conn, Secure: !cfg.Dev, Now: time.Now}
	r.Mount("/api/admin", (&admin.Handler{DB: conn, Auth: authSvc, Limiter: auth.NewLoginLimiter()}).Routes())
	r.Mount("/api", (&api.API{DB: conn, Site: site, Now: time.Now}).Routes())
	r.Get("/media/photos/{photoId}/{size}.jpg", func(w http.ResponseWriter, r *http.Request) {
		media.ServePhoto(w, r, conn, true)
	})
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
