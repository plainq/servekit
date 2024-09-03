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
type Option func(*mongooptions.ClientOptions)

// WithAppName sets app name to the connection properties.
func WithAppName(name string) Option {
	return func(o *mongooptions.ClientOptions) { o.SetAppName(name) }
}

// WithDirectConnection sets ability to make direct connection.
func WithDirectConnection(direct bool) Option {
	return func(o *mongooptions.ClientOptions) { o.SetDirect(direct) }
}

// WithConnectTimeout specifies a timeout that is used for creating connections to the server. This can be set through
// ApplyURI with the "connectTimeoutMS" (e.g "connectTimeoutMS=30") option. If set to 0, no timeout will be used. The
// default is 30 seconds.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *mongooptions.ClientOptions) { o.SetConnectTimeout(timeout) }
}

// WithServerSelectionTimeout specifies how long the driver will wait to find an available,
// suitable server to execute an operation.
func WithServerSelectionTimeout(timeout time.Duration) Option {
	return func(o *mongooptions.ClientOptions) { o.SetServerSelectionTimeout(timeout) }
}

// Conn wraps the connection to the MongoDB.
type Conn struct{ *mongo.Client }

// New returns a pointer to a new instance of Conn struct.
// Receives variadic Option to configure the MongoDB connection settings.
func New(addr string, options ...Option) (*Conn, error) {
	connOptions := mongooptions.Client()

	for _, option := range options {
		option(connOptions)
	}

	connOptions.ApplyURI(addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, connOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo: connection failed: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("mongo: ping failed: %w", err)
	}

	return &Conn{Client: client}, nil
}

func (c *Conn) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Disconnect(ctx)
}

// Health implements the health.Checker interface for MongoDB connection.
func (c *Conn) Health(ctx context.Context) error {
	if err := c.Client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo: ping database: %w", err)
	}

	return nil
}
