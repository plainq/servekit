# The `servkit` helps to build APIs

Is a set of well-tested (or not well 🤭) reusable components to speedup day-to-day development of HTTP and gRPC APIs.

**⚠️ Under Construction:** There could be breaking changes until v1.0.0.

## Docs

The code should be very straightforward to read, but you always can check the [go docs](https://pkg.go.dev/github.com/plainq/servekit).

## Packages

- `authkit` - Authentication and authorization utilities for securing your API endpoints
- `ctxkit` - Context management utilities and helpers for request context handling
- `dbkit` - Database utilities and helpers for working with various databases (PostgreSQL, SQLite)
- `errkit` - Error handling utilities and custom error types for better error management
- `grpckit` - gRPC server utilities and middleware for building gRPC services
- `httpkit` - HTTP server utilities, middleware, and helpers for building HTTP APIs
- `idkit` - ID generation utilities using ULID for unique identifier generation
- `logkit` - Logging utilities and structured logging helpers
- `mailkit` - Email sending utilities and templates for handling email communications
- `respond` - Response formatting utilities for consistent API responses
- `retry` - Retry mechanisms and backoff strategies for handling transient failures
- `slackkit` - Slack integration utilities for sending notifications and messages
- `tern` - Ternary operator

## On the shoulders of giants

- [github.com/VictoriaMetrics/metric](https://github.com/VictoriaMetrics/metrics)
- [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
- [github.com/benbjohnson/litestream](https://github.com/benbjohnson/litestream)
- [github.com/go-chi/chi](https://github.com/go-chi/chi)
- [github.com/jackc/pgx/v5](https://github.com/jackc/pgx/v5)
- [github.com/jackc/tern/v2](https://github.com/jackc/tern/v2)
- [github.com/maxatome/go-testdeep](https://github.com/maxatome/go-testdeep)
- [github.com/oklog/ulid/v2](https://github.com/oklog/ulid/v2)
- [github.com/redis/go-redis/v9](https://github.com/redis/go-redis/v9)

## Contributing guidelines

You can find [our guidelines here](CONTRIBUTING.md).

## License

[MIT License](LICENSE.md)
