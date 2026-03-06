package hylscraper

import (
	"context"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
)

type HylService interface {
	Scrape(ctx context.Context) error
}

type svc struct {
	repo repo.Querier
	db   repo.DBTX
}

func NewService(repo repo.Querier, db repo.DBTX) HylService {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) Scrape(ctx context.Context) error {
	// TODO: implement scraper
	return nil
}
