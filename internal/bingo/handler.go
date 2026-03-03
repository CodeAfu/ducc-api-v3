package bingo

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/json"
	"github.com/go-chi/chi/v5"
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

func (h *handler) GetBingo(w http.ResponseWriter, r *http.Request) {
	bingo, err := h.service.GetBingo(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusOK, bingo)
}

func (h *handler) GetBingoById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bingo, err := h.service.GetBingoById(r.Context(), id)
	if err != nil {
		log.Println(err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusOK, bingo)
}

func (h *handler) CreateBingo(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateBingoParams
	if err := jsonutil.Read(r, &req); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdBingo, err := h.service.CreateBingo(r.Context(), req)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusCreated, createdBingo)
}
