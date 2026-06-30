package http

import (
	"encoding/json"
	"log"
	"net/http"
)

// HealthHandler returns HTTP handlers for health and readiness probes.
type HealthHandler struct {
	// Ready is an optional readiness check.  When nil, /readyz always
	// returns 200.
	Ready func() error
}

// RegisterHealthRoutes adds /healthz and /readyz to the given mux.
func (h *HealthHandler) RegisterHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /readyz", h.handleReadyz)
}

func (h *HealthHandler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if h.Ready != nil {
		if err := h.Ready(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("health: failed to write response: %v", err)
	}
}
