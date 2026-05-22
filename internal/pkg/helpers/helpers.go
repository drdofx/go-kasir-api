package helpers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func ParseID(idStr string) (int, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

type errorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorResponse{Message: message, Code: status})
}
