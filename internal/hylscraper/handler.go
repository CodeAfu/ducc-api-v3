package hylscraper

import (
	"net/http"
)

type handler struct {
	service HylService
}

func NewHandler(s HylService) *handler {
	return &handler{
		service: s,
	}
}

// @Summary  Scrape HoyoLab data
// @Tags     hylscraper
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Failure  500 {object} map[string]string
// @Router   /api/v3/hylscraper [get]
func (h *handler) Scrape(w http.ResponseWriter, r *http.Request) {
	h.service.Scrape(r.Context())
	w.Write([]byte("Hello World!"))
}
