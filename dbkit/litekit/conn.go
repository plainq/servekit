package litekit

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/heartwilltell/hc"
	_ "github.com/mattn/go-sqlite3" //nolint // driver.
)

// Compilation time check that Conn implements the hc.HealthChecker.
var _ hc.HealthChecker = (*Conn)(nil)

const (
	// ErrUnsupportedAccessMode indicates that AccessMode which has been
	// passed to the New function by WithAccessMode option is not
	// supported by the Storage.
	ErrUnsupportedAccessMode Error = "unsupported access mode"

	// ErrUnsupportedJournalMode indicates that JournalMode which has been
	// passed to the New function by WithJournalMode option is not
	// supported by the Storage.
	ErrUnsupportedJournalMode Error = "unsupported journal mode"
)

// Error represents package level errors related to the storage engine.
type Error string

func (e Error) Error() string { return string(e) }

// JournalMode represents SQLite journaling mode.
// https://www.sqlite.org/pragma.html#pragma_journal_mode
//
// Note that the JournalMode for an InMemory database is either Memory or Off
// and can not be changed to a different value.
//
// An attempt to change the JournalMode of an InMemory database to any setting
// other than Memory or Off is ignored.
//
// Note also that the JournalMode cannot be changed while a transaction is active.
type JournalMode byte

const (
	// Delete journaling mode is the normal behavior. In the Delete mode, the rollback
	// journal is deleted at the conclusion of each transaction. Indeed, the delete
	// operation is the action that causes the transaction to commit.
	// See the document titled Atomic Commit In SQLite for additional detail:
	// https://www.sqlite.org/atomiccommit.html
	Delete JournalMode = iota

	// Truncate journaling mode commits transactions by truncating the rollback journal to
	// zero-length instead of deleting it. On many systems, truncating a file is much faster
	// than deleting the file since the containing directory does not need to be changed.
	Truncate

	// Persist journaling mode prevents the rollback journal from being deleted at the end of
	// each transaction. Instead, the header of the journal is overwritten with zeros.
	// This will prevent other database connections from rolling the journal back.
	// The Persist journaling mode is useful as an optimization on platforms where deleting or
	// truncating a file is much more expensive than overwriting the first block of a file with zeros.
	Persist

	// Memory journaling mode stores the rollback journal in volatile RAM.
	// This saves disk I/O but at the expense of database safety and integrity.
	// If the application using SQLite crashes in the middle of a transaction when
	// the Memory journaling mode is set, then the database file will very likely go corrupt.
	Memory

	// WAL journaling mode uses a write-ahead log instead of a rollback journal to implement transactions.
	// The WAL journaling mode is persistent; after being set it stays in effect across multiple database
	// connections and after closing and reopening the database.
	// A database in WAL journaling mode can only be accessed by SQLite version 3.7.0 (2010-07-21) or later.
	WAL

	// Off journaling mode disables the rollback journal completely.
	// No rollback journal is ever created and hence there is never a rollback journal to delete.
	// The Off journaling mode disables the atomic commit and rollback capabilities of SQLite.
	// The ROLLBACK command no longer works; it behaves in an undefined way.
	// Applications must avoid using the ROLLBACK command when the journal mode is Off.
	// If the application crashes in the middle of a transaction when the Off journaling mode is set,
	// then the database file will very likely go corrupt.
	Off
)

// JournalModeFromString converts a string representation of a journal mode to a JournalMode type.
// It returns the corresponding JournalMode and a nil error if the conversion is successful.
// If the input string does not match any known journal mode, it returns the default Delete mode
// and an error indicating that the access mode is unsupported.
// The supported journal modes are "WAL", "Delete", "Memory", "Off", "Persist", and "Truncate".
// The conversion is case-insensitive.
func JournalModeFromString(mode string) (JournalMode, error) {
	switch strings.ToLower(mode) {
	case WAL.String():
		return WAL, nil

	case Delete.String():
		return Delete, nil

	case Memory.String():
		return Memory, nil

	case Off.String():
		return Off, nil

	case Persist.String():
		return Persist, nil

	case Truncate.String():
		return Truncate, nil

	default:
		return Delete, fmt.Errorf("%w: %q", ErrUnsupportedAccessMode, mode)
	}
}

func (m JournalMode) String() string {
	modes := map[JournalMode]string{
		Delete:   "DELETE",
		Truncate: "TRUNCATE",
		Persist:  "PERSIST",
		Memory:   "MEMORY",
		WAL:      "WAL",
		Off:      "OFF",
	}

	return modes[m]
}

// AccessMode represents SQLite access mode.
// https://www.sqlite.org/c3ref/open.html
type AccessMode byte

