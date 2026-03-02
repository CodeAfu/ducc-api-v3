package main

import (
	"log"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/bingo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
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
	r.Get("/api/v3/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	// Bingo
	bingoService := bingo.NewService(repo.New(app.db))
	bingoHandler := bingo.NewHandler(bingoService)
	r.Get("/api/v3/bingo", bingoHandler.GetBingo)
	r.Post("/api/v3/bingo", bingoHandler.CreateBingo)

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
	db     *pgx.Conn
	// logger
}

type dbConfig struct {
	dsn string
}

type config struct {
	addr string
	db   dbConfig
}
