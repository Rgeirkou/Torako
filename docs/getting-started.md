# Getting Started

This guide walks through setting up Tyrako locally. See [Configuration](configuration.md) for the full environment reference and [Deployment](deployment.md) for production.

## Requirements

- Go 1.26+
- PostgreSQL 17 (optional — the service runs with an in-memory fallback when `DATABASE_URL` is empty)
- Redis 7 (optional — falls back to in-memory rate limiting)
- make (on Windows, use WSL or a GNU make port)

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

## Bootstrap API Key

The key store starts empty. On first boot the service registers a seed key:

```env
BOOTSTRAP_API_KEY=your-fixed-seed-key
```

- If `BOOTSTRAP_API_KEY` is set, the key is registered idempotently (skipped once it exists) with rank `admin` and scopes `tw, admin`.
- If left empty in development, a random key is generated and printed in the startup log.
- In `APP_ENV=production` the variable is **required** — the server refuses to start without it, so the seed key is never lost.

## Verify It Works

```bash
# Public endpoint — no API key required
curl "http://localhost:8080/stats"

# Authenticated redemption (use the key from the bootstrap step)
curl -X POST "http://localhost:8080/tw" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -d '{"phone":"0812345678","gift":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'
```

## Next Steps

- Read the [API Reference](api-reference.md) to integrate clients.
- Read the [Configuration](configuration.md) reference before changing defaults.
- Read [Deployment](deployment.md) when you are ready to ship.
