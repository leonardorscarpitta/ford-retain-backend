package dealerships

import (
	"net/http"
	"strconv"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ store *Store }

func NewHandler(s *Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dealerships", h.list)
}

// list returns all dealerships, sorted by distance when ?lat=&lng= are given.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("lat") == "" || q.Get("lng") == "" {
		httpx.JSON(w, http.StatusOK, h.store.All())
		return
	}
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(q.Get("lng"), 64)
	if err1 != nil || err2 != nil {
		httpx.Error(w, http.StatusBadRequest, "lat and lng must be numbers")
		return
	}
	httpx.JSON(w, http.StatusOK, h.store.Nearest(lat, lng))
}
