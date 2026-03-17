package genshin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	jsonutil "github.com/CodeAfu/go-ducc-api/internal/json"
	"github.com/go-chi/chi/v5"
)

type handler struct {
	service Service
}

func NewHandler(s Service) *handler {
	return &handler{
		service: s,
	}
}

func (h *handler) GetAllChars(w http.ResponseWriter, r *http.Request) {
	chars, err := h.service.GetAllGenshinChars(r.Context())
	if err != nil {
		slog.Error("error while fetching genshin characters", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonutil.Write(w, http.StatusOK, chars)
}

func (h *handler) AddChar(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateGenshinCharParams
	if err := jsonutil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	char, err := h.service.CreateGenshinChar(r.Context(), req)
	if err != nil {
		slog.Error("failed to create genshin character", "err", err)
		switch {
		case errors.Is(err, ErrCharAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrInvalidElement), errors.Is(err, ErrInvalidStars), errors.Is(err, ErrInvalidElement):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	jsonutil.Write(w, http.StatusOK, char)
}

func (h *handler) EditChar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("id is not a valid integer", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req repo.EditGenshinCharParams
	if err := jsonutil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ID = id
	char, err := h.service.EditGenshinChar(r.Context(), req)
	if err != nil {
		slog.Error("failed to edit genshin character", "err", err)
		switch {
		case errors.Is(err, ErrCharAlreadyExists),
			errors.Is(err, ErrInvalidElement),
			errors.Is(err, ErrInvalidStars):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrCharDoesNotExist):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	jsonutil.Write(w, http.StatusOK, char)
}

func (h *handler) DeleteChar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("id is not a valid integer", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if err := h.service.DeleteGenshinChar(r.Context(), id); err != nil {
		slog.Error("failed edit delete genshin character", "err", err, "id", id)
		switch {
		case errors.Is(err, ErrCharDoesNotExist):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetAllCharsFromProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("id is not a valid integer", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chars, err := h.service.GetAllCharsFromProfile(r.Context(), id)
	if err != nil {
		slog.Error("failed to fetch characters from profile", "err", err, "account_id", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if chars == nil {
		chars = []repo.GetAllCharsFromProfileRow{}
	}
	jsonutil.Write(w, http.StatusOK, chars)
}

func (h *handler) GetAllElements(w http.ResponseWriter, r *http.Request) {
	elements, err := h.service.GetAllElements(r.Context())
	if err != nil {
		slog.Error("error while fetching elements", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if elements == nil {
		elements = []repo.Element{}
	}
	jsonutil.Write(w, http.StatusOK, elements)
}

func (h *handler) GetElementId(w http.ResponseWriter, r *http.Request) {
	elementName := r.URL.Query().Get("element")
	if elementName == "" {
		http.Error(w, "missing 'element' param", http.StatusBadRequest)
		return
	}
	id, err := h.service.GetElementId(r.Context(), elementName)
	if err != nil {
		slog.Error("error while fetching elements", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonutil.Write(w, http.StatusOK, id)
}
