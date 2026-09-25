package vehicles

import (
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/platform/identity"
)

type Handler struct{ store *Store }

func NewHandler(s *Store) *Handler { return &Handler{store: s} }

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
		if _, ok := h.store.Owned(identity.UserID(r.Context()), r.PathValue("vin")); !ok {
			httpx.Error(w, http.StatusNotFound, "vehicle not found")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.store.ForOwner(identity.UserID(r.Context())))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, ok := h.store.Owned(identity.UserID(r.Context()), r.PathValue("vin"))
	if !ok {
		httpx.Error(w, http.StatusNotFound, "vehicle not found")
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}
