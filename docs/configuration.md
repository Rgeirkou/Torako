# Configuration

All configuration is read from environment variables, optionally loaded from a `.env` file in the working directory at startup.

## Reference

| Variable                 | Default                                    | Description                                 |
| ------------------------ | ------------------------------------------ | ------------------------------------------- |
| `ADDR`                   | `:8080`                                    | HTTP listen address                         |
| `APP_ENV`                | `development`                              | Environment name (`development` / `production`) |
| `DATABASE_URL`           | *(empty — in-memory fallback)*             | PostgreSQL connection string                |
| `ALLOW_ORIGINS`          | *(empty)*                                  | CORS allowed origins, comma-separated (`*` allowed — auth uses headers, not cookies) |
| `READ_TIMEOUT`           | `15s`                                      | Server read timeout                         |
| `WRITE_TIMEOUT`          | `35s`                                      | Server write timeout                        |
| `IDLE_TIMEOUT`           | `60s`                                      | Server idle timeout                         |
| `SHUTDOWN_TIMEOUT`       | `10s`                                      | Graceful shutdown timeout                   |
| `TRUEMONEY_API_BASE_URL` | `https://gift.truemoney.com`               | TrueMoney upstream base URL                 |
| `TRUEMONEY_API_TIMEOUT`  | `30s`                                      | TrueMoney client timeout                    |
| `BOOTSTRAP_API_KEY`      | *(empty)*                                  | Seed API key (rank `admin`, scopes `tw, admin`) |
| `RATE_LIMIT_ENABLED`     | `true`                                     | Enable per-key rate limiting                |
| `RATE_LIMIT_MEMBER_MAX`  | `60`                                       | Requests per window for rank `member`       |
| `RATE_LIMIT_PARTNER_MAX` | `1000`                                     | Requests per window for rank `partner`      |
| `RATE_LIMIT_WINDOW`      | `1m`                                       | Rate limit window (both tiers)              |
| `RATE_LIMIT_IP_ENABLED`  | `true`                                     | Enable per-IP limiting (applied before auth)|
| `RATE_LIMIT_IP_MAX`      | `1000`                                     | Max requests per IP per window              |
| `RATE_LIMIT_IP_WINDOW`   | `1m`                                       | Per-IP rate limit window                    |
| `TRUST_PROXY_HEADERS`    | `false`                                    | Read the client IP from `X-Forwarded-For` — enable only behind a trusted reverse proxy |
| `REDIS_URL`              | *(empty — in-memory limiter)*              | Redis URL for shared rate limiting          |

## Validation Rules

The service rejects invalid configurations at startup rather than failing at runtime:

- Rate limit maxima must be at least 1 when their limiter is enabled.
- Rate limit windows must be positive when enabled.
- `APP_ENV=production` requires `BOOTSTRAP_API_KEY`.

## Templates

- Development: `.env.example`
- Production: `.env.production.example`

Production settings are loaded by the Docker Compose stack — see [Deployment](deployment.md).
