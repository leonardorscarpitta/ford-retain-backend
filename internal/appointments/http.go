package appointments

import (
	"errors"
	"net/http"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vehicles/{vin}/appointments", h.list)
	mux.HandleFunc("POST /api/vehicles/{vin}/appointments", h.create)
	mux.HandleFunc("DELETE /api/vehicles/{vin}/appointments/{id}", h.cancel)
}

// list returns appointments, optionally filtered with ?when=upcoming|past.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	when := When(r.URL.Query().Get("when"))
	if !when.Valid() {
		httpx.Error(w, http.StatusBadRequest, "when must be upcoming or past")
		return
	}
	httpx.JSON(w, http.StatusOK, h.svc.ListByVIN(r.PathValue("vin"), when))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	a, err := h.svc.Create(r.PathValue("vin"), in)
	if errors.Is(err, ErrInvalid) {
		httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create appointment")
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Cancel(r.PathValue("vin"), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalid):
		httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "could not cancel appointment")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
