package image

import (
	"log"
	"net/http"
	"strconv"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	jsonutil "github.com/CodeAfu/go-ducc-api/internal/json"
)

type handler struct {
	service ImageService
}

func NewHandler(s ImageService) *handler {
	return &handler{
		service: s,
	}
}

func (h *handler) GetImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	image, err := h.service.GetImage(r.Context(), id)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(image)
}

func (h *handler) CreateImage(w http.ResponseWriter, r *http.Request) {
	var req repo.CreateImageParams
	if err := jsonutil.Read(r, &req); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdImage, err := h.service.CreateImage(r.Context(), req)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonutil.Write(w, http.StatusCreated, createdImage)
}

func (h *handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteImage(r.Context(), id); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
