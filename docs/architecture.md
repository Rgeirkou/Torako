# Architecture

## System Overview

```text
        Client
          │
          │ X-API-Key
          ▼
┌─────────────────────────────┐
│  Reverse Proxy / TLS        │   Caddy (Let's Encrypt) or Cloudflare Tunnel
│  - TLS termination          │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│  Tyrako API (Go / chi)      │
│                             │
│  Middleware pipeline:       │
│  RequestID → Logging →      │
│  Recover → CORS → Security  │
│  Headers → Timeout →        │
│  IP rate limit → Auth →     │
│  Key rate limit → Scope     │
│                             │
│  Handlers → Services →      │
│  Providers / Repositories   │
└──────┬──────────────┬───────┘
       │              │
       ▼              ▼
┌─────────────┐  ┌─────────────┐
│ PostgreSQL  │  │    Redis    │
│ keys, stats │  │ rate limits │
│ (migrations │  │ (shared)    │
│  auto-apply)│  └─────────────┘
└─────────────┘
       │
       ▼
┌─────────────────────────────┐
│  TrueMoney upstream (HTTPS) │
└─────────────────────────────┘
```

## Design Principles

- **Stateless API** — run one instance for small deployments, or many replicas behind a load balancer. PostgreSQL keeps keys and stats consistent; Redis keeps rate limiting correct across replicas.
- **No cached financial results** — concurrent duplicate redemptions are coalesced per instance via `singleflight`, but nothing is stored after the call completes. Every later request is re-verified against the upstream, so financial success responses can never be replayed from cache.
- **Embedded migrations** — SQL migrations ship inside the binary and are applied atomically at startup under an advisory lock, so concurrent replicas never apply a migration twice.
- **Graceful degradation** — without PostgreSQL or Redis the service falls back to in-memory stores and limiters; statistics recording is best-effort and never fails a redemption.

## Request Flow

1. Middleware assigns a request ID and logs the request (with sensitive paths redacted).
2. The per-IP rate limiter bounds unauthenticated traffic.
3. Auth middleware looks up the `X-API-Key` hash (constant-time), rejects revoked/expired keys, and applies scope checks.
4. The per-key rate limiter enforces the rank tier.
5. The handler validates input, the service coalesces duplicates and calls the TrueMoney provider.
6. Success is recorded (deduplicated by upstream reference) and the raw upstream response is returned in the `data` envelope.

## Project Structure

```text
├── cmd/api/                  # Entrypoint: config, DB, server wiring, graceful shutdown
├── internal/
│   ├── apikey/               # Key generation + SHA-256 hashing
│   ├── config/               # Environment configuration + validation
│   ├── handler/              # HTTP handlers (tw, keys, stats)
│   ├── middleware/           # Auth, rate limit, CORS, logging, recovery, security headers
│   ├── model/                # DTOs and domain errors
│   ├── money/                # Decimal money handling (baht ↔ cents)
│   ├── provider/truemoney/   # TrueMoney upstream client
│   ├── ratelimit/            # Fixed-window limiter (memory + Redis backends)
│   ├── repository/           # PostgreSQL stores (with in-memory fallback)
│   ├── server/               # HTTP server + router
│   ├── service/              # Business logic (coalescing, stats, keys)
│   └── validator/            # Input validation rules
├── pkg/response/             # Response envelope helpers
├── api/openapi.yaml          # OpenAPI 3.0 specification
├── deployments/              # Dockerfile, docker-compose, Caddyfile
└── Makefile
```

The layering is strict: handlers only parse and respond, services hold business logic, providers wrap upstreams, and repositories abstract persistence. Interfaces at each boundary keep the layers testable in isolation.
