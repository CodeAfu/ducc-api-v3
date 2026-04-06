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
	slog.Info("reddit scraper event triggered", "path", r.URL.Path)
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute*8)
	defer cancel()

	redditPosts, err := h.service.GetLinks(ctx, "Genshin_Impact", limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(time.Second)
	start := time.Now()
	for range 1 {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			jsonData, err := json.Marshal(redditPosts)
			if err != nil {
				http.Error(w, fmt.Sprintf("error while marshaling data to json", err.Error()), http.StatusInternalServerError)
				return
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", jsonData)
			if err != nil {
				http.Error(w, fmt.Sprintf("error occurred while attemping to stream: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			flusher.Flush()
			slog.Info("scraper status",
				"time_elapsed", fmt.Sprintf("%.4fs", time.Since(start).Seconds()),
			)
		}
	}
}
