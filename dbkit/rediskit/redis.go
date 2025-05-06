package redisconn

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Conn wraps connection with the Redis.
type Conn struct{ *redis.Client }

// Option modifies the redis.Options.
type Option func(o *redis.Options)

// WithClientName sets the client name for the Redis connection.
func WithClientName(name string) Option {
	return func(o *redis.Options) { o.ClientName = name }
}

// WithCredentials sets the credentials for the Redis connection.
func WithCredentials(username, password string) Option {
	return func(o *redis.Options) {
		o.Username = username
		o.Password = password
	}
}

// New returns a pointer to a new instance of the Conn struct.
func New(addr string, options ...Option) (*Conn, error) {
	connOptions := redis.Options{
		Addr: addr,
	}

	for _, option := range options {
		option(&connOptions)
	}

	client := redis.NewClient(&connOptions)

	return &Conn{Client: client}, nil
}

func (c *Conn) HealthCheck(ctx context.Context) error {
	if s := c.Ping(ctx); s.Err() != nil {
		return fmt.Errorf("redis: healthcheck failed: %w", s.Err())
	}

	return nil
}
