package hylscraper

import (
	"context"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HylService interface {
	Scrape(ctx context.Context) error
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool) HylService {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) Scrape(ctx context.Context) error {
	// TODO: implement scraper
	return nil
}
