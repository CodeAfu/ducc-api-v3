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

func (h *handler) Scrape(w http.ResponseWriter, r *http.Request) {
	h.service.Scrape(r.Context())
	w.Write([]byte("Hello World!"))
}
