package issues

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
		&models.Issue{},
		&models.Comment{},
		&models.Label{},
		&models.IssueLabel{},
		&models.Milestone{},
		&models.IssueColumn{},
		&models.IssueColumnItem{},
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
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func TestIssueNumberingAndCRUD(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")

	i1, err := svc.Create(1, authorID, "first", "body", nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	i2, err := svc.Create(1, authorID, "second", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if i1.Number != 1 || i2.Number != 2 {
		t.Errorf("numbers = %d, %d; want 1, 2 (per-repo auto increment)", i1.Number, i2.Number)
	}

	got, err := svc.GetByNumber(1, 2)
	if err != nil {
		t.Fatalf("GetByNumber: %v", err)
	}
	if got.Title != "second" {
		t.Errorf("title = %s", got.Title)
	}
	if _, err := svc.GetByNumber(1, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing issue err = %v, want ErrNotFound", err)
	}
	// Numbers are scoped per repo — a second repo starts at 1 again.
	i3, _ := svc.Create(2, authorID, "other repo", "", nil, nil, nil)
	if i3.Number != 1 {
		t.Errorf("second repo issue number = %d, want 1", i3.Number)
	}
}

func TestIssueCloseReopen(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")
	issue, _ := svc.Create(1, authorID, "bug", "", nil, nil, nil)

	if err := svc.Close(issue, authorID); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if issue.Status != "closed" || issue.ClosedAt == nil {
		t.Errorf("after close: status=%s closedAt=%v", issue.Status, issue.ClosedAt)
	}
	if err := svc.Reopen(issue); err != nil {
		t.Fatal(err)
	}
	if issue.Status != "open" || issue.ClosedAt != nil {
		t.Errorf("after reopen: status=%s closedAt=%v", issue.Status, issue.ClosedAt)
	}
}

func TestIssueListFilters(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")
	assigneeID := seedUser(t, svc, "bob")

	label, _ := svc.CreateLabel(1, "bug", "#ff0000")
	ms, _ := svc.CreateMilestone(1, "M1", "", nil)

	open, _ := svc.Create(1, authorID, "open issue", "login broken", &assigneeID, &ms.ID, []uint{label.ID})
	closed, _ := svc.Create(1, authorID, "closed issue", "old thing", nil, nil, nil)
	svc.Close(closed, authorID)
	svc.Create(2, authorID, "other repo", "", nil, nil, nil)

	if open.ID == 0 || closed.ID == 0 {
		t.Fatal("seed failed")
	}

	all, total, err := svc.List(1, "", "", "", "", "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(all) != 2 {
		t.Errorf("all: total=%d len=%d, want 2/2", total, len(all))
	}

	openOnly, total, _ := svc.List(1, "open", "", "", "", "", 20, 0)
	if total != 1 || openOnly[0].Title != "open issue" {
		t.Errorf("status filter: total=%d", total)
	}

	byLabel, total, err := svc.List(1, "", "bug", "", "", "", 20, 0)
	if err != nil {
		t.Fatalf("label filter: %v", err)
	}
	if total != 1 || byLabel[0].ID != open.ID {
		t.Errorf("label filter: total=%d", total)
	}

	_, byMilestoneTotal, err := svc.List(1, "", "", "M1", "", "", 20, 0)
	if err != nil {
		t.Fatalf("milestone filter: %v", err)
	}
	if byMilestoneTotal != 1 {
		t.Errorf("milestone filter: total=%d", byMilestoneTotal)
	}

	_, byAssigneeTotal, err := svc.List(1, "", "", "", "bob", "", 20, 0)
	if err != nil {
		t.Fatalf("assignee filter: %v", err)
	}
	if byAssigneeTotal != 1 {
		t.Errorf("assignee filter: total=%d", byAssigneeTotal)
	}

	bySearch, total, _ := svc.List(1, "", "", "", "", "login", 20, 0)
	if total != 1 || bySearch[0].ID != open.ID {
		t.Errorf("search filter: total=%d", total)
	}
}

func TestComments(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")
	issue, _ := svc.Create(1, authorID, "t", "", nil, nil, nil)

	c1, err := svc.AddComment(1, issue.ID, authorID, "first comment")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if c1.Author.Username != "alice" {
		t.Errorf("comment author not preloaded: %+v", c1)
	}
	comments, err := svc.ListComments(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || comments[0].Body != "first comment" {
		t.Errorf("comments = %+v", comments)
	}
}

func TestLabels(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")

	label, err := svc.CreateLabel(1, "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	// Duplicate creation is tolerated (returns the existing label).
	dup, err := svc.CreateLabel(1, "bug", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}
	if dup.ID != label.ID {
		t.Errorf("duplicate label should return existing row: %d vs %d", dup.ID, label.ID)
	}

	issue, _ := svc.Create(1, authorID, "t", "", nil, nil, nil)
	if err := svc.AddIssueLabel(issue.ID, label.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveIssueLabel(issue.ID, label.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteLabel(label.ID); err != nil {
		t.Fatalf("DeleteLabel: %v", err)
	}
	labels, _ := svc.ListLabels(1)
	if len(labels) != 0 {
		t.Errorf("labels after delete = %d", len(labels))
	}
}

func TestMilestones(t *testing.T) {
	svc := newTestService(t)

	badDate := "not-a-date"
	ms, err := svc.CreateMilestone(1, "M1", "desc", &badDate)
	if err != nil {
		t.Fatal(err)
	}
	if ms.DueDate != nil {
		t.Error("bad due date should be silently ignored (documented behavior)")
	}

	goodDate := "2026-12-31"
	ms2, _ := svc.CreateMilestone(1, "M2", "", &goodDate)
	if ms2.DueDate == nil {
		t.Error("valid due date should be stored")
	}

	if err := svc.CloseMilestone(ms); err != nil {
		t.Fatal(err)
	}
	if ms.Status != "closed" {
		t.Errorf("milestone status = %s", ms.Status)
	}

	milestones, _ := svc.ListMilestones(1)
	if len(milestones) != 2 {
		t.Errorf("milestones = %d, want 2", len(milestones))
	}
}

func TestKanban(t *testing.T) {
	svc := newTestService(t)
	authorID := seedUser(t, svc, "alice")

	cols, err := svc.GetColumns(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) == 0 {
		t.Fatal("default columns should be auto-created")
	}

	issue, _ := svc.Create(1, authorID, "kanban task", "", nil, nil, nil)
	if err := svc.MoveIssue(issue.ID, cols[0].ID, 0); err != nil {
		t.Fatalf("MoveIssue: %v", err)
	}
	colIssues, err := svc.GetColumnIssues(cols[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(colIssues) != 1 || colIssues[0].ID != issue.ID {
		t.Errorf("column issues = %+v", colIssues)
	}

	// Moving to another column relocates the issue (unique index on issue_id).
	if len(cols) > 1 {
		if err := svc.MoveIssue(issue.ID, cols[1].ID, 0); err != nil {
			t.Fatalf("MoveIssue to second column: %v", err)
		}
		first, _ := svc.GetColumnIssues(cols[0].ID)
		if len(first) != 0 {
			t.Errorf("issue should have left column 0, got %d items", len(first))
		}
		second, _ := svc.GetColumnIssues(cols[1].ID)
		if len(second) != 1 {
			t.Errorf("issue should be in column 1")
		}
	}
}
