package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func parseID(path string, prefix string) (int, error) {
	idStr := strings.TrimPrefix(path, prefix)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, errors.New("id must be positive")
	}
	return id, nil
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}
