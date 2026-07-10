package hylscraper

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/http/httputil"
	"github.com/go-chi/chi/v5"
)

const SCRAPER_UPPER_LIMIT = 5000

type handler struct {
	service Service
	isDev   bool
}

func NewHandler(s Service, isDev bool) *handler {
	return &handler{
		service: s,
		isDev:   isDev,
	}
}

func (h *handler) Init(w http.ResponseWriter, r *http.Request) {
	email, ok := httputil.GetEmailFromAuthHeader(r)
	if !ok {
		slog.Error("user not authorized", "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if email == "" {
		slog.Error("no email found on auth header")
		http.Error(w, "no email found on auth header", http.StatusBadRequest)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		slog.Error("limit param is null")
		http.Error(w, "limit param is null", http.StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("invalid limit param", "err", err.Error())
		return
	}
	if limit > SCRAPER_UPPER_LIMIT || limit <= 0 {
		slog.Error("invalid limit param", "err", fmt.Sprintf("limit must be between %d and %d", 0, SCRAPER_UPPER_LIMIT))
		http.Error(w, fmt.Sprintf("1 <= limit <= %d", SCRAPER_UPPER_LIMIT), http.StatusBadRequest)
		return
	}

	session, err := h.service.Init(r.Context(), email, limit)
	if err != nil {
		slog.Error("error occurred while creating hoyolab scrape session", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("session info", "data", session)

	httputil.Write(w, http.StatusOK, session)
}

// @Summary  Scrape HoyoLab data
// @Tags     hylscraper
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Failure  500 {object} map[string]string
// @Router   /api/v3/hylscraper [get]
func (h *handler) Scrape(w http.ResponseWriter, r *http.Request) {
	// id, ok := httputil.ParseID(w, r, "id")
	//
	//	if !ok {
	//		http.Error(w, "invalid id", http.StatusBadRequest)
	//		return
	//	}
	//
	// limitStr := r.URL.Query().Get("limit")
	//
	//	if limitStr == "" {
	//		http.Error(w, "limit param is null", http.StatusBadRequest)
	//		return
	//	}
	//
	// limit, err := strconv.Atoi(limitStr)
	//
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//
	// upperLim := 5000
	//
	//	if limit > upperLim || limit <= 0 {
	//		http.Error(w, fmt.Sprintf("1 <= limit <= %d", upperLim), http.StatusBadRequest)
	//		return
	//	}
	//
	// results, err := h.service.Scrape(id, limit)
	//
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusInternalServerError)
	//		return
	//	}
	//
	// w.Header().Set("Content-Type", "text/event-stream")
	// w.Header().Set("Cache-Control", "no-cache")
	// w.Header().Set("Connection", "keep-alive")
	// w.Header().Set("X-Accel-Buffering", "no")
	//
	// flusher, ok := w.(http.Flusher)
	//
	//	if !ok {
	//		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
	//		return
	//	}
	//
	// w.WriteHeader(http.StatusOK)
	// fmt.Fprintf(w, "data: {\"message\":\"Scrape job started in the background\"}\n\n")
	// flusher.Flush()
	//
	//	for link := range results {
	//		select {
	//		case <-r.Context().Done():
	//			// Client disconnected, but scrape continues in background
	//			return
	//		default:
	//			jsonData, err := json.Marshal(link)
	//			if err != nil {
	//				slog.Error("error while parsing ScrapeResult", "err", err)
	//				continue
	//			}
	//			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
	//			flusher.Flush()
	//		}
	//	}
	//
	// fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	// flusher.Flush()
}

func (h *handler) StreamUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	email, ok := httputil.GetEmailFromAuthHeader(r)
	if !ok {
		http.Error(w, "You are not authorized to use this endpoint", http.StatusUnauthorized)
		return
	}

	sessionIdStr := chi.URLParam(r, "id")

	if sessionIdStr == "" {
		http.Error(w, "ID param is missing", http.StatusBadRequest)
		return
	}
	sessionId, err := strconv.ParseInt(sessionIdStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("streaming scrape session posts and comments", "id", sessionId, "email", email)
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute*120)
	defer cancel()
	err = h.service.Subscribe(ctx, sessionId, func(payload []byte) {
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	})
	if err != nil && ctx.Err() == nil {
		slog.Error("subscribe error", "err", err)
	}
	if ctx.Err() != nil {
		slog.Error("context error (timeout or cancelled by user)", "err", err)
	}
	flusher.Flush()
}
