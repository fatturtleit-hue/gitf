# Testing & Quality

How gitf is tested, and the record of the 2026-10-21 full-stack audit
(comprehensive API test + fixes + optimization).

## Test layers

| Layer | Command | Scope |
|-------|---------|-------|
| Go unit tests | `make test` (`go test -race ./...`) | Handler logic with in-memory fakes, stores, middleware, gitutil, migrations |
| API end-to-end | `node scripts/api_test.mjs` | Every REST endpoint + git HTTP protocol against a **running** server |
| Admin API | covered inside the audit (needs a site-admin user) | `/api/v1/admin/*` CRUD |
| Frontend smoke | manual / browser | Register, login, repo create, file browser, repo list |

## API end-to-end suite

`scripts/api_test.mjs` (Node 22+, zero dependencies) exercises a live server:

- Health, metrics, SPA fallback
- Auth: register / login / refresh / JWT + Basic + **PAT** bearer schemes, negative cases
- Repos: CRUD, visibility (public/private), rename, search, fork
- Git HTTP: anonymous clone, authenticated push/pull/tag, rejected anonymous push
- Content: branches, commits, tree, blob, raw, path-traversal rejection, tags,
  compare, archive, stats
- Collaborators, Issues (+comments, close/reopen), Labels, Milestones, Kanban
- Pull Requests: create, read, diff, review, merge (all strategies), close
- Orgs: members, teams, team repos, leave, delete + authorization checks
- Releases + asset upload, Webhooks (incl. push-triggered delivery), PATs
- Actions runner API, package-registry stubs, input validation, body limits,
  rate limiting, XSS-shaped input rejection

```bash
# terminal 1
./bin/gitf                      # or: go run ./cmd/server

# terminal 2
node scripts/api_test.mjs       # exit code 0 = all green
```

Notes:
- Creates throwaway `zt-*`/`rl-*` users and repos; re-run safe (falls back to
  login when the register rate limit window is full).
- Admin CRUD requires promoting a user to `is_site_admin` in the DB first —
  done once per audit run.

## Audit 2026-10-21 — results

Final state: **unit tests all green; API suite 146/146; admin API 7/7;
frontend smoke pass.**

### Bugs found and fixed

1. **Repo settings update detached the repo from its owner (P0 data loss).**
   `Repository` has GORM `Owner` / `ForkedFrom` associations that lookups
   partially populate; a plain `Save()` rewrote `owner_id` to the association's
   zero key. Every `PATCH /repos/{owner}/{name}` bricked the repo (404
   everywhere, orphaned on disk). All `Save` sites now use
   `Omit(clause.Associations)`.
2. **Fork 500.** Source path built from an un-preloaded `Owner.Username`
   (empty) → `data/repositories/<name>.git` without the owner segment.
3. **All single-PR operations 404 (get/diff/review/merge/close).** `resolvePR`
   joined `issues` with an unqualified `repo_id` → SQL ambiguity error.
4. **PR merge never worked.** (a) Fetch remote was a relative path resolved
   against the bare repo's own dir (`RepoPath` is now absolute); (b) `FETCH_HEAD`
   was written before the worktree existed; (c) bare repos have no `origin`
   remote to push to; (d) merge commits had no git identity. `gitutil.Merge`
   rewritten: worktree first → fetch inside it → merge/squash/rebase → push to
   the bare path directly.
5. **PR diff panicked (500, empty body).** `gitutil.Diff(nil, …)` — nil `Config`
   dereferenced.
6. **Team endpoints had no authorization.** Any authenticated user could
   create/delete teams and manage team members/repos of *any* organization.
   Now requires org owner/admin (or site admin), and verifies the team belongs
   to the `{org}` in the URL.
7. **PATs could not authenticate.** Tokens were issued and listed but the auth
   middleware never consulted the token store. `Bearer gitf_…` now works for
   the API and git-over-HTTP.
8. **Webhooks never fired.** Nothing enqueued deliveries on push (only manual
   redeliver). Added a `post-receive` hook + `/internal/post-receive` endpoint;
   push events are dispatched to active webhooks. Startup re-installs hooks for
   existing repos (`SweepInstallHooks`).
9. **Releases: tag_name was globally unique** (composite index declared on one
   column only), so two repos could not share a tag name. Fixed model + idempotent
   index rebuild in `db.Migrate`.
10. **Workflow dispatch accepted nonexistent workflow IDs** and created orphan
    runs; now validated (404).
11. **Duplicate route registrations** for org member endpoints in the router.
12. **Archive of an invalid ref returned 500**; now 404.
13. **Hooks hardcoded `127.0.0.1:8080`**; now derived from the configured listen
    address.

### Known limitations (not fixed — by design or pending decision)

- **Actions (CI)**: the data model and runner API exist, but there is no entry
  point that *creates* workflows (no YAML scanner, no UI), so the workflows
  list is always empty. Completing or removing this subsystem is a product
  decision.
- **Package registries** (`/v2/`, npm, pypi, maven, nuget, helm) are protocol
  stubs.
- **Webhook events**: only `push` is dispatched; `pull_request` / `issues` /
  `release` events are accepted in config but not emitted.
- **Org-owned repositories** cannot be created through the REST API, so team
  repo binding is only usable for org repos created by other means.
- **Branch protection rules have no UI** — API only (`/repos/.../protection`);
  enforcement additionally requires `HOOK_SECRET` on the server.
- Repos created before hooks carried owner headers were repaired automatically
  at startup by the hook sweep; installs without `HOOK_SECRET` do not get
  pre-receive branch protection enforcement.
- `internal/issues`, `internal/org`, `internal/webhooks` and `internal/models`
  have no Go unit tests yet.

### Optimizations applied (2026-10-21)

- Frontend production sourcemaps disabled: dist **5.5 MB → 1.3 MB**; embedded
  binary **28 MB → 24 MB**.
- gzip (level 5) for HTML/CSS/JS/JSON/SVG/XML responses; binary content types
  (git smart HTTP, archives) are excluded.
- Dead code removed (`middleware.BodyLimiter` was never wired).
- `go mod tidy` and frontend dependency audit: no unused dependencies.
- SPA assets were already content-hashed with `immutable` caching; unchanged.
