package bingo

import (
	"context"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
)

type BingoService interface {
	GetBingo(ctx context.Context) ([]repo.Bingo, error)
	CreateBingo(ctx context.Context, arg repo.CreateBingoParams) (repo.Bingo, error)
}

type svc struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) BingoService {
	return &svc{
		repo: repo,
	}
}

func (s *svc) GetBingo(ctx context.Context) ([]repo.Bingo, error) {
	return s.repo.GetBingo(ctx)
}

func (s *svc) CreateBingo(ctx context.Context, arg repo.CreateBingoParams) (repo.Bingo, error) {
	return s.repo.CreateBingo(ctx, arg)
}
