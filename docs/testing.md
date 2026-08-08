# Testing

## Commands

```bash
make test              # Unit tests with race detector + coverage
make test-integration  # PostgreSQL + Redis integration tests (needs running services)
make vet               # go vet
make lint              # golangci-lint
```

## Integration Tests

Integration tests cover the PostgreSQL stores and the Redis rate limiter. They read:

- `TEST_DATABASE_URL` — PostgreSQL connection string
- `TEST_REDIS_URL` — Redis connection string

They **skip automatically** when the variables are not set, so the unit suite always runs cleanly.

`make test-integration` provides sensible local defaults pointing at `localhost` when the variables are unset.

## Race Detector

The full suite runs in CI with the race detector enabled (`make test`).

> **Windows note:** `go test -race` requires a C toolchain. Install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) (or use WSL), or run `go test` without `-race` locally — CI runs the full race suite.

## Coverage

Coverage is reported per package. Key coverage areas include:

- Auth middleware: missing / invalid / revoked / expired / wrong-scope keys
- Rate limiting: member, partner, admin tiers, per-IP budgets, `Retry-After` header
- Coalescing: concurrent duplicates hit the upstream once; sequential requests always hit it again
- Statistics: reference deduplication, error counting, totals aggregation
- Input validation: field-level error details, trusted-host gift link checks
