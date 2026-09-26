package fordapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ford-retain/backend/internal/seed"
)

// newMockServer runs the mock over real HTTP so the client is exercised end to end.
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	now := time.Now()
	srv := httptest.NewServer(NewMock("test-key", MockData{
		Vehicles:  seed.Vehicles(),
		Telemetry: seed.Telemetry(now),
		Alerts:    seed.Notifications(now),
		Dealers:   seed.Dealerships(),
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClientAgainstMock(t *testing.T) {
	c := New(newMockServer(t).URL, "test-key")
	ctx := context.Background()

	garage, err := c.Vehicles().ForOwner(ctx, seed.AnaID)
	if err != nil || len(garage) != 2 || garage[0].VIN != seed.F150VIN || garage[0].OwnerID != seed.AnaID {
		t.Fatalf("Ana's garage: %+v, err %v", garage, err)
	}
	if none, err := c.Vehicles().ForOwner(ctx, "nobody"); err != nil || len(none) != 0 {
		t.Fatalf("unknown owner: %v %v", none, err)
	}

	snap, ok, err := c.Telemetry().Get(ctx, seed.RangerVIN)
	if err != nil || !ok || snap.MileageKm != 21870 {
		t.Fatalf("telemetry: %+v %v %v", snap, ok, err)
	}
	if _, ok, err := c.Telemetry().Get(ctx, "MISSING"); ok || err != nil {
		t.Fatalf("missing telemetry: ok=%v err=%v", ok, err)
	}

	alerts, err := c.Alerts().ListByVIN(ctx, seed.F150VIN)
	if err != nil || len(alerts) != 16 || alerts[0].Read {
		t.Fatalf("alerts: %d, err %v", len(alerts), err)
	}
	if n, ok, err := c.Alerts().MarkRead(ctx, seed.F150VIN, alerts[0].ID); err != nil || !ok || !n.Read {
		t.Fatalf("mark read: %+v %v %v", n, ok, err)
	}
	if again, _ := c.Alerts().ListByVIN(ctx, seed.F150VIN); !again[0].Read {
		t.Fatal("read state not kept")
	}
	if _, ok, err := c.Alerts().MarkRead(ctx, seed.RangerVIN, alerts[0].ID); ok || err != nil {
		t.Fatalf("alert on another vehicle: ok=%v err=%v", ok, err)
	}

	dealers, err := c.Dealers(ctx)
	if err != nil || len(dealers) != 24 {
		t.Fatalf("dealers: %d, err %v", len(dealers), err)
	}
	d := dealers[1] // Ford Pinheiros: weekday preset
	if d.Schedule.Location.String() != "America/Sao_Paulo" || d.Schedule.Bays != 2 || d.Schedule.Saturday == nil || d.Schedule.Saturday.Close != 12 {
		t.Fatalf("schedule not carried over: %+v", d.Schedule)
	}
}

func TestClientErrors(t *testing.T) {
	srv := newMockServer(t)
	ctx := context.Background()

	if _, err := New(srv.URL, "wrong-key").Vehicles().ForOwner(ctx, seed.AnaID); err == nil {
		t.Fatal("a rejected API key must be an error, not an empty garage")
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()
	if _, _, err := New(down.URL, "k").Telemetry().Get(ctx, seed.F150VIN); err == nil {
		t.Fatal("5xx must be an error")
	}

	unreachable := New("http://127.0.0.1:1", "k")
	if _, err := unreachable.Dealers(ctx); err == nil {
		t.Fatal("connection failure must be an error")
	}
}
