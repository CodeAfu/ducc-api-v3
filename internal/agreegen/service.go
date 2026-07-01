package agreegen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/cache"
	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	CreateDocument(context.Context, *documentRequest) ([]byte, error)
	PreviewDocument(context.Context, *documentRequest) (string, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
	s3   *storage.S3Client
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, s3 *storage.S3Client) Service {
	return &svc{
		repo: repo,
		db:   db,
		s3:   s3,
	}
}

func (s *svc) PreviewDocument(ctx context.Context, req *documentRequest) (string, error) {
	agreementDates, err := getAgreementDates(req.AgreementDuration)
	if err != nil {
		return "", err
	}
	startDateDhivehi, err := dateToDhivehi(agreementDates.Start)
	if err != nil {
		return "", err
	}
	endDateDhivehi, err := dateToDhivehi(agreementDates.End)
	if err != nil {
		return "", err
	}
	cKey := fmt.Sprintf("agreement-generator:preview:%s:%s:%s:%s:%s:%s",
		req.TenantInfo, req.RentAmountStr, req.RentAmountNumStr,
		req.FloorNum, startDateDhivehi, endDateDhivehi,
	)

	c := cache.GetInstance()
	if cached, ok := c.Get(cKey); ok {
		if pdf, ok := cached.(string); ok {
			return pdf, nil
		}
	}

	docBytes, err := createDocument(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "agreegen-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	docxPath := filepath.Join(tmpDir, "agreement.docx")
	if err := os.WriteFile(docxPath, docBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write docx: %w", err)
	}

	cmd := exec.CommandContext(ctx,
		"libreoffice", "--headless", "--norestore",
		"--convert-to", "pdf",
		"--outdir", tmpDir,
		docxPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("libreoffice conversion failed: %s: %w", stderr.String(), err)
	}

	pdfBytes, err := os.ReadFile(filepath.Join(tmpDir, "agreement.pdf"))
	if err != nil {
		return "", fmt.Errorf("failed to read converted pdf: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(pdfBytes)
	c.Set(cKey, b64, time.Minute*5)

	return b64, nil
}

func (s *svc) CreateDocument(ctx context.Context, req *documentRequest) ([]byte, error) {
	return createDocument(ctx, req)
}

func createDocument(ctx context.Context, req *documentRequest) ([]byte, error) {
	agreementDates, err := getAgreementDates(req.AgreementDuration)
	if err != nil {
		return nil, err
	}
	startDateDhivehi, err := dateToDhivehi(agreementDates.Start)
	if err != nil {
		return nil, err
	}
	endDateDhivehi, err := dateToDhivehi(agreementDates.End)
	if err != nil {
		return nil, err
	}

	replaceMap := map[string]string{
		"ޓެނަންޓްމައުލޫމާތު":     req.TenantInfo,
		"ޓެނަންޓްފޯނުނަންބަރު":   req.TenantPhoneNumber,
		"ރެންޓްސްޓްރިންގް":       req.RentAmountStr,
		"ރެންޓްނަންބަރު":         req.RentAmountNumStr,
		"ފްލޯނަންބަރު":           req.FloorNum,
		"އެގްރީމަންޓްފެށޭތާރީހް": startDateDhivehi,
		"އެގްރީމަންޓްނިމޭތާރީހް": endDateDhivehi,
	}

	cKey := fmt.Sprintf("agreement-generator:create:%s:%s:%s:%s:%s:%s:%s",
		req.TenantInfo,
		req.TenantPhoneNumber,
		req.RentAmountStr,
		req.RentAmountNumStr,
		req.FloorNum,
		agreementDates.Start.Format(time.DateOnly),
		agreementDates.End.Format(time.DateOnly),
	)
	c := cache.GetInstance()
	if cached, ok := c.Get(cKey); ok {
		if docBytes, ok := cached.([]byte); ok {
			return docBytes, nil
		}
	}

	jsonBytes, err := json.Marshal(replaceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal replacements: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "agreegen-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "scripts/.venv/bin/python3",
		"scripts/render-agreement.py",
		"--template", "templates/agreement_template.docx",
		"--output-dir", tmpDir,
	)
	cmd.Stdin = bytes.NewReader(jsonBytes)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("render script failed: %s: %w", stderr.String(), err)
	}

	docBytes, err := os.ReadFile(filepath.Join(tmpDir, "output.docx"))
	if err != nil {
		return nil, fmt.Errorf("failed to read rendered document: %w", err)
	}

	c.Set(cKey, docBytes, time.Minute*5)

	return docBytes, nil
}

func getAgreementDates(duration int) (*agreementStartEndDates, error) {
	loc, err := time.LoadLocation("Indian/Maldives")
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	startDate := time.Now().In(loc)
	endDate := calcAgreementEnd(startDate, duration)

	dates := agreementStartEndDates{
		Start: startDate,
		End:   endDate,
	}

	return &dates, nil
}

func dateToDhivehi(date time.Time) (string, error) {
	monthDhivehi, err := monthToDhivehi(date.Month())
	if err != nil {
		return "", err
	}

	dateStr := fmt.Sprintf("%s %s %s", strconv.Itoa(date.Day()), monthDhivehi, strconv.Itoa(date.Year()))

	return dateStr, nil
}

func monthToDhivehi(month time.Month) (string, error) {
	monthMap := map[time.Month]string{
		time.January:   "",
		time.February:  "",
		time.March:     "",
		time.April:     "",
		time.May:       "",
		time.June:      "",
		time.July:      "",
		time.August:    "",
		time.September: "",
		time.October:   "",
		time.November:  "",
		time.December:  "",
	}

	monthDhivehi, ok := monthMap[month]
	if !ok {
		return "", fmt.Errorf("invalid month input: %s", month)
	}

	return monthDhivehi, nil
}

func calcAgreementEnd(date time.Time, numYears int) time.Time {
	anniversary := date.AddDate(numYears, 0, 0)
	firstOfMonth := time.Date(anniversary.Year(), anniversary.Month(), 1, 0, 0, 0, 0, anniversary.Location())
	return firstOfMonth.AddDate(0, 0, -1)
}
