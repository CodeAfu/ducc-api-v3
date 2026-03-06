package main

import (
	"log/slog"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/bingo"
	"github.com/CodeAfu/go-ducc-api/internal/hylscraper"
	"github.com/CodeAfu/go-ducc-api/internal/image"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mount

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.config.corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httprate.LimitByIP(60, 1*time.Minute))

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

	// HoyoLab Scraper
	hylscraperService := hylscraper.NewService(repo.New(app.db), app.db)
	hylscraperHandler := hylscraper.NewHandler(hylscraperService)
	r.Get("/api/v3/hylscraper", hylscraperHandler.Scrape)

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

	slog.Info("server started", "addr", app.config.addr)

	return srv.ListenAndServe()
}

type application struct {
	config config
	db     *pgxpool.Pool
}

type dbConfig struct {
	dsn string
}

type config struct {
	addr        string
	db          dbConfig
	clerk       clerkConfig
	corsOrigins []string
}

type clerkConfig struct {
	key string
}
