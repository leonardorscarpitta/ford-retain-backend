package telemetry

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ source Source }

func NewHandler(s Source) *Handler { return &Handler{source: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles/{vin}/telemetry", h.get)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	snap, ok, err := h.source.Get(r.Context(), r.PathValue("vin"))
	if err != nil {
		httpx.Upstream(w, r, err)
		return
	}
	if !ok {
		httpx.Error(w, http.StatusNotFound, "no telemetry for vehicle")
		return
	}
	httpx.JSON(w, http.StatusOK, snap)
}
