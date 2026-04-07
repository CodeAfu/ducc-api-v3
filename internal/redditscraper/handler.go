package redditscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type handler struct {
	service Service
}

func NewHandler(s Service) *handler {
	return &handler{
		service: s,
	}
}

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
		http.Error(w, "limit param is not a valid number", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute*8)
	defer cancel()

	// redditPosts, err := h.service.GetLinks(ctx, "Genshin_Impact", limit)
	results, err := h.service.ScrapeAndStore(ctx, "Genshin_Impact", limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for result := range results {
		if result.Err != nil {
			slog.Warn("scrape failed", "url", result.Post.URL, "err", result.Err)
			continue
		}
		jsonData, _ := json.Marshal(result)
		fmt.Fprintf(w, "data: &s\n\n", jsonData)
		flusher.Flush()
	}
}
