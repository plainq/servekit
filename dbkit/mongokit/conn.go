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

	return &Conn{Client: client}, nil
}

func (c *Conn) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Disconnect(ctx)
}

// Health implements the health.Checker interface for MongoDB connection.
func (c *Conn) Health(ctx context.Context) error {
	prefs, err := readpref.New(readpref.PrimaryPreferredMode)
	if err != nil {
		return fmt.Errorf("mongo: create read preference")
	}

	if err := c.Client.Ping(ctx, prefs); err != nil {
		return fmt.Errorf("mongo: ping database: %w", err)
	}

	return nil
}
