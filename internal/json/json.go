package jsonutil

import (
	"encoding/json"
	"net/http"
)

func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func Read(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
