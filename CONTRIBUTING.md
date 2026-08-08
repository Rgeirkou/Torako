# Contributing to Tyrako

Thanks for taking the time to contribute! This guide covers how to report
issues, propose changes, and get your pull requests merged.

## Reporting bugs

- Search the [issues](https://github.com/Rgeirkou/Torako/issues) first — the
  bug may already be reported.
- Open a new issue with a clear title and include:
  - What you did, what you expected, and what actually happened.
  - The request/response payloads (redact secrets and API keys).
  - Environment details: Go version, OS, Docker/PostgreSQL/Redis versions.

## Suggesting features

Open an issue with the `enhancement` label. Describe the problem you are
solving and a rough sketch of the API surface you have in mind, so it can be
discussed before implementation.

## Development setup

```sh
# Requirements: Go 1.26, Docker (for PostgreSQL 17 + Redis 7)
git clone https://github.com/Rgeirkou/Torako.git
cp .env.example .env   # adjust as needed
make docker-up         # start db + redis
make test-integration  # integration tests against the containers
make run               # start the API
```

## Code style

- Run `make lint` (golangci-lint v2) and `make vet` before committing.
- Run `make test` — tests must pass, including `-race`.
- Follow existing conventions: chi handlers in `internal/handler`, services in
  `internal/service`, repositories in `internal/repository`.
- Keep the OpenAPI spec (`api/openapi.yaml`) in sync with any API change.
- Migration changes go in `internal/repository/postgres/migrations` and must
  be additive — never edit an already-applied migration.
- No comments unless they explain *why*; let the code speak for itself.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add health endpoint
fix: return 409 on duplicate key
docs: expand deployment guide
ci: split lint and unit jobs
```

## Pull requests

- Base your branch on `main` and keep it focused on one concern.
- Include tests for new behavior and update docs when user-facing.
- Keep the PR description short: what changed, why, and how it was verified.
- The CI workflow runs unit tests, lint, integration tests, and a Docker
  build on every PR — all checks must pass.
- Squash-merge is preferred; the PR title becomes the commit message.

## License

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE).
