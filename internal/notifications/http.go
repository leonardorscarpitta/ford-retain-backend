package notifications

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ source Source }

func NewHandler(s Source) *Handler { return &Handler{source: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles/{vin}/notifications", h.list)
	mux.HandleFunc("PATCH /api/vehicles/{vin}/notifications/{id}/read", h.markRead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.source.ListByVIN(r.Context(), r.PathValue("vin"))
	if err != nil {
		httpx.Upstream(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	n, ok, err := h.source.MarkRead(r.Context(), r.PathValue("vin"), r.PathValue("id"))
	if err != nil {
		httpx.Upstream(w, r, err)
		return
	}
	if !ok {
		httpx.Error(w, http.StatusNotFound, "notification not found")
		return
	}
	httpx.JSON(w, http.StatusOK, n)
}
