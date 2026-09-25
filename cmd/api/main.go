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
	_ "time/tzdata" // the distroless image has no zoneinfo; dealerships need America/Sao_Paulo

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/auth"
	"github.com/ford-retain/backend/internal/fordapi"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/platform/postgres"
	"github.com/ford-retain/backend/internal/seed"
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
	users, err := seed.Users(auth.DefaultIterations)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Ford's platform owns vehicles, telemetry, alerts and dealers. Without
	// FORD_API_URL the API serves the seeded demo data instead.
	src := seedSources(time.Now())
	if base := os.Getenv("FORD_API_URL"); base != "" {
		src = fordSources(ctx, fordapi.New(base, os.Getenv("FORD_API_KEY")), 10*time.Minute)
		slog.Info("data source: Ford platform", "url", base)
	} else {
		slog.Info("data source: seed data (FORD_API_URL not set)")
	}

	bookings, closeDB, err := openBookings(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpx.Log(httpx.CORS(allowedOrigin, newHandler(users, src, bookings))),
		ReadHeaderTimeout: 5 * time.Second,
	}

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

// openBookings stores appointments in Postgres when DATABASE_URL is set
// (applying migrations, and loading demo bookings into an empty table when
// SEED_APPOINTMENTS=true), otherwise in memory with the demo bookings.
func openBookings(ctx context.Context) (appointments.Repository, func(), error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		slog.Info("appointments: in memory (DATABASE_URL not set)")
		return appointments.NewMemoryRepo(seed.Appointments(time.Now())), func() {}, nil
	}
	db, err := postgres.Open(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	if err := postgres.Migrate(ctx, db, "appointments", appointments.Migrations()); err != nil {
		db.Close()
		return nil, nil, err
	}
	repo := appointments.NewPostgresRepo(db)
	if os.Getenv("SEED_APPOINTMENTS") == "true" {
		seeded, err := repo.SeedIfEmpty(ctx, seed.Appointments(time.Now()))
		if err != nil {
			db.Close()
			return nil, nil, err
		}
		slog.Info("appointments: demo bookings", "loaded", seeded)
	}
	slog.Info("appointments: postgres")
	return repo, func() { db.Close() }, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
