// Package handlers implements the HTTP handlers for the backtester API.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// Handlers bundles the dependencies the HTTP handlers need.
type Handlers struct {
	ClaudeAPIKey string
}

// New builds a Handlers value with the given configuration.
func New(claudeAPIKey string) *Handlers {
	return &Handlers{ClaudeAPIKey: claudeAPIKey}
}

// writeJSON serialises v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("failed to encode response:", err)
	}
}

// writeError sends a JSON error body with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
