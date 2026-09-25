package appointments

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrations returns the module's SQL migrations (the files in migrations/),
// to apply with postgres.Migrate.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return sub
}

// PostgresRepo stores appointments in Postgres.
type PostgresRepo struct{ db *sql.DB }

func NewPostgresRepo(db *sql.DB) *PostgresRepo { return &PostgresRepo{db: db} }

const columns = `id, vin, dealership_id, scheduled_at, service, notes, created_at`

func scan(row interface{ Scan(...any) error }) (Appointment, error) {
	var a Appointment
	err := row.Scan(&a.ID, &a.VIN, &a.DealershipID, &a.ScheduledAt, &a.Service, &a.Notes, &a.CreatedAt)
	return a, err
}

func (p *PostgresRepo) ListByVIN(ctx context.Context, vin string) ([]Appointment, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT `+columns+` FROM appointments WHERE vin = $1`, vin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Appointment{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (p *PostgresRepo) Get(ctx context.Context, vin, id string) (Appointment, bool, error) {
	a, err := scan(p.db.QueryRowContext(ctx, `SELECT `+columns+` FROM appointments WHERE vin = $1 AND id = $2`, vin, id))
	if err == sql.ErrNoRows {
		return Appointment{}, false, nil
	}
	return a, err == nil, err
}

// Book counts and inserts inside one transaction that holds an advisory lock
// for this dealership and time, so concurrent bookings (even from several API
// instances) queue for the slot and can't overfill it.
func (p *PostgresRepo) Book(ctx context.Context, a Appointment, bays int) (Appointment, bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Appointment{}, false, err
	}
	defer tx.Rollback()

	slotKey := fmt.Sprintf("appointment-slot|%s|%d", a.DealershipID, a.ScheduledAt.Unix())
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, slotKey); err != nil {
		return Appointment{}, false, err
	}
	var booked int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM appointments WHERE dealership_id = $1 AND scheduled_at = $2`, a.DealershipID, a.ScheduledAt,
	).Scan(&booked); err != nil {
		return Appointment{}, false, err
	}
	if booked >= bays {
		return Appointment{}, false, nil
	}
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO appointments (id, vin, dealership_id, scheduled_at, service, notes, created_at)
		 VALUES ('apt-' || nextval('appointment_ids'), $1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		a.VIN, a.DealershipID, a.ScheduledAt, a.Service, a.Notes, a.CreatedAt,
	).Scan(&a.ID); err != nil {
		return Appointment{}, false, err
	}
	return a, true, tx.Commit()
}

func (p *PostgresRepo) Delete(ctx context.Context, vin, id string) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM appointments WHERE vin = $1 AND id = $2`, vin, id)
	return err
}

func (p *PostgresRepo) Booked(ctx context.Context, dealershipID string, from, to time.Time) (map[time.Time]int, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT scheduled_at, count(*) FROM appointments
		 WHERE dealership_id = $1 AND scheduled_at >= $2 AND scheduled_at < $3
		 GROUP BY scheduled_at`, dealershipID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[time.Time]int{}
	for rows.Next() {
		var at time.Time
		var n int
		if err := rows.Scan(&at, &n); err != nil {
			return nil, err
		}
		out[at.UTC()] = n
	}
	return out, rows.Err()
}

// SeedIfEmpty inserts demo appointments (keeping their IDs) only when the
// table is empty. For development databases; never enable it in production.
func (p *PostgresRepo) SeedIfEmpty(ctx context.Context, items []Appointment) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var any bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM appointments)`).Scan(&any); err != nil {
		return false, err
	}
	if any {
		return false, nil
	}
	for _, a := range items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO appointments (`+columns+`) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			a.ID, a.VIN, a.DealershipID, a.ScheduledAt, a.Service, a.Notes, a.CreatedAt,
		); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}
