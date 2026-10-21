package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/issues"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PRHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
	Issues issues.Store
	DB     *gorm.DB
}

func NewPRHandler(db *gorm.DB, cfg *config.Config, repos repo.RepositoryStore) *PRHandler {
	return &PRHandler{DB: db, Config: cfg, Repos: repos, Issues: issues.NewService(db)}
}

type CreatePRRequest struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	HeadBranch  string `json:"head_branch"`
	BaseBranch  string `json:"base_branch"`
	HeadRepoID  uint   `json:"head_repo_id"`
}

func (h *PRHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }

	var req CreatePRRequest
	if !decodeJSON(w, r, &req) { return }
	if req.Title == "" || req.HeadBranch == "" || req.BaseBranch == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "title, head_branch, base_branch are required")
		return
	}
	// Branch names are later passed to git subprocess args (merge/diff) —
	// reject flag-like names up front.
	if !gitutil.ValidateRef(req.HeadBranch) || !gitutil.ValidateRef(req.BaseBranch) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid branch name")
		return
	}
	if req.HeadRepoID == 0 {
		req.HeadRepoID = baseRepo.ID
	}

	// Create the issue first (IsPR=true).
	issue, err := h.Issues.Create(baseRepo.ID, user.ID, req.Title, req.Body, nil, nil, nil)
	if err != nil {
		writeServerError(w, r, "create pr issue", err)
		return
	}

	pr := models.PullRequest{
		RepoID:     baseRepo.ID,
		IssueID:    issue.ID,
		HeadRepoID: req.HeadRepoID,
		HeadBranch: req.HeadBranch,
		BaseBranch: req.BaseBranch,
	}
	if err := h.DB.Create(&pr).Error; err != nil {
		writeServerError(w, r, "create pr", err)
		return
	}
	issue.IsPR = true
	issue.PRID = &pr.ID
	h.Issues.Update(issue)

	writeJSON(w, r, http.StatusCreated, map[string]any{
		"pr":    pr.DTO(),
		"issue": issue.DTO(),
	})
}

func (h *PRHandler) ListPRs(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }

	status := r.URL.Query().Get("status")
	page := atoiOrZero(r.URL.Query().Get("page"))
	perPage := atoiOrZero(r.URL.Query().Get("per_page"))
	page, perPage = clampPage(page, perPage)
	offset := (page - 1) * perPage

	query := h.DB.Where("repo_id = ?", repository.ID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Model(&models.PullRequest{}).Count(&total)

	var prs []models.PullRequest
	if err := query.Order("updated_at DESC").Limit(perPage).Offset(offset).Find(&prs).Error; err != nil {
		writeServerError(w, r, "list prs", err)
		return
	}

	// Build response with linked issue data.
	type PRItem struct {
		PR    models.PullRequestDTO `json:"pr"`
		Issue *models.IssueDTO      `json:"issue"`
	}
	items := make([]PRItem, len(prs))
	for i, pr := range prs {
		var issue models.Issue
		if err := h.DB.Preload("Author").First(&issue, pr.IssueID).Error; err == nil {
			dto := issue.DTO()
			items[i] = PRItem{PR: pr.DTO(), Issue: &dto}
		} else {
			items[i] = PRItem{PR: pr.DTO()}
		}
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": NewPagination(page, perPage, int(total)),
	})
}

func (h *PRHandler) GetPR(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, issue, ok := h.resolvePR(w, r, repository.ID, number)
	if !ok { return }
	writeJSON(w, r, http.StatusOK, map[string]any{
		"pr": pr.DTO(), "issue": issue.DTO(),
	})
}

func (h *PRHandler) GetPRDiff(w http.ResponseWriter, r *http.Request) {
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, _, ok := h.resolvePR(w, r, baseRepo.ID, number)
	if !ok { return }

	// Compute diff between base and head.
	headRepo := baseRepo
	if pr.HeadRepoID != baseRepo.ID {
		h.DB.First(&headRepo, pr.HeadRepoID)
	}
	basePath := gitutil.RepoPath(h.Config, baseRepo.Owner.Username, baseRepo.Name)
	headPath := gitutil.RepoPath(h.Config, headRepo.Owner.Username, headRepo.Name)

	// Fetch head branch into base repo for diff computation.
	exec.Command("git", "-C", basePath, "fetch", headPath, pr.HeadBranch+":"+pr.HeadBranch).Run()
	diffs, err := gitutil.Diff(h.Config, baseRepo.Owner.Username, baseRepo.Name, pr.BaseBranch, "FETCH_HEAD")
	if err != nil {
		writeServerError(w, r, "compute diff", err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"diffs": diffs})
}

