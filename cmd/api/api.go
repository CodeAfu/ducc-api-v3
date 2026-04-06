package main

import (
	"log/slog"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/bingo"
	"github.com/CodeAfu/go-ducc-api/internal/genshin"
	"github.com/CodeAfu/go-ducc-api/internal/hylscraper"
	"github.com/CodeAfu/go-ducc-api/internal/image"
	"github.com/CodeAfu/go-ducc-api/internal/redditscraper"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
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
	r.Use(slogLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(clerkhttp.WithHeaderAuthorization())
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		limiter := httprate.NewRateLimiter(60, 1*time.Minute)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Token") == app.config.internalToken {
				next.ServeHTTP(w, r)
				return
			}
			limiter.Handler(next).ServeHTTP(w, r)
		})
	})

	// Swagger
	if app.config.env == "development" {
		r.Get("/swagger/*", httpSwagger.WrapHandler)
		r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
		})
	}

	// Health Check
	r.Get("/api/v3/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("service is running"))
	})

	// Bingo
	bingoService := bingo.NewService(repo.New(app.db), app.db)
	bingoHandler := bingo.NewHandler(bingoService)
	r.Get("/api/v3/bingo", bingoHandler.GetBingo)
	r.Get("/api/v3/bingo/{id}", bingoHandler.GetBingoById)
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.RequireHeaderAuthorization())
		r.Post("/api/v3/bingo", bingoHandler.CreateBingo)
	})

	// Image
	imageService := image.NewService(repo.New(app.db), app.db)
	imageHandler := image.NewHandler(imageService)
	r.Get("/api/v3/images", imageHandler.GetImages)
	r.Get("/api/v3/images/{id}", imageHandler.GetImageById)
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.RequireHeaderAuthorization())
		r.Post("/api/v3/images", imageHandler.CreateImage)
		r.Delete("/api/v3/images/{id}", imageHandler.DeleteImage)
	})

	// HoyoLab Scraper
	hylscraperService := hylscraper.NewService(repo.New(app.db), app.db)
	hylscraperHandler := hylscraper.NewHandler(hylscraperService)
	r.Get("/api/v3/hylscraper", hylscraperHandler.Scrape)

	// Reddit Scraper
	redditscraperService := redditscraper.NewService(repo.New(app.db), app.db)
	redditscraperHandler := redditscraper.NewHandler(redditscraperService)
	r.Get("/api/v3/redditscraper/scrape", redditscraperHandler.Scrape)

	// Genshin Impact
	genshinService := genshin.NewService(repo.New(app.db), app.db)
	genshinHandler := genshin.NewHandler(genshinService)
	r.Get("/api/v3/genshin/characters", genshinHandler.GetAllChars)
	r.Get("/api/v3/genshin/characters/{id}", genshinHandler.GetGenshinChar)
	r.Get("/api/v3/genshin/elements", genshinHandler.GetAllElements)
	r.Get("/api/v3/genshin/elements/{element}/icon", genshinHandler.GetElementIconByName)
	r.Get("/api/v3/genshin/elements/id", genshinHandler.GetElementId)
	r.Group(func(r chi.Router) {
		r.Use(clerkhttp.RequireHeaderAuthorization())
		r.Get("/api/v3/genshin/profiles", genshinHandler.GetProfiles)
		r.Get("/api/v3/genshin/profiles/{id}", genshinHandler.GetProfile)
		r.Post("/api/v3/genshin/profiles", genshinHandler.CreateGenshinProfile)
		r.Put("/api/v3/genshin/profiles/{id}", genshinHandler.EditGenshinProfile)
		r.Delete("/api/v3/genshin/profiles/{id}", genshinHandler.DeleteGenshinProfile)

		r.Post("/api/v3/genshin/characters", genshinHandler.AddGenshinChar)
		r.Put("/api/v3/genshin/characters/{id}", genshinHandler.EditGenshinChar)
		r.Delete("/api/v3/genshin/characters/{id}", genshinHandler.DeleteGenshinChar)

		r.Get("/api/v3/genshin/profiles/{id}/characters", genshinHandler.GetAllCharsFromProfile)
		r.Post("/api/v3/genshin/profiles/{prof_id}/{char_name}", genshinHandler.AddCharToProfile)
		r.Put("/api/v3/genshin/profiles/{prof_id}/{char_id}", genshinHandler.EditCharFromProfile)
		r.Delete("/api/v3/genshin/profiles/{prof_id}/{char_id}", genshinHandler.DeleteCharFromProfile)
		r.Get("/api/v3/genshin/profiles/{id}/stats", genshinHandler.GetProfileStats)
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
	slog.Info("server started", "addr", app.config.addr)
	return srv.ListenAndServe()
}

func slogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
			"ip", r.RemoteAddr,
		)
	})
}

type application struct {
	config config
	db     *pgxpool.Pool
}

type dbConfig struct {
	dsn string
}

type config struct {
	env           string
	addr          string
	db            dbConfig
	clerk         clerkConfig
	corsOrigins   []string
	internalToken string
}

type clerkConfig struct {
	key string
}
