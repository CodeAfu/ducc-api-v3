package httputil

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func ParseID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	str := chi.URLParam(r, key)
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		slog.Error("invalid id param", "key", key, "err", err)
		http.Error(w, key+" is not a valid integer", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
