package handlers

import (
	"errors"
	"net/http"

	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/models"
	orgstore "gitf/internal/org"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// OrgHandler holds organization HTTP handlers.
type OrgHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
	Orgs   orgstore.Store
}

func NewOrgHandler(db *gorm.DB, cfg *config.Config, repos repo.RepositoryStore) *OrgHandler {
	return &OrgHandler{Config: cfg, Repos: repos, Orgs: orgstore.NewService(db)}
}

// ── Organizations ──────────────────────────────────────

type CreateOrgRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

func (h *OrgHandler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req CreateOrgRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || !gitutil.ValidateName(req.Name) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid organization name")
		return
	}
	org, err := h.Orgs.Create(req.Name, req.DisplayName, req.Description, user.ID)
	if err != nil {
		if errors.Is(err, orgstore.ErrOrgConflict) {
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "organization already exists")
			return
		}
		writeServerError(w, r, "create org", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, org.DTO())
}

type UpdateOrgRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

func (h *OrgHandler) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	// Only org owners/admins or site admins may update settings.
	if !user.IsSiteAdmin {
		member, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
			return
		}
		if member.Role != "owner" && member.Role != "admin" {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "insufficient permissions")
			return
		}
	}
	var req UpdateOrgRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	updated, err := h.Orgs.Update(org.ID, req.DisplayName, req.Description)
	if err != nil {
		if errors.Is(err, orgstore.ErrOrgNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "organization not found")
			return
		}
		if errors.Is(err, orgstore.ErrOrgConflict) {
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "organization display name conflict")
			return
		}
		writeServerError(w, r, "update org", err)
		return
	}
	writeJSON(w, r, http.StatusOK, updated.DTO())
}

func (h *OrgHandler) GetOrg(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "org")
	org, err := h.Orgs.GetByName(name)
	if err != nil {
		if errors.Is(err, orgstore.ErrOrgNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "organization not found")
			return
		}
		writeServerError(w, r, "get org", err)
		return
	}
	writeJSON(w, r, http.StatusOK, org.DTO())
}

func (h *OrgHandler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgs, err := h.Orgs.List(user.ID)
	if err != nil {
		writeServerError(w, r, "list orgs", err)
		return
	}
	items := make([]models.OrganizationDTO, len(orgs))
	for i, o := range orgs {
		items[i] = o.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

// ── Members ────────────────────────────────────────────

func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, name)
	if !ok {
		return
	}
	user, uok := requireUser(w, r)
	if !uok {
		return
	}
	if !user.IsSiteAdmin {
		_, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
			return
		}
	}
	members, err := h.Orgs.ListMembers(org.ID)
	if err != nil {
		writeServerError(w, r, "list members", err)
		return
	}
	items := make([]models.OrgMemberDTO, len(members))
	for i, m := range members {
		items[i] = m.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

type AddMemberRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *OrgHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	if !user.IsSiteAdmin {
		member, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil || (member.Role != "owner" && member.Role != "admin") {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "insufficient permissions")
			return
		}
	}
	var req AddMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Username == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "username required")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "owner" && req.Role != "admin" && req.Role != "member" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid role")
		return
	}
	// Find user by username
	var targetUser models.User
	if err := h.Orgs.(*orgstore.Service).DB.Where("username = ?", req.Username).First(&targetUser).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "user not found")
		return
	}
	member, err := h.Orgs.AddMember(org.ID, targetUser.ID, req.Role)
	if err != nil {
		writeServerError(w, r, "add member", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, member.DTO())
}

func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	userID := atoiOrZero(chi.URLParam(r, "userID"))
	if userID == 0 {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user ID")
		return
	}
	if !user.IsSiteAdmin {
		member, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil || (member.Role != "owner" && member.Role != "admin") {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "insufficient permissions")
			return
		}
	}
	// Prevent removing owners to avoid orphan organizations.
	target, err := h.Orgs.GetMember(org.ID, uint(userID))
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "member not found")
		return
	}
	if target.Role == "owner" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "cannot remove organization owner")
		return
	}
	if err := h.Orgs.RemoveMember(org.ID, uint(userID)); err != nil {
		writeServerError(w, r, "remove member", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type UpdateMemberRequest struct {
	Role string `json:"role"`
}

func (h *OrgHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	userID := atoiOrZero(chi.URLParam(r, "userID"))
	if userID == 0 {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user ID")
		return
	}
	if !user.IsSiteAdmin {
		member, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil || member.Role != "owner" {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "only owner can change roles")
			return
		}
	}
	var req UpdateMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role != "owner" && req.Role != "admin" && req.Role != "member" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid role")
		return
	}
	// Get member by org_id + user_id
	member, err := h.Orgs.GetMember(org.ID, uint(userID))
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "member not found")
		return
	}
	updated, err := h.Orgs.UpdateMember(member.ID, req.Role)
	if err != nil {
		writeServerError(w, r, "update member", err)
		return
	}
	writeJSON(w, r, http.StatusOK, updated.DTO())
}

