package genshin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/env"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestGetAllCharsIntegration(t *testing.T) {
	_, h := getHandlerAndService(t)
	checkArrayResponseShape(t, h.GetAllChars, func(t *testing.T, i int, c charResponse) {
		if c.ID == 0 {
			t.Errorf("body[%d]: missing ID", i)
		}
		if c.Name == "" {
			t.Errorf("body[%d]: missing Name", i)
		}
		if c.Stars == 0 {
			t.Errorf("body[%d]: missing Stars", i)
		}
		if c.ElementName == "" {
			t.Errorf("body[%d]: missing ElementName", i)
		}
	})
}

func TestAddCharIntegration(t *testing.T) {
	db := setupDB(t)

	tx, err := db.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	t.Cleanup(func() { tx.Rollback(context.Background()) })

	svc := NewService(repo.New(tx), tx)
	h := NewHandler(svc)

	payload := `{"name":"Test Char","element_name":"pyro","stars":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/genshin/characters", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.AddChar(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
	}

	var char repo.CreateGenshinCharRow
	json.NewDecoder(res.Body).Decode(&char)

	if char.ID == 0 {
		t.Error("expected created char to have ID")
	}
	if char.Name != "Test Char" {
		t.Errorf("unexpected name: %s", char.Name)
	}
}

func TestEditCharIntegration(t *testing.T) {
	db := setupDB(t)

	tx, err := db.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	t.Cleanup(func() { tx.Rollback(context.Background()) })

	svc := NewService(repo.New(tx), tx)
	h := NewHandler(svc)

	targetId := 1
	payload := `{"name":"Changed Character Name","element_name":"cryo","stars":4}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v3/genshin/characters/%d", targetId), strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fmt.Sprintf("%d", targetId))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.EditChar(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
	}

	var char map[string]any
	json.NewDecoder(res.Body).Decode(&char)

	if char["id"] == 0 {
		t.Error("expected created char to have ID")
	}
	if char["name"] != "Changed Character Name" {
		t.Errorf("unexpected name: %s", char["name"])
	}
	if char["element_name"] != "cryo" {
		t.Errorf("unexpected element type: %s", char["element_name"])
	}
	if char["stars"].(float64) != 4 {
		t.Errorf("unexpected stars: %d", char["stars"])
	}
}

func checkResponseShape[T any](t *testing.T, fn func(http.ResponseWriter, *http.Request), validate func(t *testing.T, obj T)) {
	req := httptest.NewRequest(http.MethodGet, "/api/v3/genshin/characters", nil)
	w := httptest.NewRecorder()

	fn(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body T
	json.NewDecoder(res.Body).Decode(&body)

	validate(t, body)
}

func checkArrayResponseShape[T any](t *testing.T, fn func(http.ResponseWriter, *http.Request), validate func(t *testing.T, i int, item T)) {
	req := httptest.NewRequest(http.MethodGet, "/api/v3/genshin/characters", nil)
	w := httptest.NewRecorder()

	fn(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body []T
	json.NewDecoder(res.Body).Decode(&body)

	if len(body) == 0 {
		t.Error("expected characters, got empty array")
	}

	for i, item := range body {
		validate(t, i, item)
	}
}

func getHandlerAndService(t *testing.T) (Service, *handler) {
	db := setupDB(t)
	svc := NewService(repo.New(db), db)
	h := NewHandler(svc)
	return svc, h
}

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn, err := envutils.GetString("GOOSE_DBSTRING")
	if err != nil {
		t.Fatalf("missing GOOSE_DBSTRING: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}
