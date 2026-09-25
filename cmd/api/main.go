// Command api runs the Ford Retain HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := env("ADDR", ":8080")
	allowedOrigin := env("ALLOWED_ORIGIN", "http://localhost:3000")
	now := time.Now()

	// Wire modules. Each module owns its data and registers its own routes.
	dealershipStore := dealerships.NewStore(seedDealerships())
	modules := []interface{ Register(*http.ServeMux) }{
		vehicles.NewHandler(vehicles.NewStore(seedVehicles())),
		notifications.NewHandler(notifications.NewStore(seedNotifications(now))),
		telemetry.NewHandler(telemetry.NewStore(seedTelemetry(now))),
		dealerships.NewHandler(dealershipStore),
		appointments.NewHandler(appointments.NewService(dealershipStore, seedAppointments(now))),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	for _, m := range modules {
		m.Register(mux)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpx.Log(httpx.CORS(allowedOrigin, mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
