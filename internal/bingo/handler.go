package bingo

import (
	"log"
	"net/http"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/json"
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
