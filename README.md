# Tyrako

[![License: MIT](https://img.shields.io/github/license/Rgeirkou/Torako?style=flat-square)](LICENSE)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/dl/)
[![CI](https://img.shields.io/github/actions/workflow/status/Rgeirkou/Torako/ci.yml?style=flat-square&logo=github&logoColor=white)](https://github.com/Rgeirkou/Torako/actions/workflows/ci.yml)
[![Status: Production](https://img.shields.io/badge/status-production-2ea44f?style=flat-square)](docs/roadmap.md)
[![Docs](https://img.shields.io/badge/docs-full%20guide-2f81f7?style=flat-square)](docs/index.md)

Production-grade REST API for **TrueMoney Wallet gift redemption**, built with Go. The service exposes authenticated redemption endpoints, an admin key-management API, and all-time success statistics — engineered for performance, horizontal scalability, and defense-in-depth security.

## Features

| Area | What you get |
|---|---|
| **Redemption API** | Redeem gift codes or gift links via `GET` / `POST` with transparent passthrough of the upstream response |
| **API key auth** | Scoped (`tw`, `admin`), ranked (member / partner / admin), SHA-256 hashed storage and constant-time verification |
| **Key lifecycle** | Create, list, revoke, and rotate keys; optional expiration; per-key usage tracking |
| **Rate limiting** | Fixed-window limits per key tier and per client IP, backed by in-memory or shared Redis state |
| **Statistics** | All-time totals (amount, count, errors) with per-channel breakdown and reference-based deduplication |
| **Concurrency safety** | Duplicate redemptions coalesced via `singleflight`; nothing cached across requests, every retry re-verified upstream |
| **Security hardening** | Request ID tracing, structured logging with path redaction, CORS, security headers, panic recovery, strict input validation |
| **Zero-downtime ops** | Embedded SQL migrations applied atomically at startup, graceful shutdown, multi-stage non-root Docker image |

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP Router | chi v5 |
| Database | PostgreSQL 17 (pgx v5, embedded migrations) |
| Rate Limiting | Redis 7 (shared state across instances) |
| Deploy | Docker, Docker Compose, Caddy |

## Setup

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

## Usage

All endpoints except `GET /stats` require an API key in the `X-API-Key` header:

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/tw` | `tw` | Redeem a TrueMoney gift (JSON body) |
| GET | `/tw/{phone}/{gift}` | `tw` | Redeem a TrueMoney gift (path params) |
| POST | `/keys` | admin | Create an API key |
| GET | `/keys` | admin | List API keys |
| DELETE | `/keys/{id}` | admin | Revoke an API key |
| POST | `/keys/{id}/rotate` | admin | Rotate an API key |
| GET | `/stats` | — | All-time statistics (public) |

```bash
curl -X POST "https://api.example.com/tw" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -d '{
    "phone": "0812345678",
    "gift": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }'
```

Full endpoint reference, error shapes, and status codes in [docs/api-reference.md](docs/api-reference.md).

## Development

```bash
make run               # start the API locally
make test              # unit tests with race detector + coverage
make lint              # golangci-lint
make test-integration  # PostgreSQL + Redis integration tests
```

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

## License

[MIT](LICENSE) © [ByteInDev](https://github.com/Rgeirkou) & Rgeirkou.
