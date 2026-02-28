package main

import (
	"log"
	"net/http"
	"time"

	handler "github.com/CodeAfu/go-ducc-api/internals"
	"github.com/CodeAfu/go-ducc-api/internals/bingo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// mount

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Get("/", handler.HealthCheck)

	// Bingo
	bingoService := bingo.NewService()
	bingoHandler := bingo.NewHandler(bingoService)
	r.Get("/api/bingo", bingoHandler.GetBingo)

	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	srv.SetKeepAlivesEnabled(true)

	log.Printf("server started at %s\n", app.config.addr)

	return srv.ListenAndServe()
}

type application struct {
	config config
	// logger
	// db driver
}

type dbConfig struct {
	dsn string
}

type config struct {
	addr string
	db   dbConfig
}
