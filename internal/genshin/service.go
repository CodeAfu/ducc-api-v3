package genshin

import (
	"context"
	"errors"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/pgerr"
	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service interface {
	GetAllGenshinChars(ctx context.Context) ([]repo.GetAllGenshinCharsRow, error)
	CreateGenshinChar(ctx context.Context, arg repo.CreateGenshinCharParams) (repo.CreateGenshinCharRow, error)
	EditGenshinChar(ctx context.Context, arg repo.EditGenshinCharParams) (repo.EditGenshinCharRow, error)
	DeleteGenshinChar(ctx context.Context, id int64) error

	GetAllCharsFromProfile(ctx context.Context, accID int64) ([]repo.GetAllCharsFromProfileRow, error)

	GetAllElements(ctx context.Context) ([]repo.Element, error)
	GetElementId(ctx context.Context, name string) (int16, error)
}

type svc struct {
	repo repo.Querier
	db   repo.DBTX
}

func NewService(repo repo.Querier, db repo.DBTX) Service {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) GetAllGenshinChars(ctx context.Context) ([]repo.GetAllGenshinCharsRow, error) {
	return s.repo.GetAllGenshinChars(ctx)
}

func (s *svc) GetAllElements(ctx context.Context) ([]repo.Element, error) {
	return s.repo.GetAllElements(ctx)
}

func (s *svc) GetElementId(ctx context.Context, name string) (int16, error) {
	return s.repo.GetElementId(ctx, name)
}

func (s *svc) CreateGenshinChar(ctx context.Context, arg repo.CreateGenshinCharParams) (repo.CreateGenshinCharRow, error) {
	char, err := s.repo.CreateGenshinChar(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.UniqueViolation:
				return repo.CreateGenshinCharRow{}, ErrCharAlreadyExists
			case pgerr.ForeignKeyViolation:
				return repo.CreateGenshinCharRow{}, ErrInvalidElement
			case pgerr.CheckViolation:
				return repo.CreateGenshinCharRow{}, ErrInvalidStars
			case pgerr.NotNullViolation:
				return repo.CreateGenshinCharRow{}, ErrInvalidElement
			}
		}
		return repo.CreateGenshinCharRow{}, err
	}
	return char, nil
}

func (s *svc) EditGenshinChar(ctx context.Context, arg repo.EditGenshinCharParams) (repo.EditGenshinCharRow, error) {
	char, err := s.repo.EditGenshinChar(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.EditGenshinCharRow{}, ErrCharDoesNotExist
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.UniqueViolation:
				return repo.EditGenshinCharRow{}, ErrCharAlreadyExists
			case pgerr.NotNullViolation:
				return repo.EditGenshinCharRow{}, ErrInvalidElement
			case pgerr.CheckViolation:
				return repo.EditGenshinCharRow{}, ErrInvalidStars
			}
		}
		return repo.EditGenshinCharRow{}, err
	}
	return char, nil
}

func (s *svc) DeleteGenshinChar(ctx context.Context, id int64) error {
	if err := s.repo.DeleteGenshinChar(ctx, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.NotFound:
				return ErrCharDoesNotExist
			}
		}
		return err
	}
	return nil
}

func (s *svc) GetAllCharsFromProfile(ctx context.Context, accID int64) ([]repo.GetAllCharsFromProfileRow, error) {
	return s.repo.GetAllCharsFromProfile(ctx, accID)
}
