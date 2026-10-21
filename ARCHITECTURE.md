# gitf Architecture

This document explains how the major pieces of gitf fit together. It is the
first thing a new engineer should read after `README.md`.

## High-level flow

```
Client (browser / git CLI)
        │
        ▼
   chi Router (gzip for text payloads)
        │
        ├─ Global middleware ───────────┐
        │  RequestID → Logger → Metrics │
        │  → SecurityHeaders → Recoverer │
        │                                │
        ▼                                │
   Route-specific middleware             │
   (Auth / OptionalAuth / RateLimit)     │
        │                                │
        ▼                                │
   Handlers ────────────────────────────┘
   AuthHandler | APIHandler | GitHTTPHandler | HealthHandler
        │
        ├─ Auth handlers → UserStore (interface) → GORM
        ├─ API handlers  → RepositoryStore (interface) → GORM
        ├─ Git handlers  → git subprocess + RepositoryStore
        │                  └─ post-receive hook → /internal/post-receive → webhook dispatcher
        └─ Health        → sql.DB.Ping
```

## Key design decisions

1. **Interfaces at the handler boundary.**
   - [`handlers.UserStore`](internal/handlers/userstore.go) abstracts user persistence.
   - [`repo.RepositoryStore`](internal/repo/interface.go) abstracts repository persistence and access control.
   - Handlers and auth middleware depend on these interfaces, never on `*gorm.DB`. This makes unit tests fast and deterministic because we can inject in-memory fakes.

2. **Single source of truth for errors.**
   - [`handlers.respond.go`](internal/handlers/respond.go) owns `writeJSON`, `writeAPIError`, and `writeServerError`.
   - The wire format is always `{ "code": "...", "message": "..." }`.
   - Every application log line carries `request_id` so requests can be traced end-to-end.

3. **CSP is strict.**
   - `script-src 'self'` — no `'unsafe-inline'`. Vue emits hashed external chunks, so inline scripts are unnecessary.
   - Untrusted content (repository READMEs and rendered blobs) is additionally sanitized on the frontend with DOMPurify.

4. **Git operations are delegated to the real `git` binary.**
   - `git upload-pack` / `git receive-pack` are spawned with `exec.CommandContext` so they respect request cancellation.
   - Access control (`CanRead` / `CanWrite`) happens before the subprocess is started.
   - `gitutil.RepoPath` always returns an **absolute** path: merge/diff run
     `git -C <repo>` and use repo paths as fetch remotes — a relative path
     would resolve against the bare repo's own directory.

5. **Never `Save()` a model with partially populated associations.**
   - `models.Repository` and friends declare GORM associations (`Owner`,
     `ForkedFrom`, …) that lookups often fill in only partially. A plain
     `db.Save()` upserts associations and rewrites the foreign key to the
     association's zero key — this has silently detached repositories from
     their owners before.
   - Convention: persistence helpers use `db.Omit(clause.Associations).Save()`
     and mutate scalar columns only.

6. **Webhooks are dispatched from a post-receive hook.**
   - After a successful push, the repo's `post-receive` hook posts the
     `old new ref` lines to the loopback-only `/internal/post-receive`, which
     enqueues deliveries for the repo's active webhooks.
   - Hooks are (re)installed on repo creation and swept at startup
     (`gitutil.SweepInstallHooks`), so repos created by older versions are
     repaired automatically.
   - Branch protection enforcement lives in the `pre-receive` hook +
     `/internal/pre-receive` (active when `HOOK_SECRET` is configured).

## Security model

- Authentication: JWT **or personal access token** (`Bearer gitf_…`) for the
  API, Basic Auth (account password) for git-over-HTTP. The same `UserLookup`
  interface serves both.
- Authorization: only repository owners (and collaborators with the right
  role) can read private repos and push. Org team management requires org
  owner/admin.
- Public repos are readable by anonymous callers.
- Rate limiting is IP-based via `httprate`.
- `ENABLE_PPROF` must be set explicitly; it is off by default.
- `HEALTH_LEAK_DETAILS` must be set explicitly to include DB error details in
  readiness responses.

## Adding a new endpoint

1. Define the request/response types in the handler file.
2. Call `requireUser` for authenticated endpoints, or use `UserFromContext` for optional auth.
3. Use `decodeJSON` for request bodies; it enforces `MaxBodyBytes` and rejects unknown fields.
4. Return errors via `writeAPIError` or `writeServerError`; never write raw strings.
5. Add a unit test in the corresponding `*_test.go` using `httptest.NewRecorder`.
6. If the endpoint mutates a model, persist through the store's `Update`
   (`Omit(clause.Associations)`) — never a bare `Save()` on a loaded struct.
7. Cover the endpoint in `scripts/api_test.mjs` so the end-to-end suite keeps
   the whole API surface honest.

## Adding a new migration

Models use GORM `AutoMigrate` on startup. If you change [`models/`](internal/models/):

1. Update the struct.
2. For destructive changes, add a manual migration in [`internal/db/db.go`](internal/db/db.go).
3. Test against both SQLite and PostgreSQL if the change touches SQL beyond GORM's portable subset.

## Conventions

- Go 1.25+, built with `CGO_ENABLED=0`.
- Tests run against an in-memory SQLite database using `gorm.io/driver/sqlite` + `modernc.org/sqlite` (no CGO).
- Frontend: Vue 3 + Element Plus + Vite, composition API, `script setup`.
