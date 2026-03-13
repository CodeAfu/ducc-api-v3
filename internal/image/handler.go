package image

import (
	"errors"
	"log/slog"
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

// @Summary  Get all images
// @Tags     images
// @Produce  json
// @Success  200 {array} image.ImageResponse
// @Failure  500 {object} map[string]string
// @Router   /api/v3/images [get]
func (h *handler) GetImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.service.GetImages(r.Context())
	if err != nil {
		slog.Error("message", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonutil.Write(w, http.StatusOK, images)
}

// @Summary  Upload image
// @Tags     images
// @Accept   json
// @Produce  json
// @Param    body body     image.CreateImageRequest true "Image"
// @Success  201  {object} image.ImageResponse
// @Failure  400  {object} map[string]string
// @Security BearerAuth
// @Router   /api/v3/images [post]
func (h *handler) GetImageById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("message", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	image, err := h.service.GetImageById(r.Context(), id)
	if err != nil {
		slog.Error("message", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mimeType := http.DetectContentType(image.ImgData)
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=21600") // 6 hours
	w.WriteHeader(http.StatusOK)
	w.Write(image.ImgData)
}

// @Summary  Upload image
// @Tags     images
// @Accept   json
// @Produce  json
// @Param    body body     image.CreateImageRequest true "Image"
// @Success  201  {object} image.ImageResponse
// @Failure  400  {object} map[string]string
// @Security BearerAuth
// @Router   /api/v3/images [post]
func (h *handler) CreateImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "no session claims found", http.StatusUnauthorized)
		return
	}

	// TODO: Add filename and fileext
	var req repo.CreateImageParams
	if err := jsonutil.Read(r, &req); err != nil {
		slog.Error("message", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := claims.Subject
	req.AddedBy = pgtype.Text{String: userID, Valid: true}

	createdImage, err := h.service.CreateImage(r.Context(), req)
	if err != nil {
		slog.Error("message", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusCreated, createdImage)
}

// @Summary  Delete image
// @Tags     images
// @Param    id  path     int  true  "Image ID"
// @Success  204
// @Failure  404 {object} map[string]string
// @Security BearerAuth
// @Router   /api/v3/images/{id} [delete]
func (h *handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("message", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteImage(r.Context(), id); err != nil {
		slog.Error("message", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
