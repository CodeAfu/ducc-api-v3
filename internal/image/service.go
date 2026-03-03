package image

import (
	"context"
	"errors"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
)

type ImageService interface {
	GetImages(ctx context.Context) ([]repo.Image, error)
	GetImageById(ctx context.Context, id int64) ([]byte, error)
	CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error)
	DeleteImage(ctx context.Context, id int64) error
}

type svc struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) ImageService {
	return &svc{
		repo: repo,
	}
}

func (s *svc) GetImages(ctx context.Context) ([]repo.Image, error) {
	return s.repo.GetImages(ctx)
}

func (s *svc) GetImageById(ctx context.Context, id int64) ([]byte, error) {
	return s.repo.GetImageById(ctx, id)
}

func (s *svc) CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error) {
	if !CheckValidImage(image.ImgData) {
		return repo.Image{}, errors.New("invalid image")
	}
	image.ImgHash = GenerateImageHash(image.ImgData)
	return s.repo.CreateImage(ctx, image)
}

func (s *svc) DeleteImage(ctx context.Context, id int64) error {
	return s.repo.DeleteImage(ctx, id)
}
