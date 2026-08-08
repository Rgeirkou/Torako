# Deployment

## Production Stack

The production deployment ships as Docker Compose: **API + PostgreSQL 17 + Redis 7 + Caddy** (automatic HTTPS via Let's Encrypt).

```text
Internet ──► Cloudflare Tunnel (TLS, api.example.com)
                 └─► api (:8080 on loopback, non-root)
                       ├─► PostgreSQL 17 (keys, stats, migrations auto-applied)
                       └─► Redis 7 (rate limit buckets, shared across replicas)
```

## Deploy on Ubuntu

```bash
# 1. Install Docker Engine + Compose plugin + make
sudo apt update && sudo apt install -y docker.io docker-compose-plugin make
sudo systemctl enable --now docker

# 2. Copy the project to the server (this project is not published to a remote repo)
#    scp -r . user@server:/srv/tyrako

# 3. Configure and start
cd /srv/tyrako
cp .env.production.example .env.production
nano .env.production   # set DOMAIN, POSTGRES_PASSWORD, REDIS_PASSWORD,
                       # BOOTSTRAP_API_KEY, ALLOW_ORIGINS
make deploy-up
```

Point the domain's DNS A/AAAA record to the server **before** starting — Caddy provisions and renews the Let's Encrypt certificate automatically.

`APP_ENV=production` refuses to start without `BOOTSTRAP_API_KEY` — the seed key must be set explicitly in production so it is never lost.

## Scaling Horizontally

Run multiple `api` replicas behind the load balancer:

- PostgreSQL persists keys and stats; Redis keeps rate limiting consistent across replicas.
- Concurrent duplicate redemptions are coalesced per instance (`singleflight`), and nothing is cached across requests — replaying a request always re-verifies against the upstream, so financial responses are never replayed from cache.
- Migrations are safe under concurrency: they are applied under a PostgreSQL advisory lock.

## Operations

```bash
make deploy-logs    # follow logs (last 100 lines)
make deploy-down    # stop the stack
docker compose -f deployments/docker-compose.prod.yml ps
docker compose -f deployments/docker-compose.prod.yml logs -f api
```

## Local Docker Development

```bash
cp .env.example .env
make docker-up      # docker compose -f deployments/docker-compose.yml up -d --build
```
