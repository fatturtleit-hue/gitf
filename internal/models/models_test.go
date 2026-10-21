package models

import (
	"testing"
	"time"
)

// TestDTOsHandleNilPointers pins the nil-guard behavior of the conditional
// DTO fields (dates, optional relations) — a regression here serializes
// "<nil>" or panics.
func TestDTOsHandleNilPointers(t *testing.T) {
	now := time.Now()

	issue := Issue{Author: User{Username: "alice"}}
	dto := issue.DTO()
	if dto.ClosedAt != nil {
		t.Errorf("nil ClosedAt should serialize nil, got %v", *dto.ClosedAt)
	}
	if dto.AssigneeID != nil || dto.AssigneeUsername != "" {
		t.Errorf("nil Assignee should serialize empty, got %+v", dto)
	}

	closedAt := now
	assignee := User{Username: "bob"}
	issue.ClosedAt = &closedAt
	issue.Assignee = &assignee
	dto = issue.DTO()
	if dto.ClosedAt == nil || dto.AssigneeUsername != "bob" {
		t.Errorf("non-nil fields lost: %+v", dto)
	}

	ms := Milestone{Title: "M1"}
	if ms.DTO().DueDate != nil {
		t.Error("nil milestone DueDate should serialize nil")
	}

	pr := PullRequest{}
	if pr.DTO().MergedAt != nil {
		t.Error("nil PR MergedAt should serialize nil")
	}
	mergedAt := now
	pr.MergedAt = &mergedAt
	if pr.DTO().MergedAt == nil || *pr.DTO().MergedAt == "" {
		t.Error("set MergedAt should serialize")
	}

	pat := PersonalAccessToken{ExpiresAt: nil}
	if pat.DTO().ExpiresAt != nil {
		t.Error("nil PAT ExpiresAt should serialize nil")
	}
}

func TestTableNameOverrides(t *testing.T) {
	// Singular/plural table names are load-bearing for raw SQL in services.
	// Pointer values match TableName regardless of receiver form.
	cases := map[any]string{
		&Repository{}:          "repositories",
		&Organization{}:        "organizations",
		&OrgMember{}:           "org_members",
		&Team{}:                "teams",
		&TeamMember{}:          "team_members",
		&TeamRepo{}:            "team_repos",
		&Issue{}:               "issues",
		&PullRequest{}:         "pull_requests",
		&Review{}:              "reviews",
		&Release{}:             "releases",
		&ReleaseAsset{}:        "release_assets",
		&Webhook{}:             "webhooks",
		&WebhookDelivery{}:     "webhook_deliveries",
		&PersonalAccessToken{}: "personal_access_tokens",
		&BranchProtection{}:    "branch_protections",
		&Comment{}:             "comments",
		&Label{}:               "labels",
		&Milestone{}:           "milestones",
	}
	for model, want := range cases {
		var got string
		switch m := model.(type) {
		case interface{ TableName() string }:
			got = m.TableName()
		}
		if got != want {
			t.Errorf("TableName() = %q, want %q", got, want)
		}
	}
}
