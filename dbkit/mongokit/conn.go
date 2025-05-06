package mongokit

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	mongooptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Option configures the mongo options.ClientOptions.
type Option func(options *ConnOptions)

// ConnOptions represents the configuration options
// for the MongoDB connection Conn.
type ConnOptions struct {
	mongoOptions *mongooptions.ClientOptions
	readPref     *readpref.ReadPref
}

// WithAppName sets app name to the connection properties.
func WithAppName(name string) Option {
	return func(o *ConnOptions) { o.mongoOptions.SetAppName(name) }
}

// WithDirectConnection sets ability to make direct connection.
func WithDirectConnection(direct bool) Option {
	return func(o *ConnOptions) { o.mongoOptions.SetDirect(direct) }
}

// WithConnectTimeout specifies a timeout that is used for creating connections to the server. This can be set through
// ApplyURI with the "connectTimeoutMS" (e.g "connectTimeoutMS=30") option. If set to 0, no timeout will be used. The
// default is 30 seconds.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *ConnOptions) { o.mongoOptions.SetConnectTimeout(timeout) }
}

// WithReadPref sets the Mongo read preference which determines
// which servers are considered suitable for read operations.
func WithReadPref(readPref *readpref.ReadPref) Option {
	return func(o *ConnOptions) { o.readPref = readPref }
}

// Conn wraps the connection to the MongoDB.
type Conn struct{ *mongo.Client }

// New returns a pointer to a new instance of Conn struct.
// Receives variadic Option to configure the MongoDB connection settings.
func New(addr string, options ...Option) (*Conn, error) {
	ctx := context.Background()

	connOptions := ConnOptions{
		mongoOptions: mongooptions.Client(),
		readPref:     readpref.PrimaryPreferred(),
	}

	for _, option := range options {
		option(&connOptions)
	}

	connOptions.mongoOptions.ApplyURI(addr)

	client, err := mongo.Connect(ctx, connOptions.mongoOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo: connection failed: %w", err)
	}

	if err := client.Ping(ctx, connOptions.readPref); err != nil {
		return nil, fmt.Errorf("mongo: ping failed: %w", err)
	}

	return &Conn{Client: client}, nil
}

func (c *Conn) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return c.Disconnect(ctx)
}

// Health implements the health.Checker interface for MongoDB connection.
func (c *Conn) Health(ctx context.Context) error {
	if err := c.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo: ping database: %w", err)
	}

	return nil
}
