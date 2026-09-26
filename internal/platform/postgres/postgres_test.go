package postgres

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

// Needs no database: an empty migration set is rejected before connecting
// would matter, catching embed paths that match nothing.
func TestMigrateRejectsEmptySet(t *testing.T) {
	err := Migrate(context.Background(), nil, "x", fstest.MapFS{"migrations/001.sql": {}})
	if err == nil || !strings.Contains(err.Error(), "no *.sql migrations") {
		t.Fatalf("want an error for a set with no top-level *.sql, got %v", err)
	}
}
