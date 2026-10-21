package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"gitf/internal/config"
	"gitf/internal/models"
	"gitf/internal/repo"
	"gitf/internal/webhooks"

	"gorm.io/gorm"
)

// GitInternalHandler serves internal Git hook endpoints authenticated by
// a shared hook secret configured via HOOK_SECRET.
type GitInternalHandler struct {
	Repos      repo.RepositoryStore
	DB         *gorm.DB
	Config     *config.Config
	Dispatcher *webhooks.Dispatcher
}

func NewGitInternalHandler(db *gorm.DB, repos repo.RepositoryStore, cfg *config.Config, dispatcher *webhooks.Dispatcher) *GitInternalHandler {
	return &GitInternalHandler{DB: db, Repos: repos, Config: cfg, Dispatcher: dispatcher}
}

func (h *GitInternalHandler) authorizeHook(w http.ResponseWriter, r *http.Request) bool {
	// Same backward-compat rule as PreReceive: without a configured secret,
	// accept requests (hooks are loopback-only).
	if h.Config != nil && h.Config.HookSecret != "" {
		if r.Header.Get("X-Gitf-Hook-Token") != h.Config.HookSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
	}
	return true
}

// PostReceive enqueues webhook deliveries for refs updated by a push. The
// post-receive hook posts the standard "old new ref" lines here.
func (h *GitInternalHandler) PostReceive(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeHook(w, r) {
		return
	}
	owner := r.Header.Get("X-Gitf-Owner")
	name := r.Header.Get("X-Gitf-Repo")
	if owner == "" || name == "" {
		http.Error(w, "missing owner or repo header", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	if h.Dispatcher == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	repository, err := h.Repos.GetByOwnerAndName(owner, name)
	if err != nil {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	var hooks []models.Webhook
	if err := h.DB.Where("owner_type = ? AND owner_id = ? AND active = ?", "repo", repository.ID, true).
		Find(&hooks).Error; err != nil || len(hooks) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		refName := fields[2]
		// Only branch updates are dispatched as push events today.
		branch := strings.TrimPrefix(refName, "refs/heads/")
		if branch == refName {
			continue
		}
		payload, err := json.Marshal(map[string]any{
			"event":  "push",
			"ref":    refName,
			"branch": branch,
			"before": fields[0],
			"after":  fields[1],
			"repository": map[string]any{
				"owner": owner,
				"name":  name,
			},
		})
		if err != nil {
			continue
		}
		for _, wb := range hooks {
			if !webhookWantsEvent(wb.Events, "push") {
				continue
			}
			h.Dispatcher.Enqueue(wb, "push", string(payload))
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// webhookWantsEvent matches an event against a space-separated events list.
// Empty or "*" subscribes to everything.
func webhookWantsEvent(events, event string) bool {
	events = strings.TrimSpace(events)
	if events == "" || events == "*" {
		return true
	}
	for _, e := range strings.Fields(events) {
		if e == event {
			return true
		}
	}
	return false
}

// PreReceive validates branch-protection rules before accepting a push.
func (h *GitInternalHandler) PreReceive(w http.ResponseWriter, r *http.Request) {
	// Authenticate via shared hook secret. If HOOK_SECRET is not configured,
	// accept all requests for backward compatibility with repos that have
	// per-repo hook tokens created before this feature was added.
	if h.Config != nil && h.Config.HookSecret != "" {
		if r.Header.Get("X-Gitf-Hook-Token") != h.Config.HookSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	owner := r.Header.Get("X-Gitf-Owner")
	name := r.Header.Get("X-Gitf-Repo")
	if owner == "" || name == "" {
		http.Error(w, "missing owner or repo header", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	repoPath := filepath.Join(h.Config.ReposDir(), owner, name+".git")

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		oldHash := fields[0]
		newHash := fields[1]
		refName := fields[2]

		if isZeroHash(newHash) {
			branch := strings.TrimPrefix(refName, "refs/heads/")
			tag := strings.TrimPrefix(refName, "refs/tags/")
			if tag != refName {
				continue
			}
			if protections, err := h.matchingProtectionsForBranch(branch); err == nil {
				for _, bp := range protections {
					if !bp.AllowDelete {
						http.Error(w, "branch '"+branch+"' is protected from deletion", http.StatusForbidden)
						return
					}
				}
			}
			continue
		}
		if isZeroHash(oldHash) {
			continue
		}

		branch := strings.TrimPrefix(refName, "refs/heads/")
		if branch == refName {
			continue
		}

		protections, err := h.matchingProtectionsForBranch(branch)
		if err != nil || len(protections) == 0 {
			continue
		}

		for _, bp := range protections {
			if !bp.AllowForcePush {
				if !isFastForward(repoPath, oldHash, newHash) {
					http.Error(w, "force push to '"+branch+"' is blocked by branch protection", http.StatusForbidden)
					return
				}
			}
			if bp.RequireApprovals > 0 {
				http.Error(w, "push to '"+branch+"' requires approved pull requests", http.StatusForbidden)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func isFastForward(repoPath, oldHash, newHash string) bool {
	cmd := exec.Command("git", "-C", repoPath, "merge-base", "--is-ancestor", oldHash, newHash)
	return cmd.Run() == nil
}

func isZeroHash(hash string) bool {
	for _, c := range hash {
		if c != '0' {
			return false
		}
	}
	return len(hash) > 0
}

func (h *GitInternalHandler) matchingProtectionsForBranch(branch string) ([]models.BranchProtection, error) {
	var rules []models.BranchProtection
	err := h.DB.Where("pattern = ?", branch).Or("pattern = ?", "*").Or("pattern = ?", "**").Find(&rules).Error
	return rules, err
}
