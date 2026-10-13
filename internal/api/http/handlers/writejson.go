package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON answers with the status and the body encoded as JSON, logging an encoding failure.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error(responseBodyFailureText, "error", err)
	}
}
