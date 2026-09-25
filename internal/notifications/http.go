package notifications

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ store *Store }

func NewHandler(s *Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles/{vin}/notifications", h.list)
	mux.HandleFunc("PATCH /api/notifications/{id}/read", h.markRead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.store.ListByVIN(r.PathValue("vin")))
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	n, ok := h.store.MarkRead(r.PathValue("id"))
	if !ok {
		httpx.Error(w, http.StatusNotFound, "notification not found")
		return
	}
	httpx.JSON(w, http.StatusOK, n)
}
