package vehicles

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/platform/identity"
)

type Handler struct{ source Source }

func NewHandler(s Source) *Handler { return &Handler{source: s} }

// Register adds the garage routes. They need an authenticated user.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles", h.list)
	mux.HandleFunc("GET /api/vehicles/{vin}", h.get)
}

// RequireOwner guards routes under /api/vehicles/{vin}/: the vehicle must
// belong to the authenticated user. Anything else is reported as not found,
// so VINs that belong to other owners aren't revealed.
func (h *Handler) RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok, err := Owned(r.Context(), h.source, identity.UserID(r.Context()), r.PathValue("vin"))
		if err != nil {
			httpx.Upstream(w, r, err)
			return
		}
		if !ok {
			httpx.Error(w, http.StatusNotFound, "vehicle not found")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.source.ForOwner(r.Context(), identity.UserID(r.Context()))
	if err != nil {
		httpx.Upstream(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, ok, err := Owned(r.Context(), h.source, identity.UserID(r.Context()), r.PathValue("vin"))
	if err != nil {
		httpx.Upstream(w, r, err)
		return
	}
	if !ok {
		httpx.Error(w, http.StatusNotFound, "vehicle not found")
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}
