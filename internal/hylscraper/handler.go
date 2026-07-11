package hylscraper

import (
	"context"
	"errors"
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

func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
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
	session, err := h.service.Create(r.Context(), email)
	if err != nil {
		slog.Error("error occurred while creating hoyolab scrape session", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("session info", "data", session)

	httputil.Write(w, http.StatusCreated, session)
}

func (h *handler) Start(w http.ResponseWriter, r *http.Request) {
	email, ok := httputil.GetEmailFromAuthHeader(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, ok := httputil.ParseID(w, r, "id")
	if !ok {
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
	sortBy := r.URL.Query().Get("sort-by")
	session, err := h.service.Start(r.Context(), email, sessionID, limit, sortBy)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			http.Error(w, "scrape session not found", http.StatusNotFound)
		case errors.Is(err, ErrSessionForbidden):
			http.Error(w, "Forbidden", http.StatusForbidden)
		case errors.Is(err, ErrSessionStarted):
			http.Error(w, "scrape session has already started", http.StatusConflict)
		default:
			slog.Error("error occurred while starting hoyolab scrape session", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	httputil.Write(w, http.StatusAccepted, session)
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
	err = h.service.Subscribe(
		ctx,
		email,
		sessionId,
		func() {
			fmt.Fprintf(w, "event: ready\ndata: {\"session_id\":%d}\n\n", sessionId)
			flusher.Flush()
		},
		func(payload []byte) {
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		},
	)
	if err != nil && ctx.Err() == nil {
		slog.Error("subscribe error", "err", err)
	}
	if ctx.Err() != nil {
		slog.Error("context error (timeout or cancelled by user)", "err", err)
	}
	flusher.Flush()
}
