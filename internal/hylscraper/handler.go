package hylscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/http/httputil"
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

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

	upperLim := 5000
	if limit > upperLim || limit <= 0 {
		http.Error(w, fmt.Sprintf("1 <= limit <= %d", upperLim), http.StatusBadRequest)
		return
	}

	bgCtx, _ := context.WithTimeout(context.Background(), time.Minute*40)
	// defer bgCancel()

	results, err := h.service.Scrape(bgCtx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "data: {\"message\":\"Scrape job started in the background\"}\n\n")
	flusher.Flush()

	for link := range results {
		select {
		case <-r.Context().Done():
			return
		default:
			jsonData, err := json.Marshal(link)
			if err != nil {
				slog.Error("error while parsing ScrapeResult", "err", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

func (h *handler) StreamUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	_, ok := httputil.GetEmailFromAuthHeader(r)
	if !ok {
		http.Error(w, "Email not found on auth token", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// svc that interacts with db

	flusher.Flush()
}
