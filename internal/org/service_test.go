package org

import (
	"errors"
	"testing"

	"gitf/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrgMember{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamRepo{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t))
}

func seedUser(t *testing.T, svc *Service, username string) uint {
	t.Helper()
	u := models.User{Username: username, Email: username + "@test.local", PasswordHash: "x"}
	if err := svc.DB.Create(&u).Error; err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}
	return u.ID
}

func seedOrg(t *testing.T, svc *Service, name string, ownerID uint) *models.Organization {
	t.Helper()
	org, err := svc.Create(name, name+" display", "desc", ownerID)
	if err != nil {
		t.Fatalf("seed org %s: %v", name, err)
	}
	return org
}

func TestCreateAndGetOrg(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")

	org, err := svc.Create("acme", "Acme Inc", "the org", ownerID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if org.Name != "acme" || org.DisplayName != "Acme Inc" {
		t.Errorf("created org = %+v", org)
	}

	got, err := svc.GetByName("acme")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.ID != org.ID {
		t.Errorf("GetByName ID = %d, want %d", got.ID, org.ID)
	}

	// Duplicate name is rejected.
	if _, err := svc.Create("acme", "dup", "", ownerID); !errors.Is(err, ErrOrgConflict) {
		t.Errorf("duplicate create err = %v, want ErrOrgConflict", err)
	}
}

func TestUpdateOrg(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	org := seedOrg(t, svc, "acme", ownerID)

	updated, err := svc.Update(org.ID, "New Name", "new desc")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != "New Name" || updated.Description != "new desc" {
		t.Errorf("updated org = %+v", updated)
	}
}

func TestDeleteOrg(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	org := seedOrg(t, svc, "acme", ownerID)

	if err := svc.Delete(org.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.GetByName("acme"); !errors.Is(err, ErrOrgNotFound) {
		t.Errorf("after delete GetByName err = %v, want ErrOrgNotFound", err)
	}
}

func TestListOrgs(t *testing.T) {
	svc := newTestService(t)
	aliceID := seedUser(t, svc, "alice")
	bobID := seedUser(t, svc, "bob")
	seedOrg(t, svc, "org-a", aliceID)
	seedOrg(t, svc, "org-b", aliceID)

	aliceOrgs, err := svc.List(aliceID)
	if err != nil {
		t.Fatalf("List alice: %v", err)
	}
	if len(aliceOrgs) != 2 {
		t.Errorf("alice orgs = %d, want 2", len(aliceOrgs))
	}
	bobOrgs, err := svc.List(bobID)
	if err != nil {
		t.Fatalf("List bob: %v", err)
	}
	if len(bobOrgs) != 0 {
		t.Errorf("bob orgs = %d, want 0", len(bobOrgs))
	}
}

func TestMembers(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	bobID := seedUser(t, svc, "bob")
	org := seedOrg(t, svc, "acme", ownerID)

	// Owner was added automatically at creation.
	if _, err := svc.GetMember(org.ID, ownerID); err != nil {
		t.Errorf("owner should be a member: %v", err)
	}

	m, err := svc.AddMember(org.ID, bobID, "member")
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	got, err := svc.GetMember(org.ID, bobID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.ID != m.ID || got.Role != "member" {
		t.Errorf("GetMember = %+v, want role member", got)
	}

	promoted, err := svc.UpdateMember(m.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if promoted.Role != "admin" {
		t.Errorf("role after promote = %s", promoted.Role)
	}

	members, err := svc.ListMembers(org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("members = %d, want 2", len(members))
	}

	if err := svc.RemoveMember(org.ID, bobID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if _, err := svc.GetMember(org.ID, bobID); !errors.Is(err, ErrNotOrgMember) {
		t.Errorf("after remove err = %v, want ErrNotOrgMember", err)
	}
}

func TestTeams(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	org := seedOrg(t, svc, "acme", ownerID)

	team, err := svc.CreateTeam(org.ID, "devs", "developers")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.OrgID != org.ID || team.Name != "devs" {
		t.Errorf("team = %+v", team)
	}

	if _, err := svc.CreateTeam(org.ID, "devs", "dup"); !errors.Is(err, ErrTeamConflict) {
		t.Errorf("duplicate team err = %v, want ErrTeamConflict", err)
	}

	got, err := svc.GetTeam(team.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "devs" {
		t.Errorf("GetTeam name = %s", got.Name)
	}

	teams, err := svc.ListTeams(org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 {
		t.Errorf("teams = %d, want 1", len(teams))
	}

	if err := svc.DeleteTeam(team.ID); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if _, err := svc.GetTeam(team.ID); !errors.Is(err, ErrTeamNotFound) {
		t.Errorf("after delete err = %v, want ErrTeamNotFound", err)
	}
}

func TestTeamMembers(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	bobID := seedUser(t, svc, "bob")
	org := seedOrg(t, svc, "acme", ownerID)
	team, err := svc.CreateTeam(org.ID, "devs", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddTeamMember(team.ID, bobID); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	members, err := svc.ListTeamMembers(team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("team members = %d, want 1", len(members))
	}
	if err := svc.RemoveTeamMember(team.ID, bobID); err != nil {
		t.Fatalf("RemoveTeamMember: %v", err)
	}
	members, _ = svc.ListTeamMembers(team.ID)
	if len(members) != 0 {
		t.Errorf("team members after remove = %d, want 0", len(members))
	}
}

func TestTeamRepos(t *testing.T) {
	svc := newTestService(t)
	ownerID := seedUser(t, svc, "owner")
	org := seedOrg(t, svc, "acme", ownerID)
	team, err := svc.CreateTeam(org.ID, "devs", "")
	if err != nil {
		t.Fatal(err)
	}

	tr, err := svc.AddTeamRepo(team.ID, 42, "write")
	if err != nil {
		t.Fatalf("AddTeamRepo: %v", err)
	}
	if tr.RepoID != 42 || tr.Permission != "write" {
		t.Errorf("team repo = %+v", tr)
	}

	if _, err := svc.AddTeamRepo(team.ID, 42, "read"); !errors.Is(err, ErrTeamRepoConflict) {
		t.Errorf("duplicate team repo err = %v, want ErrTeamRepoConflict", err)
	}

	repos, err := svc.ListTeamRepos(team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("team repos = %d, want 1", len(repos))
	}

	if err := svc.RemoveTeamRepo(tr.ID); err != nil {
		t.Fatalf("RemoveTeamRepo: %v", err)
	}
	repos, _ = svc.ListTeamRepos(team.ID)
	if len(repos) != 0 {
		t.Errorf("team repos after remove = %d, want 0", len(repos))
	}
}
