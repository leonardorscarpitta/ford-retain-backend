package vehicles

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ store *Store }

func NewHandler(s *Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles", h.list)
	mux.HandleFunc("GET /api/vehicles/{vin}", h.get)
}

func (h *Handler) list(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, h.store.All())
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, ok := h.store.Get(r.PathValue("vin"))
	if !ok {
		httpx.Error(w, http.StatusNotFound, "vehicle not found")
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}
