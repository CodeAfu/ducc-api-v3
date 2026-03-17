package image

import (
	"context"
	"errors"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
)

type ImageService interface {
	GetImages(ctx context.Context) ([]repo.Image, error)
	GetImageById(ctx context.Context, id int64) (repo.Image, error)
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

func (s *svc) GetImageById(ctx context.Context, id int64) (repo.Image, error) {
	return s.repo.GetImageById(ctx, id)
}

func (s *svc) CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error) {
	if !CheckValidImage(image.ImgData) {
		return repo.Image{}, errors.New("invalid image")
	}
	image.ImgHash = GenerateImageHash(image.ImgData)
	result, err := s.repo.CreateImage(ctx, image)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repo.Image{}, ErrDuplicateImage
		}
	}
	return result, nil
}

func (s *svc) DeleteImage(ctx context.Context, id int64) error {
	image, err := s.repo.GetImageById(ctx, id)
	if err != nil {
		return err
	}
	if image.IsProtected {
		return ErrProtectedImage
	}
	return s.repo.DeleteImage(ctx, id)
}
