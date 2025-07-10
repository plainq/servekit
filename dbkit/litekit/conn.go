package litekit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/benbjohnson/litestream/file"
	"github.com/benbjohnson/litestream/s3"
	"github.com/heartwilltell/hc"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver is required by the litestream.
	"github.com/plainq/servekit/logkit"
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

// WithBackupToS3 sets backup to S3-like storages.
func WithBackupToS3(cfg S3BackupConfig) Option {
	return func(c *Conn) {
		c.backup = true
		c.backupTo = S3
		c.backupS3 = cfg
	}
}

// S3BackupConfig holds S3 backup configuration.
type S3BackupConfig struct {
	RestoreTimeout  time.Duration
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Endpoint        string
}

// FileBackupConfig holds file backup configuration.
type FileBackupConfig struct {
	RestoreTimeout time.Duration
	Path           string
}

// WithBackupToFile sets backup to a file.
func WithBackupToFile(cfg FileBackupConfig) Option {
	return func(c *Conn) {
		c.backup = true
		c.backupTo = File
		c.backupFile = cfg
	}
}

// To represents backup destination type.
type To byte

const (
	// S3 represents backup destination to S3-like storages.
	S3 To = iota + 1

	// File represents backup destination to a file.
	File
)

type Conn struct {
	*sql.DB
	logger *slog.Logger

	// path holds an absolute path to the database file.
	path string

	// accessMode represents SQLite access mode.
	// https://www.sqlite.org/c3ref/open.html
	accessMode AccessMode

	// journalingMode represents SQLite journaling mode.
	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	journalingMode JournalMode

	backup               bool
	backupCloser         io.Closer
	backupTo             To
	backupS3             S3BackupConfig
	backupFile           FileBackupConfig
	backupRestoreTimeout time.Duration
}

func New(path string, options ...Option) (*Conn, error) {
	absPath, absPathErr := filepath.Abs(path)
	if absPathErr != nil {
		return nil, fmt.Errorf("determite absolute path to the database file: %w", absPathErr)
	}

	conn := Conn{
		DB:             nil,
		logger:         logkit.NewNop(),
		path:           absPath,
		accessMode:     ReadWriteCreate,
		journalingMode: WAL,
	}

	for _, option := range options {
		option(&conn)
	}

	conn.logger.Debug("Setting up database backups")

	if conn.backup {
		if err := conn.configureBackups(); err != nil {
			return nil, fmt.Errorf("sqlite: setup backups: %w", err)
		}
	}

	connString, connStringErr := conn.connString()
	if connStringErr != nil {
		return nil, fmt.Errorf("sqlite: parsing connection string: %w", connStringErr)
	}

	conn.logger.Debug("Opening database connection")

	db, openErr := sql.Open("sqlite3", connString)
	if openErr != nil {
		return nil, fmt.Errorf("sqlite: open database: %w", openErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("sqlite: connect to database: %w", err)
	}

	conn.DB = db

	conn.logger.Debug("Database connection has been established")

	return &conn, nil
}

// Health implements hc.HealthChecker interface.
func (c *Conn) Health(ctx context.Context) error {
	if err := c.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite: health check: %w", err)
	}

	return nil
}

func (c *Conn) Close() (closeErr error) {
	if c.backup && c.backupCloser != nil {
		if err := c.backupCloser.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close backup file: %w", err))
		}
	}

	if err := c.DB.Close(); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("close database connection: %w", err))
	}

	return closeErr
}

// configureBackups use litestream to restore and stream backups to S3-like
// storages. Follow the examples and official documentations if you have any troubles.
// https://litestream.io/getting-started
// https://github.com/benbjohnson/litestream-library-example/blob/main/main.go
func (c *Conn) configureBackups() error {
	lsdb := litestream.NewDB(c.path)
	lsdb.Logger = c.logger

	var rc litestream.ReplicaClient

	c.logger.Debug("Creating replica client")

	switch c.backupTo {
	case S3:
		client := s3.NewReplicaClient()
		client.AccessKeyID = c.backupS3.AccessKeyID
		client.SecretAccessKey = c.backupS3.SecretAccessKey
		client.Bucket = c.backupS3.Bucket
		client.Endpoint = c.backupS3.Endpoint
		client.Region = c.backupS3.Region
		c.backupRestoreTimeout = c.backupS3.RestoreTimeout

		rc = client

		c.logger.Debug("Replica client has been configured to backup to S3",
			slog.String("bucket", c.backupS3.Bucket),
			slog.String("endpoint", c.backupS3.Endpoint),
			slog.String("region", c.backupS3.Region),
		)

	case File:
		rc = file.NewReplicaClient(c.backupFile.Path)
		c.backupRestoreTimeout = c.backupFile.RestoreTimeout

		c.logger.Debug("Replica client has been configured to backup to file",
			slog.String("path", c.backupFile.Path),
		)

	default:
		return fmt.Errorf("unknown backup destination type: %q", c.backupTo)
	}

	lsr := litestream.NewReplica(lsdb, "main")
	lsr.Client = rc

	lsdb.Replicas = append(lsdb.Replicas, lsr)

	c.logger.Debug("Replica has been attached to litestream")

	ctx, cancel := context.WithTimeout(context.Background(), c.backupRestoreTimeout)
	defer cancel()

	if err := c.restoreBackup(ctx, lsr); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	if err := lsdb.Open(); err != nil {
		return fmt.Errorf("open database for replication: %w", err)
	}

	c.backupCloser = lsdb

	return nil
}

func (c *Conn) restoreBackup(ctx context.Context, replica *litestream.Replica) error {
	if _, err := os.Stat(replica.DB().Path()); err == nil {
		c.logger.Debug("Database file already exists, skipping restore")

		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("get database file stats: %w", err)
	}

	// Configure restore to write out to DSN path.
	opt := litestream.NewRestoreOptions()
	opt.OutputPath = replica.DB().Path()

	// Determine the latest generation to restore from.
	gen, updatedAt, restoreErr := replica.CalcRestoreTarget(ctx, opt)
	if restoreErr != nil {
		return fmt.Errorf("calculate restore target: %w", restoreErr)
	}

	opt.Generation = gen

	// Only restore if there is a generation available on the replica.
	// Otherwise, we'll let the application create a new database.
	if opt.Generation == "" {
		c.logger.Debug("No generation found, creating new database")

		return nil
	}

	c.logger.Debug("Restoring replica for generation",
		slog.String("generation", opt.Generation),
		slog.Time("updatedAt", updatedAt),
	)

	if err := replica.Restore(ctx, opt); err != nil {
		return err
	}

	c.logger.Debug("Restore completed successfully")

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
