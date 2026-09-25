package telemetry

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ store *Store }

func NewHandler(s *Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles/{vin}/telemetry", h.get)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.store.Get(r.PathValue("vin"))
	if !ok {
		httpx.Error(w, http.StatusNotFound, "no telemetry for vehicle")
		return
	}
	httpx.JSON(w, http.StatusOK, snap)
}
