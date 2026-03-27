package image

import (
	"context"
	"errors"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/pgerr"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImageService interface {
	GetImages(ctx context.Context) ([]repo.Image, error)
	GetImageById(ctx context.Context, id int64) (repo.Image, error)
	CreateImage(ctx context.Context, image repo.CreateImageParams) (repo.Image, error)
	DeleteImage(ctx context.Context, id int64) error
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool) ImageService {
	return &svc{
		repo: repo,
		db:   db,
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
		return repo.Image{}, ErrInvalidImage
	}
	image.ImgHash = GenerateImageHash(image.ImgData)
	result, err := s.repo.CreateImage(ctx, image)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.UniqueViolation:
				return repo.Image{}, ErrDuplicateImage
			default:
				return repo.Image{}, err
			}
		}
		return repo.Image{}, err
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
