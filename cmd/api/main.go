package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/env"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	dsn, err := env.GetString("GOOSE_DBSTRING")
	if err != nil {
		slog.Error("failed to get GOOSE_DBSTRING", "err", err)
		os.Exit(1)
	}

	clerkKey, err := env.GetString("CLERK_SECRET_KEY")
	if err != nil {
		slog.Error("failed to get CLERK_SECRET_KEY", "err", err)
		os.Exit(1)
	}
	clerk.SetKey(clerkKey)

	corsOrigins, err := env.GetString("CORS_ORIGINS")
	if err != nil {
		slog.Error("failed to get CORS_ORIGINS", "err", err)
		os.Exit(1)
	}

	cfg := config{
		addr: ":8088",
		db: dbConfig{
			dsn: dsn,
		},
		clerk: clerkConfig{
			key: clerkKey,
		},
		corsOrigins: strings.Split(corsOrigins, ","),
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		slog.Error("failed to parse db config", "err", err)
		os.Exit(1)
	}
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	conn, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer conn.Close() // note: no ctx argument for pool

	logger.Info("connected to database")

	app := application{
		config: cfg,
		db:     conn,
	}

	h := app.mount()

	if err := app.run(h); err != nil {
		slog.Error("server has failed to start", "err", err)
		os.Exit(1)
	}
}
