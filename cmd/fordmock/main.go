// Command fordmock is a stand-in for Ford's platform APIs, serving the demo
// data over the contract in internal/fordapi. Run it for local development
// and point the API at it with FORD_API_URL.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ford-retain/backend/internal/fordapi"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/seed"
)

func main() {
	addr := env("ADDR", ":8090")
	now := time.Now()
	handler := fordapi.NewMock(env("FORD_API_KEY", "dev-key"), fordapi.MockData{
		Vehicles:  seed.Vehicles(),
		Telemetry: seed.Telemetry(now),
		Alerts:    seed.Notifications(now),
		Dealers:   seed.Dealerships(),
	})
	srv := &http.Server{Addr: addr, Handler: httpx.Log(handler), ReadHeaderTimeout: 5 * time.Second}
	slog.Info("mock Ford API listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("mock Ford API stopped", "err", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
