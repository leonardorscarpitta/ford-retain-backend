package appointments

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ford-retain/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// RegisterAvailability adds the availability route. It isn't tied to a
// vehicle, so mount it with the other signed-in routes.
func (h *Handler) RegisterAvailability(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dealerships/{id}/availability", h.availability)
}

// Register adds the vehicle-scoped routes (mount behind the ownership guard).
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
	list, err := h.svc.ListByVIN(r.Context(), r.PathValue("vin"), when)
	if err != nil {
		internalError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	a, err := h.svc.Create(r.Context(), r.PathValue("vin"), in)
	if errors.Is(err, ErrInvalid) {
		httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Cancel(r.Context(), r.PathValue("vin"), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalid):
		httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
	case err != nil:
		internalError(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

const maxAvailabilityDays = 31

type availabilityResponse struct {
	DealershipID string `json:"dealershipId"`
	TimeZone     string `json:"timeZone"`
	Days         []Day  `json:"days"`
}

// availability returns bookable slots: ?from=YYYY-MM-DD (default today) and
// ?days=N (default 14, max 31).
func (h *Handler) availability(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	days := 14
	if v := q.Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxAvailabilityDays {
			httpx.Error(w, http.StatusBadRequest, "days must be between 1 and 31")
			return
		}
		days = n
	}
	from := h.svc.now()
	if v := q.Get("from"); v != "" {
		d, err := time.Parse(time.DateOnly, v)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "from must be a date (YYYY-MM-DD)")
			return
		}
		// Midday UTC lands on the same calendar date in every Americas zone.
		from = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
	}
	id := r.PathValue("id")
	out, loc, err := h.svc.Availability(r.Context(), id, from, days)
	if errors.Is(err, ErrUnknownDealership) {
		httpx.Error(w, http.StatusNotFound, "dealership not found")
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, availabilityResponse{DealershipID: id, TimeZone: loc.String(), Days: out})
}

// internalError logs a storage failure and hides the details from the caller.
func internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("appointments", "method", r.Method, "path", r.URL.Path, "err", err)
	httpx.Error(w, http.StatusInternalServerError, "something went wrong, please try again")
}
