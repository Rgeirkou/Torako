# Security

Tyrako is a public API, so security is designed into every layer: authentication, rate limiting, input validation, transport, and logging.

## Authentication & Key Storage

- Every request (except `GET /stats`) requires a valid `X-API-Key`.
- Keys are stored **only** as SHA-256 hashes; verification uses constant-time comparison to resist timing attacks.
- Access is enforced per **scope** (`tw`, `admin`); revoked, expired, or scope-less keys receive `403`.

## Rate Limiting

Two independent fixed-window limiters:

- **Per API key, by rank** — `member` 60 req/min, `partner` 1000 req/min, `admin` unlimited; applied after authentication.
- **Per client IP** — 1000 req/min (configurable); applied **before** authentication so unauthenticated traffic is also bounded. Behind a trusted reverse proxy, set `TRUST_PROXY_HEADERS=true` to read the real client IP from `X-Forwarded-For`.

Limiters run in-memory per instance, or share Redis state (`REDIS_URL`) across replicas. Exceeded requests return `429` with a `Retry-After` header.

## Input Validation

All input is validated before processing:

- Thai phone numbers — exactly 10 digits, starting with `0`.
- Gift codes — 20–60 alphanumeric characters; gift links must be HTTPS on a trusted `truemoney.com` host (rejects phishing links).
- Strict JSON decoding — unknown fields and malformed payloads are rejected with `400`.
- URL paths are validated with the same rules as JSON bodies (no bypass via path parameters).

## Transport Security

TLS is terminated at the reverse proxy (Caddy with automatic Let's Encrypt certificates, or Cloudflare Tunnel). The API never serves keys over plain HTTP in production. Security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Cache-Control) are set on every response.

## Logging

Structured `slog` logging with request IDs. Sensitive path segments — phone numbers and gift codes — are redacted in logs so credentials never leak to the log stream.

## Operational Guidance

- Never expose API keys in source code, git repositories, logs, screenshots, or public documentation.
- Store keys in a secret manager or environment variables.
- Set `TRUST_PROXY_HEADERS=true` **only** behind a trusted reverse proxy that overwrites `X-Forwarded-For`.
- In production, always set `BOOTSTRAP_API_KEY` explicitly (the server refuses to start without it).
