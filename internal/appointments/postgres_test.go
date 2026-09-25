package appointments

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/ford-retain/backend/internal/platform/postgres"
)

// testDB returns a connection to a fresh, empty schema (dropped afterwards),
// or skips when TEST_DATABASE_URL isn't set. `make test` and CI set it.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := postgres.Open(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })

	b := make([]byte, 6)
	rand.Read(b)
	schema := "test_" + hex.EncodeToString(b)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE") })

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := postgres.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPostgresRepo(t *testing.T) {
	testRepository(t, func(t *testing.T) Repository {
		db := testDB(t)
		if err := postgres.Migrate(ctx, db, "appointments", Migrations()); err != nil {
			t.Fatal(err)
		}
		return NewPostgresRepo(db)
	})
}

func TestPostgresMigrateAndSeed(t *testing.T) {
	db := testDB(t)
	for i := 0; i < 2; i++ { // the second run must be a no-op
		if err := postgres.Migrate(ctx, db, "appointments", Migrations()); err != nil {
			t.Fatalf("migrate run %d: %v", i+1, err)
		}
	}
	repo := NewPostgresRepo(db)
	seed := []Appointment{{ID: "apt-seed-1", VIN: "VIN1", DealershipID: "d1", Service: "Oil"}}
	if seeded, err := repo.SeedIfEmpty(ctx, seed); err != nil || !seeded {
		t.Fatalf("first seed: %v %v", seeded, err)
	}
	if seeded, err := repo.SeedIfEmpty(ctx, seed); err != nil || seeded {
		t.Fatalf("second seed must do nothing: %v %v", seeded, err)
	}
	if a, ok, err := repo.Get(ctx, "VIN1", "apt-seed-1"); err != nil || !ok || a.Service != "Oil" {
		t.Fatalf("seeded row: %+v %v %v", a, ok, err)
	}
}
