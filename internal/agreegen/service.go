package agreegen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	repo  *repo.Queries
	db    *pgxpool.Pool
	s3    *storage.S3Client
	isDev bool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, s3 *storage.S3Client, isDev bool) Service {
	return &svc{
		repo:  repo,
		db:    db,
		s3:    s3,
		isDev: isDev,
	}
}

func (s *svc) PreviewDocument(ctx context.Context, req *documentRequest) (string, error) {
	agreementDates, err := getAgreementDates(req.AgreementStart, req.AgreementDuration)
	if err != nil {
		return "", err
	}
	replaceMap, err := s.buildReplaceMap(req, agreementDates)
	if err != nil {
		return "", err
	}
	hash, _, err := hashReplaceMap(replaceMap)
	if err != nil {
		return "", err
	}
	previewCKey := "agreement-generator:preview:" + hash

	c := cache.GetInstance()
	if cached, ok := c.Get(previewCKey); ok {
		if pdf, ok := cached.(string); ok {
			return pdf, nil
		}
	}

	docBytes, err := s.createDocument(ctx, req)
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
	c.Set(previewCKey, b64, time.Minute*30)

	return b64, nil
}

func (s *svc) CreateDocument(ctx context.Context, req *documentRequest) ([]byte, error) {
	return s.createDocument(ctx, req)
}

func (s *svc) createDocument(ctx context.Context, req *documentRequest) ([]byte, error) {
	agreementDates, err := getAgreementDates(req.AgreementStart, req.AgreementDuration)
	if err != nil {
		return nil, err
	}
	replaceMap, err := s.buildReplaceMap(req, agreementDates)
	if err != nil {
		return nil, err
	}
	hash, jsonBytes, err := hashReplaceMap(replaceMap)
	if err != nil {
		return nil, err
	}
	cKey := "agreement-generator:create:" + hash
	slog.Debug("cache key (agreement create)", "key", cKey)

	c := cache.GetInstance()
	if cached, ok := c.Get(cKey); ok {
		if docBytes, ok := cached.([]byte); ok {
			return docBytes, nil
		}
	}

	tmpDir, err := os.MkdirTemp("", "agreegen-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pythonBin := "python3"
	if s.isDev {
		pythonBin = "scripts/.venv/bin/python3"
	}
	cmd := exec.CommandContext(ctx, pythonBin,
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

	c.Set(cKey, docBytes, time.Minute*30)

	return docBytes, nil
}

func (s *svc) buildReplaceMap(req *documentRequest, dates *agreementStartEndDates) (map[string]string, error) {
	startDateDhivehi, err := dateToDhivehi(dates.Start)
	if err != nil {
		return nil, err
	}
	endDateDhivehi, err := dateToDhivehi(dates.End)
	if err != nil {
		return nil, err
	}

	replaceMap := map[string]string{
		"ޓެނަންޓްމައުލޫމާތު":     req.TenantInfo,
		"ރެންޓްއެމައުންޓް":       req.RentAmountStr,
		"ވަންޑިޕޮސިޓް":           req.SingleDeposit,
		"ފްލޯނަންބަރު":           req.FloorNum,
		"އެގްރީމަންޓްފެށޭތާރީހް": startDateDhivehi,
		"އެގްރީމަންޓްނިމޭތާރީހް": endDateDhivehi,
		"އެގްރީމަންޓްމުއްދަތު":   strconv.Itoa(req.AgreementDuration),
	}
	if req.SigFieldTenantName != "" {
		replaceMap["ސައިންޓެނަންޓްނަން"] = req.SigFieldTenantName
	}
	if req.SigFieldTenantId != "" {
		replaceMap["ސައިންޓެނަންޓްއައިޑީ"] = req.SigFieldTenantId
	}
	if req.SigFieldTenantAddress != "" {
		replaceMap["ސައިންޓެނަންޓްއެޑްރެސް"] = req.SigFieldTenantAddress
	}
	if req.TenantPhoneNumber != "" {
		replaceMap["ޓެނަންޓްފޯނުނަންބަރު"] = req.TenantPhoneNumber
	}
	return replaceMap, nil
}

func hashReplaceMap(replaceMap map[string]string) (string, []byte, error) {
	jsonBytes, err := json.Marshal(replaceMap)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal replacements: %w", err)
	}
	h := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", h), jsonBytes, nil
}

func getAgreementDates(startDate time.Time, duration int) (*agreementStartEndDates, error) {
	loc, err := time.LoadLocation("Indian/Maldives")
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	if startDate.IsZero() {
		startDate = time.Now().In(loc)
	} else {
		startDate = startDate.In(loc)
	}

	endDate := calcAgreementEnd(startDate, duration)

	return &agreementStartEndDates{Start: startDate, End: endDate}, nil
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
		time.January:   "ޖަނަވަރީ",
		time.February:  "ފެބްރުއަރީ",
		time.March:     "މާޗް",
		time.April:     "އެޕްރިލް",
		time.May:       "މެއި",
		time.June:      "ޖޫން",
		time.July:      "ޖުލައި",
		time.August:    "އޯގަސްޓް",
		time.September: "ސެޕްޓެމްބަރ",
		time.October:   "އޮކްޓޯބަރ",
		time.November:  "ނޮވެމްބަރ",
		time.December:  "ޑިސެމްބަރ",
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

func formatRentAmountNumStr(rentAmountNumStr string) (string, error) {
	n, err := strconv.Atoi(rentAmountNumStr)
	if err != nil {
		return "", fmt.Errorf("rent amount is not a valid integer: %s", err)
	}

	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)

	result := strings.Join(parts, "،")
	if neg {
		result = "-" + result
	}

	return result, nil
}
