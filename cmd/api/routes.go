package main

import (
	"net/http"
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/auth"
	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/seed"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

const sessionTTL = 7 * 24 * time.Hour

// newHandler wires every module into one router, in three layers:
//
//	public   /healthz, POST /api/auth/login
//	signed   everything else under /api/ — needs a session (auth.RequireUser)
//	owned    /api/vehicles/{vin}/…      — the VIN must be the user's (vehicles.RequireOwner)
//
// Data owned by Ford's platform comes from src (seed data or Ford's APIs);
// appointments are this service's own.
func newHandler(now time.Time, users []auth.User, src sources) http.Handler {
	authH := auth.NewHandler(auth.NewService(users, sessionTTL))
	vehiclesH := vehicles.NewHandler(src.vehicles)
	dealershipStore := src.dealers

	// Scheduling reads opening hours through a function, so appointments
	// doesn't depend on the dealerships package.
	hours := func(id string, day time.Weekday) (appointments.Hours, bool) {
		s, ok := dealershipStore.ScheduleOf(id)
		if !ok {
			return appointments.Hours{}, false
		}
		h, open := s.On(day)
		if !open {
			return appointments.Hours{Location: s.Location}, true
		}
		return appointments.Hours{Open: h.Open, Close: h.Close, Bays: s.Bays, Location: s.Location}, true
	}
	appointmentsH := appointments.NewHandler(appointments.NewService(hours, seed.Appointments(now)))

	owned := http.NewServeMux()
	notifications.NewHandler(src.alerts).Register(owned)
	telemetry.NewHandler(src.telemetry).Register(owned)
	appointmentsH.Register(owned)

	signed := http.NewServeMux()
	authH.Register(signed)
	vehiclesH.Register(signed)
	dealerships.NewHandler(dealershipStore).Register(signed)
	appointmentsH.RegisterAvailability(signed)
	signed.Handle("/api/vehicles/{vin}/", vehiclesH.RequireOwner(owned))

	public := http.NewServeMux()
	public.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	authH.RegisterPublic(public)
	public.Handle("/api/", authH.RequireUser(signed))
	return public
}
