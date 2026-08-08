# Tyrako

Production-grade REST API for **TrueMoney Wallet gift redemption**, built with Go. The service exposes authenticated redemption endpoints, an admin key-management API, and all-time success statistics — engineered for performance, horizontal scalability, and defense-in-depth security.

## Features

- **TrueMoney redemption API** — redeem gift codes or gift links via `GET` / `POST` with transparent passthrough of the upstream response.
- **API key authentication** — scoped (`tw`, `admin`), ranked (member / partner / admin), with SHA-256 hashed storage and constant-time verification.
- **Full key lifecycle** — create, list, revoke, and rotate keys; optional expiration; per-key usage tracking.
- **Rate limiting** — fixed-window limits per key tier and per client IP, backed by in-memory or shared Redis state for multi-instance deployments.
- **Statistics** — all-time totals (amount, count, errors) with per-channel breakdown and reference-based deduplication.
- **Concurrency safety** — concurrent duplicate redemptions are coalesced via `singleflight`; nothing is cached across requests, so every retry is re-verified upstream.
- **Security hardening** — request ID tracing, structured logging with path redaction, CORS, security headers, panic recovery, and strict input validation.
- **Zero-downtime operations** — embedded SQL migrations applied atomically at startup, graceful shutdown, multi-stage non-root Docker image.

## Tech Stack

| Layer         | Technology                                  |
| ------------- | ------------------------------------------- |
| Language      | Go 1.26                                     |
| HTTP Router   | chi v5                                      |
| Database      | PostgreSQL 17 (pgx v5, embedded migrations) |
| Rate Limiting | Redis 7 (shared state across instances)     |
| Deploy        | Docker, Docker Compose, Caddy               |

## Quick Start

```bash
# 1. Start PostgreSQL and create the database
psql -U postgres -c "CREATE ROLE tyrako LOGIN PASSWORD 'tyrako';"
psql -U postgres -c "CREATE DATABASE tyrako OWNER tyrako;"

# 2. Configure environment (copy and adjust)
cp .env.example .env

# 3. Run — migrations are embedded and applied automatically at startup
make run
```

See [Getting Started](docs/getting-started.md) for details, including the bootstrap API key.

## Documentation

The full documentation lives in [`docs/`](docs/index.md):

- [Getting Started](docs/getting-started.md) — requirements, setup, bootstrap key
- [Configuration](docs/configuration.md) — environment variable reference
- [API Reference](docs/api-reference.md) — endpoints, authentication, errors
- [Architecture](docs/architecture.md) — system design and project layout
- [Security](docs/security.md) — threat model, rate limiting, validation
- [Testing](docs/testing.md) — test commands and integration setup
- [Deployment](docs/deployment.md) — production stack, scaling, operations
- [Roadmap](docs/roadmap.md) — implemented and planned work

## Development

```bash
make run               # start the API locally
make test              # unit tests with race detector + coverage
make lint              # golangci-lint
make test-integration  # PostgreSQL + Redis integration tests
```

## License

Released under the [MIT License](LICENSE). Copyright (c) 2026 **ByteInDev** and **Rgeirkou**.
