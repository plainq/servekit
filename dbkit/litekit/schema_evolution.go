package litekit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	schema = `
		create table if not exists "schema_version"
		(
			id         int       default 0                 not null,
			version    int       default 0                 not null,
			created_at timestamp default current_timestamp not null,
			updated_at timestamp default current_timestamp not null,
		
			constraint schema_version_pk
				primary key (id)
		);
		
		create unique index if not exists id_uindex
			on schema_version (id);
		
		insert into schema_version default
		values;
	`

	querySelectSchemaVersionTable = `select name from sqlite_master where type='table' and name='schema_version';`
	querySelectSchemaVersionInfo  = `select * from schema_version where id=0;`
	queryUpdateSchemaVersionInfo  = `update schema_version set version = ? where id = 0;`

	// mutationTimeout represents default timeout for the Storage schema mutation.
	mutationTimeout = 30 * time.Second
)

// Mutation represents a single schema mutation.
// It contains the version number of the mutation and the changes to be applied.
type Mutation struct {
	version uint
	changes []byte
}

// SchemaVersionInfo represents information about a schema version.
// It contains the id, version number, and timestamps of creation and update.
type SchemaVersionInfo struct {
	ID        int
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WithBackupBeforeMutations returns an EvolverOption that sets
// the backupBeforeMutations field of the Evolver to true.
func WithBackupBeforeMutations() EvolverOption {
	return func(e *Evolver) { e.backupBeforeMutations = true }
}

// WithMutationTimeout returns an EvolverOption that sets
// the mutationTimeout field of the Evolver to the specified timeout value.
func WithMutationTimeout(timeout time.Duration) EvolverOption {
	return func(e *Evolver) { e.mutationTimeout = timeout }
}

type EvolverOption func(*Evolver)

// Evolver is responsible for database schema evolution.
type Evolver struct {
	db *Conn

	// mutations holds virtual filesystem with schema mutation files.
	mutations fs.FS

	// backupBeforeMutations indicates whether to back up the database file
	// before the schema mutation.
	backupBeforeMutations bool

	// mutationTimeout represents timeout after which the mutation considered as failed.
	mutationTimeout time.Duration
}

// NewEvolver creates a new instance of Evolver.
func NewEvolver(db *Conn, mutations fs.FS, options ...EvolverOption) (*Evolver, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	e := Evolver{
		db:                    db,
		mutations:             mutations,
		backupBeforeMutations: false,
		mutationTimeout:       mutationTimeout,
	}

	for _, option := range options {
		option(&e)
	}

	return &e, nil
}

func (e *Evolver) MutateSchema() (eErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.mutationTimeout)
	defer cancel()

	var schemaVersionInfo SchemaVersionInfo

	if err := e.ensureSchemaVersionTable(ctx); err != nil {
		return err
	}

	if err := e.db.QueryRowContext(ctx, querySelectSchemaVersionInfo).Scan(
		&schemaVersionInfo.ID,
		&schemaVersionInfo.Version,
		&schemaVersionInfo.CreatedAt,
		&schemaVersionInfo.UpdatedAt,
	); err != nil {
		return fmt.Errorf("get schema_version info: %w", err)
	}

	mutations, loadErr := e.loadMutations()
	if loadErr != nil {
		return loadErr
	}

	if e.backupBeforeMutations {
		if err := e.backupBeforeMutate(); err != nil {
			return err
		}
	}

	tx, txErr := e.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if txErr != nil {
		return fmt.Errorf("begin transaction: %w", txErr)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			eErr = errors.Join(eErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Range over mutation to apply each one.
	for _, m := range mutations {
		// Skip already applied mutations.
		if schemaVersionInfo.Version < 0 || m.version <= uint(schemaVersionInfo.Version) {
			continue
		}

		if _, err := tx.ExecContext(ctx, string(m.changes)); err != nil {
			return fmt.Errorf("apply schema mutation: %w", err)
		}

		if _, err := tx.ExecContext(ctx, queryUpdateSchemaVersionInfo, m.version); err != nil {
			return fmt.Errorf("update schema_version table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (e *Evolver) ensureSchemaVersionTable(ctx context.Context) error {
	var svt string

	if err := e.db.QueryRowContext(ctx, querySelectSchemaVersionTable).Scan(&svt); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Create schema_version table.
			if _, err := e.db.ExecContext(ctx, schema); err != nil {
				return fmt.Errorf("create schema_version table: %w", err)
			}

		default:
			return fmt.Errorf("check schema_version table exist: %w", err)
		}
	}

	return nil
}

func (e *Evolver) loadMutations() ([]Mutation, error) {
	entries, readErr := fs.ReadDir(e.mutations, ".")
	if readErr != nil {
		return nil, fmt.Errorf("load mutations: %w", readErr)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	evolutions := make([]Mutation, 0, len(entries))

	for i, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("get mutation file info: %w", infoErr)
		}

		if strings.HasSuffix(info.Name(), ".sql") {
			changes, readFileErr := fs.ReadFile(e.mutations, info.Name())
			if readFileErr != nil {
				return nil, fmt.Errorf("load mutation file '%s': %w", info.Name(), readFileErr)
			}

			evolutions = append(evolutions, Mutation{
				version: uint(i + 1), //nolint:gosec // i is always positive
				changes: changes,
			})
		}
	}

	return evolutions, nil
}

func (e *Evolver) backupBeforeMutate() (bErr error) {
	stat, statErr := os.Stat(e.db.path)
	if statErr != nil {
		return fmt.Errorf("get database file information: %w", statErr)
	}

	if stat.IsDir() {
		return fmt.Errorf("database path is a directory istead of file: %s", e.db.path)
	}

	srcDir, _ := path.Split(e.db.path)

	src, srcOpenErr := os.Open(e.db.path)
	if srcOpenErr != nil {
		return fmt.Errorf("open database file: %w", srcOpenErr)
	}

	defer func() {
		if err := src.Close(); err != nil {
			bErr = errors.Join(bErr, fmt.Errorf("close database source file: %w", err))
		}
	}()

	dst, dstOpenErr := os.Create(path.Join(srcDir, fmt.Sprintf("backup-%s", stat.Name())))
	if dstOpenErr != nil {
		return fmt.Errorf("create database backup file: %w", dstOpenErr)
	}

	defer func() {
		if err := dst.Close(); err != nil {
			bErr = errors.Join(bErr, fmt.Errorf("close database backup file: %w", err))
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	return nil
}
