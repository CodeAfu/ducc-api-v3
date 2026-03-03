package main

import (
	"log"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/bingo"
	"github.com/CodeAfu/go-ducc-api/internal/image"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
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
	r.Get("/api/v3/bingo/{id}", bingoHandler.GetBingoById)
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Post("/api/v3/bingo", bingoHandler.CreateBingo)
	})

	// Image
	imageService := image.NewService(repo.New(app.db))
	imageHandler := image.NewHandler(imageService)
	r.Get("/api/v3/images", imageHandler.GetImages)
	r.Get("/api/v3/images/{id}", imageHandler.GetImageById)
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.WithHeaderAuthorization())
		r.Post("/api/v3/images", imageHandler.CreateImage)
		r.Delete("/api/v3/images/{id}", imageHandler.DeleteImage)
	})

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
	db     *pgxpool.Pool
	// logger
}

type dbConfig struct {
	dsn string
}

type config struct {
	addr  string
	db    dbConfig
	clerk clerkConfig
}

type clerkConfig struct {
	key string
}
