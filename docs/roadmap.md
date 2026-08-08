# Roadmap

## Implemented

- [x] Foundation: config, server, middleware
- [x] Core API: `/tw` (GET + POST) with upstream passthrough
- [x] API key authentication + full key management (`/keys`)
- [x] PostgreSQL persistence (embedded migrations, auto-applied)
- [x] Rate limiting by rank tier (member / partner / admin) + per-IP, with memory and Redis backends
- [x] Security headers, path-redacted structured logging, strict input validation
- [x] Statistics endpoint (`/stats`) with reference-based deduplication
- [x] CI (unit, lint, integration), OpenAPI 3.0 spec
- [x] Production Docker stack (PostgreSQL, Redis, non-root image) behind Cloudflare Tunnel

## Planned

- [ ] Container image publication (GitHub Container Registry)
- [ ] Blue/green or rolling deployments
- [ ] Observability: metrics, tracing, structured logs to a collector
