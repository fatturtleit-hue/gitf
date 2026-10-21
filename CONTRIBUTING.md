# Contributing to gitf

Thanks for contributing! This document gets you from `git clone` to a merged PR.

## Quick start

```bash
# 1. Install prerequisites: Go 1.25+ and Node.js 22+.

# 2. Clone and enter the repo.
git clone <repo-url>
cd gitf

# 3. Copy environment template.
cp .env.example .env
# Edit .env and set JWT_SECRET to a fixed value (32+ bytes hex or base64).
# Without it, every server restart invalidates all sessions.

# 4. Build everything (frontend + backend).
make build

# 5. Run.
make run
```

## Project layout

- `cmd/server` — HTTP server entrypoint and router wiring.
- `internal/handlers` — HTTP handlers. Depends on interfaces (`UserStore`, `RepositoryStore`).
- `internal/middleware` — Auth, logging, request IDs, security headers.
- `internal/repo` — `RepositoryStore` implementation backed by GORM.
- `internal/auth` — Password hashing, JWT generation/validation.
- `internal/gitutil` — Git repository helpers and cache.
- `internal/models` — GORM models and DTOs.
- `internal/db` — Database connection and migrations.
- `web/` — Vue 3 frontend.
- `internal/web` — Embed target for the compiled frontend.

## Development workflow

1. Create a branch: `git checkout -b feat/<name>` or `fix/<name>`.
2. Make your change. Add or update tests.
3. Run the full local check before pushing:
   ```bash
   make test
   make lint-all
   make build
   ```
4. Open a PR. Fill out the checklist in the PR template.

## Testing

### Go

```bash
go test -race ./...
```

If you add a handler, add a test in the same package using `httptest.NewRecorder`
and the mock store implementations from `internal/handlers/api_test.go`.

### API end-to-end

```bash
# Terminal 1: run the server.
go run ./cmd/server

# Terminal 2: run the black-box suite (146 checks across every endpoint,
# git HTTP protocol, auth schemes, rate limits and negative cases).
node scripts/api_test.mjs
```

The suite creates throwaway `zt-*` users/repos and is safe to re-run. See
[`TESTING.md`](TESTING.md) for what it covers and its known limitations.

### Frontend

```bash
cd web
npm run lint
```

## Code style

- Go: `gofmt`, `goimports`, and `golangci-lint`.
- TypeScript: ESLint + Prettier. Run `npm run format` before committing.
- Error messages are user-facing lowercase sentences without trailing periods.
- Every structured log line should include `request_id` when called from a request context.

## Commit message convention

```
<type>: <short summary>

<body>

Fixes #<issue>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `security`.

## PR review checklist

Before requesting a review, run through [`REVIEW_CHECKLIST.md`](REVIEW_CHECKLIST.md).

## Security issues

If you discover a security vulnerability, do not open a public issue. Email the
maintainers directly or use the repository's security advisory form.
