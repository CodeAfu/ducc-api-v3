package bingo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5"
)

type BingoService interface {
	GetBingo(ctx context.Context) ([]repo.Bingo, error)
	GetBingoById(ctx context.Context, id int64) (repo.Bingo, error)
	DeleteBingo(ctx context.Context, id int64) error
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

func (s *svc) GetBingoById(ctx context.Context, id int64) (repo.Bingo, error) {
	return s.repo.GetBingoById(ctx, id)
}

func (s *svc) DeleteBingo(ctx context.Context, id int64) error {
	err := s.repo.DeleteBingo(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBingoNotFound
	}
	return err
}

func (s *svc) CreateBingo(ctx context.Context, arg repo.CreateBingoParams) (repo.Bingo, error) {
	if err := validateCells(arg.Cells); err != nil {
		return repo.Bingo{}, err
	}
	return s.repo.CreateBingo(ctx, arg)
}

var validCellKeys = func() map[string]struct{} {
	m := make(map[string]struct{}, 25)
	for i := 1; i <= 25; i++ {
		m[fmt.Sprintf("cell%d", i)] = struct{}{}
	}
	return m
}()

func validateCells(cells []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(cells, &data); err != nil {
		return ErrCellsNotJson
	}
	if len(data) != 25 {
		return ErrCellLenMismatch
	}
	for k, v := range data {
		if _, ok := validCellKeys[k]; !ok {
			return ErrInvalidCellKey
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return ErrValueNotString
			}
		}
	}
	return nil
}
