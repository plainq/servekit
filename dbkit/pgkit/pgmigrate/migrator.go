package pgmigrate

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
)

// New returns a pointer to a new instance of Migrator.
func New(conn *pgxpool.Pool, migrations fs.FS) (*Migrator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mConn, mConnErr := conn.Acquire(ctx)
	if mConnErr != nil {
		return nil, fmt.Errorf("failed to acquire connection for migrator: %w", mConnErr)
	}

	m, mErr := migrate.NewMigratorEx(ctx, mConn.Conn(), "migration", &migrate.MigratorOptions{})
	if mErr != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", mErr)
	}

	if err := m.LoadMigrations(migrations); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return &Migrator{conn: mConn.Conn(), m: m}, nil
}

// Migrator holds logic of how to load and apply database
// migrations to Postgres.
type Migrator struct {
	conn *pgx.Conn
	m    *migrate.Migrator
}

func (m *Migrator) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return m.conn.Close(ctx)
}

// Migrate performs migration of database schema.
func (m *Migrator) Migrate(ctx context.Context) error { return m.m.Migrate(ctx) }
