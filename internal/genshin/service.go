package genshin

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/pgerr"
	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	GetAllGenshinChars(ctx context.Context) ([]repo.GetAllGenshinCharsRow, error)
	GetGenshinChar(ctx context.Context, id int64) (repo.GetGenshinCharRow, error)
	CreateGenshinChar(ctx context.Context, arg repo.CreateGenshinCharParams) (repo.CreateGenshinCharRow, error)
	EditGenshinChar(ctx context.Context, arg repo.EditGenshinCharParams) (repo.EditGenshinCharRow, error)
	DeleteGenshinChar(ctx context.Context, id int64) error

	GetProfile(ctx context.Context, id int64) (repo.GenshinProfile, error)
	GetProfiles(ctx context.Context) ([]repo.GenshinProfile, error)
	GetAllCharsFromProfile(ctx context.Context, accID int64) (*ProfileResponse, error)
	CreateGenshinProfile(ctx context.Context, arg repo.CreateGenshinProfileParams) (repo.GenshinProfile, error)
	EditGenshinProfile(ctx context.Context, arg editGenshinProfileRequest) (repo.GenshinProfile, error)
	DeleteGenshinProfile(ctx context.Context, id int64) error
	AddCharToProfile(ctx context.Context, arg repo.AddCharToProfileParams) (repo.AddCharToProfileRow, error)
	EditCharFromProfile(ctx context.Context, arg repo.EditCharFromProfileParams) (repo.EditCharFromProfileRow, error)
	DeleteCharFromProfile(ctx context.Context, arg repo.DeleteCharFromProfileParams) error

	GetAllElements(ctx context.Context) ([]repo.Element, error)
	GetElementId(ctx context.Context, name string) (int16, error)
	GetElementIconByName(ctx context.Context, name string) (string, error)

	GetProfileStats(ctx context.Context, profID int64) (profileStatsResponse, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) GetAllGenshinChars(ctx context.Context) ([]repo.GetAllGenshinCharsRow, error) {
	return s.repo.GetAllGenshinChars(ctx)
}

func (s *svc) GetGenshinChar(ctx context.Context, id int64) (repo.GetGenshinCharRow, error) {
	char, err := s.repo.GetGenshinChar(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.GetGenshinCharRow{}, ErrCharDoesNotExist
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.NotFound:
				return repo.GetGenshinCharRow{}, ErrCharDoesNotExist
			}
		}
		return repo.GetGenshinCharRow{}, err
	}
	return char, err
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

func (s *svc) GetProfile(ctx context.Context, id int64) (repo.GenshinProfile, error) {
	prof, err := s.repo.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.GenshinProfile{}, ErrProfileDoesNotExist
		}
		return repo.GenshinProfile{}, err
	}
	return prof, nil
}

func (s *svc) GetProfiles(ctx context.Context) ([]repo.GenshinProfile, error) {
	return s.repo.GetProfiles(ctx)
}

func (s *svc) GetAllCharsFromProfile(ctx context.Context, accID int64) (*ProfileResponse, error) {
	rows, err := s.repo.GetAllCharsFromProfile(ctx, accID)
	if err != nil {
		return nil, err
	}

	prof, err := s.repo.GetProfile(ctx, accID)
	if err != nil {
		return nil, err
	}

	res := &ProfileResponse{
		ID:         prof.ID,
		Name:       prof.Name,
		Notes:      prof.Notes.String,
		Characters: make([]CharacterResponse, 0, len(rows)),
	}

	for _, r := range rows {
		char := CharacterResponse{
			CharID:        r.ProfileChar.CharID,
			Name:          r.CharDetail.Name,
			DisplayName:   r.CharDetail.DisplayName.String,
			Level:         r.ProfileChar.Level,
			AscLevel:      r.ProfileChar.AscLevel,
			Stars:         r.CharDetail.Stars,
			Constellation: r.ProfileChar.Constellation,
			TalentNa:      r.ProfileChar.TalentNa,
			TalentE:       r.ProfileChar.TalentE,
			TalentQ:       r.ProfileChar.TalentQ,
			CharNotes:     r.ProfileChar.Notes.String,
			ElementName:   r.Element.Name,
			ElementIcon:   r.Element.IconUrl.String,
		}
		res.Characters = append(res.Characters, char)
	}

	return res, nil
}

func (s *svc) CreateGenshinProfile(ctx context.Context, arg repo.CreateGenshinProfileParams) (repo.GenshinProfile, error) {
	return s.repo.CreateGenshinProfile(ctx, arg)
}

func (s *svc) EditGenshinProfile(ctx context.Context, req editGenshinProfileRequest) (repo.GenshinProfile, error) {
	existing, err := s.repo.GetProfile(ctx, req.ID)

	if err != nil {
		return repo.GenshinProfile{}, err
	}

	params := repo.EditGenshinProfileParams{
		ID:    req.ID,
		Name:  existing.Name,
		Notes: existing.Notes,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Notes != nil {
		params.Notes = pgtype.Text{
			String: *req.Notes,
			Valid:  true,
		}
	}

	return s.repo.EditGenshinProfile(ctx, params)
}

func (s *svc) DeleteGenshinProfile(ctx context.Context, id int64) error {
	return s.repo.DeleteGenshinProfile(ctx, id)
}

func (s *svc) AddCharToProfile(ctx context.Context, arg repo.AddCharToProfileParams) (repo.AddCharToProfileRow, error) {
	return s.repo.AddCharToProfile(ctx, arg)
}

func (s *svc) EditCharFromProfile(ctx context.Context, arg repo.EditCharFromProfileParams) (repo.EditCharFromProfileRow, error) {
	return s.repo.EditCharFromProfile(ctx, arg)
}

func (s *svc) DeleteCharFromProfile(ctx context.Context, arg repo.DeleteCharFromProfileParams) error {
	return s.repo.DeleteCharFromProfile(ctx, arg)
}

func (s *svc) GetAllElements(ctx context.Context) ([]repo.Element, error) {
	return s.repo.GetAllElements(ctx)
}

func (s *svc) GetElementId(ctx context.Context, name string) (int16, error) {
	return s.repo.GetElementId(ctx, name)
}

func (s *svc) GetElementIconByName(ctx context.Context, name string) (string, error) {
	iconUrl, err := s.repo.GetElementIconByName(ctx, name)
	if err != nil {
		return "", err
	}
	if iconUrl.Valid == false {
		return "", ErrIconUrlNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconUrl.String, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrExternalAPINotOK
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	return string(body), nil
}

func (s *svc) GetProfileStats(ctx context.Context, profID int64) (profileStatsResponse, error) {
	charCount, err := s.repo.GetProfileCharStats(ctx, profID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profileStatsResponse{}, ErrProfileDoesNotExist
		}
		return profileStatsResponse{}, err
	}
	elementStats, err := s.repo.GetProfileElementCounts(ctx, profID)
	if err != nil {
		return profileStatsResponse{}, err
	}
	mappedStats := make([]elementStat, len(elementStats))
	for i, stat := range elementStats {
		mappedStats[i] = elementStat{
			ElementName: stat.ElementName,
			CharCount:   stat.CharCount,
		}
	}
	res := profileStatsResponse{
		CharCount:     charCount,
		ElementCounts: mappedStats,
	}
	return res, nil
}
