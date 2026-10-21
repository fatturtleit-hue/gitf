package handlers

import (
	"net/http"

	"gitf/internal/config"
	"gitf/internal/repo"
)

// AdminHandler provides server-level administration endpoints.
type AdminHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
}

// NewAdminHandler creates an admin handler.
func NewAdminHandler(cfg *config.Config, store repo.RepositoryStore) *AdminHandler {
	return &AdminHandler{Config: cfg, Repos: store}
}

// ServerStats holds overall server statistics.
type ServerStats struct {
	DiskTotal uint64 `json:"disk_total"`
	DiskUsed  uint64 `json:"disk_used"`
	DiskFree  uint64 `json:"disk_free"`
	RepoCount int64  `json:"repo_count"`
	UserCount int64  `json:"user_count"`
}

// GetStats returns server-level statistics.
// The endpoint is available to any authenticated user, but disk usage and
// total user count are only exposed to site administrators.
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	var stats ServerStats

	// Repo count is visible to all authenticated users.
	repoCount, err := h.Repos.CountRepos()
	if err == nil {
		stats.RepoCount = repoCount
	}

	// Disk usage and user count are site-admin only.
	user, ok := UserFromContext(r.Context())
	if ok && user.IsSiteAdmin {
		if diskTotal, diskFree, ok := getDiskStats(h.Config.DataDir); ok {
			stats.DiskTotal = diskTotal
			stats.DiskFree = diskFree
			stats.DiskUsed = diskTotal - diskFree
		}

		userCount, err := h.Repos.CountUsers()
		if err == nil {
			stats.UserCount = userCount
		}
	}

	writeJSON(w, r, http.StatusOK, stats)
}
