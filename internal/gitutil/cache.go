package gitutil

import (
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
)

type repoCacheEntry struct {
	repo    *git.Repository
	expires time.Time
}

var (
	repoCache   = sync.Map{} // key: "owner/name" -> *repoCacheEntry
	repoCacheMu sync.Mutex
)

// getRepo returns a cached or newly opened repository handle.
// The cache provides request-level reuse (within a 30s TTL) so that
// multiple gitutil calls within a single HTTP request share one handle.
func getRepo(owner, name, repoPath string) (*git.Repository, error) {
	key := owner + "/" + name

	// Fast path: check cache without lock.
	if entry, ok := repoCache.Load(key); ok {
		e := entry.(*repoCacheEntry)
		if time.Now().Before(e.expires) {
			return e.repo, nil
		}
	}

	repoCacheMu.Lock()
	defer repoCacheMu.Unlock()

	// Double-check after acquiring lock.
	if entry, ok := repoCache.Load(key); ok {
		e := entry.(*repoCacheEntry)
		if time.Now().Before(e.expires) {
			return e.repo, nil
		}
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	repoCache.Store(key, &repoCacheEntry{
		repo:    repo,
		expires: time.Now().Add(30 * time.Second),
	})
	return repo, nil
}

// InvalidateRepoCache removes a repository from the cache.
// Must be called after repository deletion or recreation.
func InvalidateRepoCache(owner, name string) {
	repoCache.Delete(owner + "/" + name)
}
