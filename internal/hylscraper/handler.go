package hylscraper

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
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
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		http.Error(w, "limit param is null", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if limit > 2000 || limit <= 0 {
		http.Error(w, "1 <= limit <= 2000", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute*60)
	defer cancel()

	results, err := h.service.Scrape(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for link := range results {
		select {
		case <-r.Context().Done():
			return
		default:
			fmt.Fprintf(w, "data: %s\n\n", link)
			flusher.Flush()
		}
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