func (m AccessMode) String() string {
	modes := map[AccessMode]string{
		ReadWriteCreate: "rwc",
		ReadOnly:        "ro",
		ReadWrite:       "rw",
		InMemory:        "memory",
	}

	return modes[m]
}

// AccessModeFromString converts a string representation of an access mode to an AccessMode type.
// It returns the corresponding AccessMode and a nil error if the conversion is successful.
// If the input string does not match any known access mode, it returns the default ReadWriteCreate mode
// and an error indicating that the access mode is unsupported.
// The supported access modes are "rwc" (ReadWriteCreate), "ro" (ReadOnly), "rw" (ReadWrite), and "memory" (InMemory).
// The conversion is case-insensitive.
func AccessModeFromString(mode string) (AccessMode, error) {
	switch strings.ToLower(mode) {
	case InMemory.String():
		return InMemory, nil

	case ReadWriteCreate.String():
		return ReadWriteCreate, nil

	case ReadWrite.String():
		return ReadWrite, nil

	case ReadOnly.String():
		return ReadOnly, nil

	default:
		return ReadWriteCreate, fmt.Errorf("%w: %q", ErrUnsupportedAccessMode, mode)
	}
}

const (
	// ReadWriteCreate is a mode in which the database is opened for reading and writing,
	// and is created if it does not already exist.
	ReadWriteCreate AccessMode = iota

	// ReadOnly is a mode in which the database is opened in read-only mode.
	// If the database does not already exist, an error is returned.
	ReadOnly

	// ReadWrite is a mode in which database is opened for reading and writing if possible,
	// or reading only if the file is write-protected by the operating system.
	// In either case the database must already exist, otherwise an error is returned.
	// For historical reasons, if opening in read-write mode fails due to OS-level permissions,
	// an attempt is made to open it in read-only mode.
	ReadWrite

	// InMemory is a mode in which database will be opened as an in-memory database.
	// The database is named by the "filename" argument for the purposes of cache-sharing,
	// if shared cache mode is enabled, but the "filename" is otherwise ignored.
	InMemory
)

// Option represents an optional functions which configures the Storage.
type Option func(c *Conn)

// WithAccessMode enables SQLite to use picked access mode.
func WithAccessMode(mode AccessMode) Option {
	return func(c *Conn) { c.accessMode = mode }
}

// WithJournalMode sets SQLite to use picked journal mode.
func WithJournalMode(mode JournalMode) Option {
	return func(c *Conn) { c.journalingMode = mode }
}

type Conn struct {
	*sql.DB

	// path holds an absolute path to the database file.
	path string

	// accessMode represents SQLite access mode.
	// https://www.sqlite.org/c3ref/open.html
	accessMode AccessMode

	// journalingMode represents SQLite journaling mode.
	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	journalingMode JournalMode
}

func New(path string, options ...Option) (*Conn, error) {
	absPath, absPathErr := filepath.Abs(path)
	if absPathErr != nil {
		return nil, fmt.Errorf("determite absolute path to the database file: %w", absPathErr)
	}

	conn := Conn{
		DB:             nil,
		path:           absPath,
		accessMode:     ReadWriteCreate,
		journalingMode: WAL,
	}

	for _, option := range options {
		option(&conn)
	}

	connString, connStringErr := conn.connString()
	if connStringErr != nil {
		return nil, fmt.Errorf("parsing connection string: %w", connStringErr)
	}

	db, openErr := sql.Open("sqlite3", connString)
	if openErr != nil {
		return nil, fmt.Errorf("open database: %w", openErr)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	conn.DB = db

	return &conn, nil
}

// Health implements hc.HealthChecker interface.
func (c *Conn) Health(ctx context.Context) error {
	if err := c.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite: health check: %w", err)
	}

	return nil
}

func (c *Conn) connString() (string, error) {
	params := make([]string, 0, 2)

	switch c.accessMode {
	case ReadWriteCreate, InMemory, ReadOnly, ReadWrite:
		params = append(params, "mode="+c.accessMode.String())

	default:
		return "", ErrUnsupportedAccessMode
	}

	switch c.journalingMode {
	case Delete, Truncate, Persist, WAL, Memory, Off:
		params = append(params, "_journal="+c.journalingMode.String())

	default:
		return "", ErrUnsupportedJournalMode
	}

	var b strings.Builder

	b.WriteString("file:" + c.path)

	if len(params) > 0 {
		b.WriteString("?")
	}

	for i, p := range params {
		b.WriteString(p)

		if i != len(params)-1 {
			b.WriteString("&")
		}
	}

	return b.String(), nil
}
