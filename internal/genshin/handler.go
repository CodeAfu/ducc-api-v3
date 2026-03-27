package genshin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/http/httputil"
	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
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
	httputil.Write(w, http.StatusOK, chars)
}

func (h *handler) GetGenshinChar(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	char, err := h.service.GetGenshinChar(r.Context(), id)
	if err != nil {
		slog.Error("error while fetching genshin character", "err", err, "id", id)
		if errors.Is(err, ErrCharDoesNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, char)
}

func (h *handler) AddGenshinChar(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateGenshinCharParams
	if err := httputil.Read(r, &req); err != nil {
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
	httputil.Write(w, http.StatusOK, char)
}

func (h *handler) EditGenshinChar(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	var req repo.EditGenshinCharParams
	if err := httputil.Read(r, &req); err != nil {
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
	httputil.Write(w, http.StatusOK, char)
}

func (h *handler) DeleteGenshinChar(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
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
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	prof, err := h.service.GetProfile(r.Context(), id)
	if err != nil {
		slog.Error("error while fetching profile", "err", err, "id", id)
		switch {
		case errors.Is(err, ErrProfileDoesNotExist):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	httputil.Write(w, http.StatusOK, prof)
}

func (h *handler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	profs, err := h.service.GetProfiles(r.Context())
	if err != nil {
		slog.Error("error while fetching profiles", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, profs)
}

func (h *handler) GetAllCharsFromProfile(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	email, err := getKeyFromBearerToken(token, "email")
	if err != nil {
		slog.Error("error while decoding token", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("from token", "email", email)
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	prof, err := h.service.GetAllCharsFromProfile(r.Context(), id)
	if err != nil {
		slog.Error("failed to fetch characters from profile", "err", err, "account_id", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, prof)
}

func (h *handler) CreateGenshinProfile(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateGenshinProfileParams
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	profile, err := h.service.CreateGenshinProfile(r.Context(), req)
	if err != nil {
		slog.Error("error while creating genshin profile", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusCreated, profile)
}

func (h *handler) EditGenshinProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	var req editGenshinProfileRequest
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ID = id
	profile, err := h.service.EditGenshinProfile(r.Context(), req)
	if err != nil {
		slog.Error("error while editing genshin profile", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, profile)
}

func (h *handler) DeleteGenshinProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	err := h.service.DeleteGenshinProfile(r.Context(), id)
	if err != nil {
		slog.Error("failed to delete genshin character", "err", err, "id", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) AddCharToProfile(w http.ResponseWriter, r *http.Request) {
	profIdStr := chi.URLParam(r, "prof_id")
	charName := chi.URLParam(r, "char_name")
	profId, err := strconv.ParseInt(profIdStr, 10, 64)
	if err != nil {
		slog.Error("prof_id is not a valid integer", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req repo.AddCharToProfileParams
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ProfID = profId
	req.CharName = charName
	resp, err := h.service.AddCharToProfile(r.Context(), req)
	if err != nil {
		slog.Error("failed to add character to profile", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusCreated, resp)
}

func (h *handler) EditCharFromProfile(w http.ResponseWriter, r *http.Request) {
	profId, ok := httputil.ParseID(w, r, "prof_id")
	if !ok {
		return
	}
	charId, ok := httputil.ParseID(w, r, "char_id")
	if !ok {
		return
	}
	var req repo.EditCharFromProfileParams
	req.ProfID = profId
	req.CharID = charId
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.service.EditCharFromProfile(r.Context(), req)
	if err != nil {
		slog.Error("error occured while attempting to edit profile character", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, resp)
}

func (h *handler) DeleteCharFromProfile(w http.ResponseWriter, r *http.Request) {
	profId, ok := httputil.ParseID(w, r, "prof_id")
	if !ok {
		return
	}
	charId, ok := httputil.ParseID(w, r, "char_id")
	if !ok {
		return
	}
	var req repo.DeleteCharFromProfileParams
	req.ProfID = profId
	req.CharID = charId
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("failed to read request from body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.service.DeleteCharFromProfile(r.Context(), req)
	if err != nil {
		slog.Error("error occured while attempting to edit profile character", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	httputil.Write(w, http.StatusOK, elements)
}

func (h *handler) GetElementIconByName(w http.ResponseWriter, r *http.Request) {
	elementName := chi.URLParam(r, "element")
	if elementName == "" {
		http.Error(w, "url param 'element' not found", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()
	webpIcon, err := h.service.GetElementIconByName(ctx, elementName)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("request timed out", "element", elementName)
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
			return
		}
		slog.Error("error while fetching elements", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(webpIcon))
	if err != nil {
		slog.Error("failed to write image to response", "err", err)
	}
}

func (h *handler) GetElementId(w http.ResponseWriter, r *http.Request) {
	elementName := r.URL.Query().Get("element")
	if elementName == "" {
		http.Error(w, "search param 'element' is missing", http.StatusBadRequest)
		return
	}
	id, err := h.service.GetElementId(r.Context(), elementName)
	if err != nil {
		slog.Error("error while fetching elements", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.Write(w, http.StatusOK, id)
}

func (h *handler) GetProfileStats(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseID(w, r, "id")
	if !ok {
		return
	}
	stats, err := h.service.GetProfileStats(r.Context(), id)
	if err != nil {
		slog.Error("failed to get profile stats", "err", err, "prof_id", id)
		switch {
		case errors.Is(err, ErrProfileDoesNotExist):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	httputil.Write(w, http.StatusOK, stats)
}
