package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/fordapi"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/seed"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

// sources is where the data owned by Ford's platform comes from.
type sources struct {
	vehicles  vehicles.Source
	telemetry telemetry.Source
	alerts    notifications.Source
	dealers   *dealerships.Store
}

// seedSources serves the demo data (no Ford platform needed).
func seedSources(now time.Time) sources {
	return sources{
		vehicles:  vehicles.NewStore(seed.Vehicles()),
		telemetry: telemetry.NewStore(seed.Telemetry(now)),
		alerts:    notifications.NewStore(seed.Notifications(now)),
		dealers:   dealerships.NewStore(seed.Dealerships()),
	}
}

const dealerRetry = 15 * time.Second

// fordSources reads from Ford's platform. Vehicles, telemetry and alerts are
// fetched per request. The dealer directory is reference data: it's loaded
// now and refreshed in the background (retrying sooner while it has never
// loaded) until ctx ends, so scheduling lookups never wait on the network.
func fordSources(ctx context.Context, c *fordapi.Client, refresh time.Duration) sources {
	dealers := dealerships.NewStore(nil)
	load := func() bool {
		lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		list, err := c.Dealers(lctx)
		if err != nil {
			slog.Error("loading Ford dealer directory", "err", err)
			return false
		}
		dealers.Replace(list)
		slog.Info("Ford dealer directory loaded", "dealers", len(list))
		return true
	}
	loaded := load()
	go func() {
		for {
			wait := refresh
			if !loaded {
				wait = dealerRetry
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				loaded = load() || loaded
			}
		}
	}()
	return sources{
		vehicles:  c.Vehicles(),
		telemetry: c.Telemetry(),
		alerts:    c.Alerts(),
		dealers:   dealers,
	}
}
