package httputil

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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

func GetEmailFromAuthHeader(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	prefix := "Bearer "
	ok := strings.HasPrefix(authHeader, prefix)
	if !ok {
		return "", false
	}
	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", false
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", false
	}

	emailObj, ok := claims["email"]
	if !ok {
		return "", false
	}

	email, ok := emailObj.(string)
	if !ok {
		return "", false
	}

	return email, true && email != ""
}
