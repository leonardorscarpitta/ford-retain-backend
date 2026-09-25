package main

import (
	"net/http"
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/auth"
	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

const sessionTTL = 7 * 24 * time.Hour

// newHandler wires every module into one router, in three layers:
//
//	public   /healthz, POST /api/auth/login
//	signed   everything else under /api/ — needs a session (auth.RequireUser)
//	owned    /api/vehicles/{vin}/…      — the VIN must be the user's (vehicles.RequireOwner)
func newHandler(now time.Time, users []auth.User) http.Handler {
	authH := auth.NewHandler(auth.NewService(users, sessionTTL))
	vehiclesH := vehicles.NewHandler(vehicles.NewStore(seedVehicles()))
	dealershipStore := dealerships.NewStore(seedDealerships())

	owned := http.NewServeMux()
	notifications.NewHandler(notifications.NewStore(seedNotifications(now))).Register(owned)
	telemetry.NewHandler(telemetry.NewStore(seedTelemetry(now))).Register(owned)
	appointments.NewHandler(appointments.NewService(dealershipStore, seedAppointments(now))).Register(owned)

	signed := http.NewServeMux()
	authH.Register(signed)
	vehiclesH.Register(signed)
	dealerships.NewHandler(dealershipStore).Register(signed)
	signed.Handle("/api/vehicles/{vin}/", vehiclesH.RequireOwner(owned))

	public := http.NewServeMux()
	public.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	authH.RegisterPublic(public)
	public.Handle("/api/", authH.RequireUser(signed))
	return public
}
