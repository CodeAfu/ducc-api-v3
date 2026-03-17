package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/CodeAfu/go-ducc-api/docs"
	"github.com/CodeAfu/go-ducc-api/internal/env"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

// @title           Ducc API
// @version         3.0
// @host            localhost:8088
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	clerk.SetKey(cfg.clerk.key)

	ctx := context.Background()
	conn, err := newDBPool(ctx, cfg.db.dsn)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

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

func newDBPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, poolConfig)
}

func loadConfig() (config, error) {
	internalToken, err := envutils.GetString("INTERNAL_TOKEN")
	if err != nil {
		return config{}, fmt.Errorf("INTERNAL_TOKEN: %w", err)
	}

	envVar, err := envutils.GetString("ENV")
	if err != nil {
		return config{}, fmt.Errorf("ENV: %w", err)
	}

	dsn, err := envutils.GetString("GOOSE_DBSTRING")
	if err != nil {
		return config{}, fmt.Errorf("GOOSE_DBSTRING: %w", err)
	}

	clerkKey, err := envutils.GetString("CLERK_SECRET_KEY")
	if err != nil {
		return config{}, fmt.Errorf("CLERK_SECRET_KEY: %w", err)
	}

	corsOrigins, err := envutils.GetString("CORS_ORIGINS")
	if err != nil {
		return config{}, fmt.Errorf("CORS_ORIGINS: %w", err)
	}

	return config{
		env:           envVar,
		addr:          ":8088",
		internalToken: internalToken,
		db:            dbConfig{dsn: dsn},
		clerk:         clerkConfig{key: clerkKey},
		corsOrigins:   strings.Split(corsOrigins, ","),
	}, nil
}