func (h *OrgHandler) DeleteOrg(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	if !user.IsSiteAdmin {
		member, err := h.Orgs.GetMember(org.ID, user.ID)
		if err != nil || member.Role != "owner" {
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "only owner can delete organization")
			return
		}
	}
	if err := h.Orgs.Delete(org.ID); err != nil {
		writeServerError(w, r, "delete org", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) LeaveOrg(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	member, err := h.Orgs.GetMember(org.ID, user.ID)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "not a member of this organization")
		return
	}
	if member.Role == "owner" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "owner cannot leave; transfer ownership first")
		return
	}
	if err := h.Orgs.RemoveMember(org.ID, user.ID); err != nil {
		writeServerError(w, r, "leave org", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Teams ──────────────────────────────────────────────

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *OrgHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	if !h.requireOrgAdmin(w, r, org) {
		return
	}
	var req CreateTeamRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	team, err := h.Orgs.CreateTeam(org.ID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, orgstore.ErrTeamConflict) {
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "team already exists")
			return
		}
		writeServerError(w, r, "create team", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, team.DTO())
}

func (h *OrgHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return
	}
	teams, err := h.Orgs.ListTeams(org.ID)
	if err != nil {
		writeServerError(w, r, "list teams", err)
		return
	}
	items := make([]models.TeamDTO, len(teams))
	for i, t := range teams {
		items[i] = t.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

// resolveTeamInOrg loads the {teamID} URL param and verifies the team belongs
// to the {org} URL param, then requires the caller to be an org admin.
func (h *OrgHandler) resolveTeamInOrg(w http.ResponseWriter, r *http.Request) (*models.Organization, *models.Team, bool) {
	orgName := chi.URLParam(r, "org")
	org, ok := h.resolveOrg(w, r, orgName)
	if !ok {
		return nil, nil, false
	}
	teamID := atoiOrZero(chi.URLParam(r, "teamID"))
	team, err := h.Orgs.GetTeam(uint(teamID))
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "team not found")
		return nil, nil, false
	}
	if team.OrgID != org.ID {
		// Team exists but lives under another org — hide its existence.
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "team not found")
		return nil, nil, false
	}
	if !h.requireOrgAdmin(w, r, org) {
		return nil, nil, false
	}
	return org, team, true
}

func (h *OrgHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	_, team, ok := h.resolveTeamInOrg(w, r)
	if !ok {
		return
	}
	if err := h.Orgs.DeleteTeam(team.ID); err != nil {
		writeServerError(w, r, "delete team", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	_, team, ok := h.resolveTeamInOrg(w, r)
	if !ok {
		return
	}
	var req AddMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Find user by username
	var user models.User
	if err := h.Orgs.(*orgstore.Service).DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "user not found")
		return
	}
	_, err := h.Orgs.AddTeamMember(team.ID, user.ID)
	if err != nil {
		writeServerError(w, r, "add team member", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *OrgHandler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	_, team, ok := h.resolveTeamInOrg(w, r)
	if !ok {
		return
	}
	userID := atoiOrZero(chi.URLParam(r, "userID"))
	if err := h.Orgs.RemoveTeamMember(team.ID, uint(userID)); err != nil {
		writeServerError(w, r, "remove team member", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID := atoiOrZero(chi.URLParam(r, "teamID"))
	members, err := h.Orgs.ListTeamMembers(uint(teamID))
	if err != nil {
		writeServerError(w, r, "list team members", err)
		return
	}
	items := make([]models.TeamMemberDTO, len(members))
	for i, m := range members {
		items[i] = m.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *OrgHandler) AddTeamRepo(w http.ResponseWriter, r *http.Request) {
	org, team, ok := h.resolveTeamInOrg(w, r)
	if !ok {
		return
	}
	var req struct {
		RepoName   string `json:"repo_name"`
		Permission string `json:"permission"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	// Resolve repo (must belong to this org)
	repo, err := h.Repos.GetByOwnerAndName(org.Name, req.RepoName)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
		return
	}
	tr, err := h.Orgs.AddTeamRepo(team.ID, repo.ID, req.Permission)
	if err != nil {
		writeServerError(w, r, "add team repo", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, tr.DTO())
}

func (h *OrgHandler) RemoveTeamRepo(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.resolveTeamInOrg(w, r); !ok {
		return
	}
	teamRepoID := atoiOrZero(chi.URLParam(r, "teamRepoID"))
	if err := h.Orgs.RemoveTeamRepo(uint(teamRepoID)); err != nil {
		writeServerError(w, r, "remove team repo", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) ListTeamRepos(w http.ResponseWriter, r *http.Request) {
	teamID := atoiOrZero(chi.URLParam(r, "teamID"))
	repos, err := h.Orgs.ListTeamRepos(uint(teamID))
	if err != nil {
		writeServerError(w, r, "list team repos", err)
		return
	}
	items := make([]models.TeamRepoDTO, len(repos))
	for i, r := range repos {
		items[i] = r.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

// requireOrgAdmin ensures the caller is a site admin or an owner/admin member
// of the organization. Team management routes previously had no permission
// checks at all — any authenticated user could mutate any org's teams.
func (h *OrgHandler) requireOrgAdmin(w http.ResponseWriter, r *http.Request, org *models.Organization) bool {
	user, ok := requireUser(w, r)
	if !ok {
		return false
	}
	if user.IsSiteAdmin {
		return true
	}
	member, err := h.Orgs.GetMember(org.ID, user.ID)
	if err != nil || (member.Role != "owner" && member.Role != "admin") {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "insufficient permissions")
		return false
	}
	return true
}

// ── helpers ─────────────────────────────────────────────

func (h *OrgHandler) resolveOrg(w http.ResponseWriter, r *http.Request, name string) (*models.Organization, bool) {
	org, err := h.Orgs.GetByName(name)
	if err != nil {
		if errors.Is(err, orgstore.ErrOrgNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "organization not found")
			return nil, false
		}
		writeServerError(w, r, "resolve org", err)
		return nil, false
	}
	return org, true
}
