package image

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	jsonutil "github.com/CodeAfu/go-ducc-api/internal/json"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type handler struct {
	service ImageService
}

func NewHandler(s ImageService) *handler {
	return &handler{
		service: s,
	}
}

func (h *handler) GetImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.service.GetImages(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonutil.Write(w, http.StatusOK, images)
}

func (h *handler) GetImageById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	image, err := h.service.GetImageById(r.Context(), id)
	if err != nil {
		log.Println(err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(image)
}

func (h *handler) CreateImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "no session claims found", http.StatusUnauthorized)
		return
	}

	var req repo.CreateImageParams
	if err := jsonutil.Read(r, &req); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := claims.Subject
	req.AddedBy = pgtype.Text{String: userID, Valid: true}

	createdImage, err := h.service.CreateImage(r.Context(), req)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusCreated, createdImage)
}

func (h *handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteImage(r.Context(), id); err != nil {
		log.Println(err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
