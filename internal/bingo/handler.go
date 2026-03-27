package bingo

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/http/httputil"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5"
)

type handler struct {
	service BingoService
}

func NewHandler(s BingoService) *handler {
	return &handler{
		service: s,
	}
}

// @Summary  Get all bingo cards
// @Tags     bingo
// @Produce  json
// @Success  200 {array}  bingo.BingoResponse
// @Failure  500 {object} map[string]string
// @Router   /api/v3/bingo [get]
func (h *handler) GetBingo(w http.ResponseWriter, r *http.Request) {
	bingo, err := h.service.GetBingo(r.Context())
	if err != nil {
		slog.Error("failed to read request body", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, bingo)
}

// @Summary  Get bingo card by ID
// @Tags     bingo
// @Produce  json
// @Param    id  path     int  true  "Bingo ID"
// @Success  200 {object} bingo.BingoResponse
// @Failure  404 {object} map[string]string
// @Router   /api/v3/bingo/{id} [get]
func (h *handler) GetBingoById(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	bingo, err := h.service.GetBingoById(r.Context(), id)
	if err != nil {
		slog.Error("failed to get bingo card by id", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, bingo)
}

func (h *handler) DeleteBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteBingo(r.Context(), id); err != nil {
		if errors.Is(err, ErrBingoNotFound) {
			slog.Error("bingo card not found", "err", err, "id", id)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		slog.Error("error occurred while deleting bingo card", "err", err, "id", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary  Create a bingo card
// @Tags     bingo
// @Accept   json
// @Produce  json
// @Param    body body     bingo.CreateBingoRequest true "Bingo card"
// @Success  201  {object} bingo.BingoResponse
// @Failure  400  {object} map[string]string
// @Security BearerAuth
// @Router   /api/v3/bingo [post]
func (h *handler) CreateBingo(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateBingoParams
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	createdBingo, err := h.service.CreateBingo(r.Context(), req)
	if err != nil {
		slog.Error("failed to create bingo", "err", err)
		switch {
		case errors.Is(err, ErrCellsNotJson),
			errors.Is(err, ErrCellLenMismatch),
			errors.Is(err, ErrValueNotString),
			errors.Is(err, ErrInvalidCellKey):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	httputil.Write(w, http.StatusCreated, createdBingo)
}