func (h *PRHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, issue, ok := h.resolvePR(w, r, baseRepo.ID, number)
	if !ok { return }
	if pr.Status != "open" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "PR is not open")
		return
	}

	// Check branch protection for the base branch.
	var protections []models.BranchProtection
	h.DB.Where("repo_id = ? AND pattern = ?", baseRepo.ID, pr.BaseBranch).Find(&protections)
	for _, bp := range protections {
		if bp.RequireApprovals > 0 {
			var count int64
			h.DB.Model(&models.Review{}).Where("pr_id = ? AND status = ?", pr.ID, "approved").Count(&count)
			if int(count) < bp.RequireApprovals {
				writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, fmt.Sprintf("需要 %d 个审批，当前 %d 个", bp.RequireApprovals, count))
				return
			}
		}
	}

	var req struct{ Strategy *string `json:"strategy"` }
	if !decodeJSON(w, r, &req) { return }
	strategy := "merge"
	if req.Strategy != nil { strategy = *req.Strategy }

	mergeSHA, err := gitutil.Merge(
		gitutil.RepoPath(h.Config, baseRepo.Owner.Username, baseRepo.Name),
		gitutil.RepoPath(h.Config, baseRepo.Owner.Username, baseRepo.Name),
		pr.HeadBranch, pr.BaseBranch, strategy,
	)
	if err != nil {
		writeServerError(w, r, "merge", err)
		return
	}
	sha := strings.TrimSpace(mergeSHA)
	pr.Status = "merged"
	pr.MergeCommitSHA = &sha
	pr.MergeStrategy = &strategy
	pr.MergedByID = &user.ID
	now := time.Now()
	pr.MergedAt = &now
	h.DB.Omit(clause.Associations).Save(pr)
	issue.Status = "closed"
	now2 := time.Now()
	issue.ClosedAt = &now2
	h.Issues.Update(issue)

	writeJSON(w, r, http.StatusOK, map[string]any{
		"pr": pr.DTO(), "merge_sha": sha,
	})
}

func (h *PRHandler) ClosePR(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok { return }
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, issue, ok := h.resolvePR(w, r, baseRepo.ID, number)
	if !ok { return }
	pr.Status = "closed"
	h.DB.Omit(clause.Associations).Save(pr)
	issue.Status = "closed"
	now := time.Now()
	issue.ClosedAt = &now
	h.Issues.Update(issue)
	writeJSON(w, r, http.StatusOK, pr.DTO())
}

func (h *PRHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, _, ok := h.resolvePR(w, r, baseRepo.ID, number)
	if !ok { return }

	var req struct {
		Body      string `json:"body"`
		Status    string `json:"status"`
		CommitSHA string `json:"commit_sha"`
	}
	if !decodeJSON(w, r, &req) { return }
	if req.Status == "" { req.Status = "commented" }

	review := models.Review{
		PRID:       pr.ID,
		ReviewerID: user.ID,
		Body:       req.Body,
		Status:     req.Status,
		CommitSHA:  req.CommitSHA,
	}
	if err := h.DB.Create(&review).Error; err != nil {
		writeServerError(w, r, "create review", err)
		return
	}
	review.Reviewer = *user
	writeJSON(w, r, http.StatusCreated, review.DTO())
}

func (h *PRHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	baseRepo, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	number := atoiOrZero(chi.URLParam(r, "number"))
	pr, _, ok := h.resolvePR(w, r, baseRepo.ID, number)
	if !ok { return }

	var reviews []models.Review
	if err := h.DB.Preload("Reviewer").Where("pr_id = ?", pr.ID).
		Order("created_at DESC").Find(&reviews).Error; err != nil {
		writeServerError(w, r, "list reviews", err)
		return
	}
	items := make([]models.ReviewDTO, len(reviews))
	for i, rv := range reviews {
		items[i] = rv.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *PRHandler) resolvePR(w http.ResponseWriter, r *http.Request, repoID uint, number int) (*models.PullRequest, *models.Issue, bool) {
	var pr models.PullRequest
	// Both pull_requests and issues have a repo_id column — the bare column
	// name is ambiguous once the join is applied, so it must be qualified.
	if err := h.DB.Where("pull_requests.repo_id = ?", repoID).
		Joins("JOIN issues ON issues.id = pull_requests.issue_id AND issues.number = ?", number).
		First(&pr).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "PR not found")
		return nil, nil, false
	}
	var issue models.Issue
	if err := h.DB.Preload("Author").First(&issue, pr.IssueID).Error; err != nil {
		writeServerError(w, r, "load issue", err)
		return nil, nil, false
	}
	return &pr, &issue, true
}

func repoFromPRHandler(store repo.RepositoryStore, w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	h := &APIHandler{Repos: store}
	return h.repoFromRequestWithRead(w, r)
}
