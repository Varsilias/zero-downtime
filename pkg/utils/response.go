package utils

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with status and sensible headers.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
