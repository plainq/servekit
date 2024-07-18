package pgconn

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

func (c *Conn) DeferredRollback(ctx context.Context, tx pgx.Tx, deferredErr *error) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		*deferredErr = errors.Join(*deferredErr, err)
	}
}

func (c *Conn) Close() error {
	c.Pool.Close()
	return nil
}

func (c *Conn) Health(ctx context.Context) error {
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: healthcheck failed: %w", err)
	}

	return nil
}

func PgError(err error) (error, bool) {
	var pgErr *pgconn.PgError

	if errors.Is(err, pgx.ErrNoRows) {
		return errkit.ErrNotFound, true
	}

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "02000":
			return fmt.Errorf("postgres: %w: %s", errkit.ErrNotFound, pgErr.Detail), true

		case "23505":
			return fmt.Errorf("postgres: %w: %s", errkit.ErrAlreadyExists, pgErr.Detail), true

		default:
			return errkit.Error(fmt.Sprintf("postgres: %s", pgErr.Error())), true
		}
	}

	return err, false
}
