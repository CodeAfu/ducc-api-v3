package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/CodeAfu/go-ducc-api/internal/env"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	dsn, err := env.GetString("GOOSE_DBSTRING")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err != nil {
		slog.Error("failed to get GOOSE_DBSTRING", "err", err)
		os.Exit(1)
	}

	cfg := config{
		addr: ":8088",
		db: dbConfig{
			dsn: dsn,
		},
	}

	conn, err := pgx.Connect(ctx, cfg.db.dsn)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	logger.Info("connected to database")

	api := application{
		config: cfg,
		db:     conn,
	}

	h := api.mount()

	if err := api.run(h); err != nil {
		slog.Error("server has failed to start", "err", err)
		os.Exit(1)
	}
}
