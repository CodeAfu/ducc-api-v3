package image

import (
	"context"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
)

type ImageService interface {
	GetImage(ctx context.Context, id int64) ([]byte, error)
	CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error)
	DeleteImage(ctx context.Context, id int64) error
}

func NewService(repo repo.Querier) ImageService {
	return &svc{
		repo: repo,
	}
}

type svc struct {
	repo repo.Querier
}

func (s *svc) GetImage(ctx context.Context, id int64) ([]byte, error) {
	return s.repo.GetImage(ctx, id)
}

func (s *svc) CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error) {
	return s.repo.CreateImage(ctx, image)
}

func (s *svc) DeleteImage(ctx context.Context, id int64) error {
	return s.repo.DeleteImage(ctx, id)
}
