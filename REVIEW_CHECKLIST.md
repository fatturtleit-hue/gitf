# Code Review Checklist

## Correctness
- [ ] Logic handles all edge cases (empty input, null, error paths)
- [ ] Error messages are user-friendly and not leaking internals
- [ ] Authorization is checked before any data access
- [ ] Response status codes are appropriate (200/201/204/400/401/403/404/409/429/500)

## Security
- [ ] No hardcoded credentials, tokens, or secrets
- [ ] Input validated (length, format, range) — see `gitutil.ValidateName`
- [ ] SQL injection protected (all queries use GORM parameterized queries)
- [ ] Path traversal prevented (all file paths go through `RepoPath`)
- [ ] JWT tokens include expiration and are validated per request
- [ ] Rate limiting in place for auth and Git HTTP endpoints

## Performance
- [ ] No N+1 queries (use JOINs where appropriate)
- [ ] Large allocations avoided in hot paths (prefer typed structs over map[string]any)
- [ ] Caching used where appropriate (repo handles, config paths, environment)
- [ ] No blocking operations without timeout (see http.Server timeouts)
- [ ] Response compression enabled (chi middleware)

## Testability
- [ ] New code is covered by tests (target ≥ 60% overall)
- [ ] Dependencies are injectable (interfaces, not concrete types)
- [ ] No global mutable state that would break parallel tests
- [ ] Test files alongside source: `foo_test.go` next to `foo.go`

## Maintainability
- [ ] Naming is clear and consistent with project conventions
- [ ] No magic numbers (use named constants or config values)
- [ ] Functions are single-responsibility (≤ 80 lines guideline)
- [ ] Comments explain "why", not "what"
- [ ] New TODO: `// TODO(username): description - ETA:YYYY-MM-DD`
- [ ] API changes documented in OpenAPI spec if applicable
