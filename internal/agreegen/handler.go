package agreegen

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/http/httputil"
)

type handler struct {
	service Service
}

func NewHandler(s Service) *handler {
	return &handler{
		service: s,
	}
}

func (h *handler) PreviewDocument(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	var req documentRequest
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("error while reading request body", "err", err)
		http.Error(w, "error while reading request body", http.StatusBadRequest)
		return
	}
	base64Doc, err := h.service.PreviewDocument(ctx, &req)
	if err != nil {
		slog.Error("internal server error", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(base64Doc))
}

func (h *handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	var req documentRequest
	if err := httputil.Read(r, &req); err != nil {
		slog.Error("error while reading request body", "err", err)
		http.Error(w, "error while reading request body", http.StatusBadRequest)
		return
	}
	data, err := h.service.CreateDocument(ctx, &req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("request timed out", "err", err)
			http.Error(w, "request timed out", http.StatusGatewayTimeout)
			return
		}
		slog.Error("internal error occurred", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="lease_agreement.docx"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
