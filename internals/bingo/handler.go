package bingo

import (
	"log"
	"net/http"

	"github.com/CodeAfu/go-ducc-api/internals/json"
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
	err := h.service.GetBingo(r.Context())

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cards := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}

	json.Write(w, http.StatusOK, cards)
}
