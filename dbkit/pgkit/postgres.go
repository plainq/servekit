package pgkit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/heartwilltell/hc"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plainq/servekit/errkit"
)

// Compilation time check that Conn implements
// the hc.HealthChecker.
var _ hc.HealthChecker = (*Conn)(nil)

// Conn represents connection to Postgres.
type Conn struct{ *pgxpool.Pool }

// New returns a pointer to a new instance of *Conn structure.
// Takes connstr - connection string in Postgres format.
func New(connstr string) (*Conn, error) {
	config, parseErr := pgxpool.ParseConfig(connstr)
	if parseErr != nil {
		return nil, fmt.Errorf("postgres: failed to parse config: %w", parseErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pgConn, connErr := pgxpool.NewWithConfig(ctx, config)
	if connErr != nil {
		return nil, fmt.Errorf("postgres: %w: %s", errkit.ErrConnFailed, connErr.Error())
	}

	return &Conn{Pool: pgConn}, nil
}

// Close closes the connection to the Postgres database.
func (c *Conn) Close() error {
	c.Pool.Close()
	return nil
}

// Health checks if the connection is healthy.
func (c *Conn) Health(ctx context.Context) error {
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: healthcheck failed: %w", err)
	}

	return nil
}

// PgError checks if the error is a Postgres error and returns true if it is.
// If it is, it returns the error wrapped with the appropriate error from errkit package.
// Otherwise, it returns false and the original error.
func PgError(err error) (bool, error) {
	var pgErr *pgconn.PgError

	if errors.Is(err, pgx.ErrNoRows) {
		return true, errkit.ErrNotFound
	}

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "02000":
			return true, fmt.Errorf("postgres: %w: %s", errkit.ErrNotFound, pgErr.Detail)

		case "23505":
			return true, fmt.Errorf("postgres: %w: %s", errkit.ErrAlreadyExists, pgErr.Detail)

		default:
			return true, errkit.Error(fmt.Sprintf("postgres: %s", pgErr.Error()))
		}
	}

	return false, err
}
