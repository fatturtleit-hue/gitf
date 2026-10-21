package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/middleware"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
)

// testPassword is the shared fixture credential for auth tests. It is read
// from the environment so no literal credential is committed; tests fall back
// to a deterministic synthetic value when the variable is unset.
func testPassword() string {
	if pw := os.Getenv("GITF_TEST_PASSWORD"); pw != "" {
		return pw
	}
	return strings.Repeat("gitf-test-", 2) + "Px7"
}

// mockRepoStore is a test double implementing repo.RepositoryStore.
type mockRepoStore struct {
	repos       []*models.Repository
	createErr   error
	deleteErr   error
	getErr      error
	listErr     error
	countErr    error
	countReturn int
}

func (m *mockRepoStore) Create(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool) (*models.Repository, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	r := &models.Repository{
		ID:            uint(len(m.repos) + 1),
		OwnerType:     ownerType,
		OwnerID:       ownerID,
		Owner:         models.User{ID: ownerID, Username: "alice"},
		Name:          name,
		Description:   description,
		IsPrivate:     isPrivate,
		DefaultBranch: defaultBranch,
	}
	if r.DefaultBranch == "" {
		r.DefaultBranch = "main"
	}
	m.repos = append(m.repos, r)
	return r, nil
}

func (m *mockRepoStore) Delete(ownerType string, ownerID uint, name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, r := range m.repos {
		if r.OwnerID == ownerID && r.Name == name {
			m.repos = append(m.repos[:i], m.repos[i+1:]...)
			return nil
		}
	}
	return repo.ErrNotFound
}

func (m *mockRepoStore) GetByOwnerAndName(ownerName, name string) (*models.Repository, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, r := range m.repos {
		if r.Owner.Username == ownerName && r.Name == name {
			return r, nil
		}
	}
	return nil, repo.ErrNotFound
}

func (m *mockRepoStore) List(userID uint, search string, limit, offset int) ([]models.Repository, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []models.Repository
	for _, r := range m.repos {
		if r.IsPrivate && r.OwnerID != userID {
			continue
		}
		if search != "" && !strings.Contains(r.Name, search) {
			continue
		}
		out = append(out, *r)
	}
	return out, nil
}

func (m *mockRepoStore) Count(userID uint, search string) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.countReturn, nil
}

func (m *mockRepoStore) CanRead(r *models.Repository, user *models.User) bool {
	return !r.IsPrivate || (user != nil && user.ID == r.OwnerID)
}

func (m *mockRepoStore) CanWrite(r *models.Repository, user *models.User) bool {
	return user != nil && user.ID == r.OwnerID
}

func (m *mockRepoStore) CanAdmin(r *models.Repository, user *models.User) bool {
	return user != nil && user.ID == r.OwnerID
}

func (m *mockRepoStore) Fork(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool, forkedFromID uint) (*models.Repository, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	r := &models.Repository{
		OwnerType:     ownerType,
		OwnerID:       ownerID,
		Owner:         models.User{ID: ownerID, Username: "alice"},
		Name:          name,
		Description:   description,
		IsPrivate:     isPrivate,
		DefaultBranch: defaultBranch,
		ForkedFromID:  &forkedFromID,
	}
	m.repos = append(m.repos, r)
	return r, nil
}

func (m *mockRepoStore) ResolveOwner(ownerName string) (string, uint, string, error) {
	return repo.OwnerTypeUser, 1, ownerName, nil
}

func (m *mockRepoStore) ListCollaborators(repoID uint) ([]models.Collaborator, error) {
	return nil, nil
}

func (m *mockRepoStore) AddCollaborator(repoID uint, username string, permission string, quotaBytes int64) (*models.Collaborator, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepoStore) UpdateCollaborator(collabID uint, permission string, quotaBytes int64) (*models.Collaborator, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepoStore) RemoveCollaborator(repoID uint, userID uint) error {
	return nil
}

func (m *mockRepoStore) CanReadWithCollab(repo *models.Repository, user *models.User) bool {
	return m.CanRead(repo, user)
}

func (m *mockRepoStore) CountRepos() (int64, error) {
	return int64(len(m.repos)), nil
}

func (m *mockRepoStore) CountUsers() (int64, error) {
	return 1, nil
}

func (m *mockRepoStore) Update(repo *models.Repository) error {
	return nil
}

// mockUserStore implements handlers.UserStore for unit tests.
type mockUserStore struct {
	users       map[string]*models.User
	createErr   error
	nextID      uint
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*models.User)}
}

func (m *mockUserStore) Create(ctx context.Context, u *models.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.nextID++
	u.ID = m.nextID
	m.users[u.Username] = u
	return nil
}

func (m *mockUserStore) ByUsername(ctx context.Context, username string) (*models.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserStore) ByID(ctx context.Context, id uint) (*models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *mockUserStore) List(ctx context.Context, search string, limit, offset int) ([]models.User, error) {
	var result []models.User
	for _, u := range m.users {
		result = append(result, *u)
	}
	// Simple offset/limit
	if offset >= len(result) {
		return nil, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockUserStore) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserStore) Update(ctx context.Context, u *models.User) error {
	if _, ok := m.users[u.Username]; !ok {
		// Handle username change: delete old key, create new
		for k, v := range m.users {
			if v.ID == u.ID {
				delete(m.users, k)
				break
			}
		}
	}
	m.users[u.Username] = u
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id uint) error {
	for k, v := range m.users {
		if v.ID == id {
			delete(m.users, k)
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *mockUserStore) Exists(ctx context.Context) (bool, error) {
	return len(m.users) > 0, nil
}

// mockAuditStore implements models.AuditStore for unit tests.
type mockAuditStore struct{}

func (m *mockAuditStore) Log(userID uint, action, resource string) error { return nil }

func newTestRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithRequestID(r.Context(), "test-req"))
}

func chiContext(t *testing.T, r *http.Request, params map[string]string) *http.Request {
	t.Helper()
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestAuthHandlerRegister(t *testing.T) {
	users := newMockUserStore()
	h := &AuthHandler{Users: users, Audit: &mockAuditStore{}, Config: &config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"}}

	req := newTestRequest(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: testPassword(),
	})
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.User.Username != "alice" {
		t.Errorf("username = %q, want alice", resp.User.Username)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthHandlerRegisterValidation(t *testing.T) {
	h := &AuthHandler{Users: newMockUserStore(), Audit: &mockAuditStore{}, Config: &config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"}}
	bad := []any{
		RegisterRequest{Username: "", Email: "a@b.com", Password: testPassword()},
		RegisterRequest{Username: "alice", Email: "not-an-email", Password: testPassword()},
		RegisterRequest{Username: "alice", Email: "a@b.com", Password: "short"},
	}
	for _, body := range bad {
		req := newTestRequest(t, http.MethodPost, "/api/v1/auth/register", body)
		w := httptest.NewRecorder()
		h.Register(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	users := newMockUserStore()
	cfg := &config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"}
	h := &AuthHandler{Users: users, Audit: &mockAuditStore{}, Config: cfg}

	hash, _ := auth.HashPassword(testPassword())
	_ = users.Create(context.Background(), &models.User{Username: "bob", Email: "bob@example.com", PasswordHash: hash})

	req := newTestRequest(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Username: "bob",
		Password: testPassword(),
	})
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token")
	}
}

func TestAuthHandlerLoginWrongPassword(t *testing.T) {
	users := newMockUserStore()
	h := &AuthHandler{Users: users, Audit: &mockAuditStore{}, Config: &config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"}}
	hash, _ := auth.HashPassword(testPassword())
	_ = users.Create(context.Background(), &models.User{Username: "bob", Email: "bob@example.com", PasswordHash: hash})

	req := newTestRequest(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Username: "bob",
		Password: "wrong",
	})
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAPIHandlerCreateRepo(t *testing.T) {
	store := &mockRepoStore{countReturn: 0}
	cfg := &config.Config{}
	h := NewAPIHandlerWithStore(cfg, store)

	req := chiContext(t, newTestRequest(t, http.MethodPost, "/api/v1/repos", CreateRepoRequest{Name: "hello"}), nil)
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 1, Username: "alice"}))
	w := httptest.NewRecorder()
	h.CreateRepo(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var dto models.RepoDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dto.Name != "hello" {
		t.Errorf("name = %q, want hello", dto.Name)
	}
}

func TestAPIHandlerCreateRepoInvalidName(t *testing.T) {
	h := NewAPIHandlerWithStore(&config.Config{}, &mockRepoStore{})
	req := newTestRequest(t, http.MethodPost, "/api/v1/repos", CreateRepoRequest{Name: "../evil"})
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 1, Username: "alice"}))
	w := httptest.NewRecorder()
	h.CreateRepo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIHandlerGetRepo(t *testing.T) {
	store := &mockRepoStore{}
	repo, _ := store.Create(repo.OwnerTypeUser, 1, "public", "", "main", false)
	repo.Owner = models.User{ID: 1, Username: "alice"}

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := chiContext(t, newTestRequest(t, http.MethodGet, "/api/v1/repos/alice/public", nil), map[string]string{
		"owner": "alice",
		"name":  "public",
	})
	w := httptest.NewRecorder()
	h.GetRepo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIHandlerGetRepoPrivateNotOwner(t *testing.T) {
	store := &mockRepoStore{}
	repo, _ := store.Create(repo.OwnerTypeUser, 1, "secret", "", "main", true)
	repo.Owner = models.User{ID: 1, Username: "alice"}

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := chiContext(t, newTestRequest(t, http.MethodGet, "/api/v1/repos/alice/secret", nil), map[string]string{
		"owner": "alice",
		"name":  "secret",
	})
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 2, Username: "bob"}))
	w := httptest.NewRecorder()
	h.GetRepo(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for private repo by non-owner, got %d", w.Code)
	}
}

func TestAPIHandlerDeleteRepo(t *testing.T) {
	store := &mockRepoStore{}
	repo, _ := store.Create(repo.OwnerTypeUser, 1, "bye", "", "main", false)
	repo.Owner = models.User{ID: 1, Username: "alice"}

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := chiContext(t, newTestRequest(t, http.MethodDelete, "/api/v1/repos/alice/bye", nil), map[string]string{
		"owner": "alice",
		"name":  "bye",
	})
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 1, Username: "alice"}))
	w := httptest.NewRecorder()
	h.DeleteRepo(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIHandlerDeleteRepoNotOwner(t *testing.T) {
	store := &mockRepoStore{}
	repo, _ := store.Create(repo.OwnerTypeUser, 1, "mine", "", "main", false)
	repo.Owner = models.User{ID: 1, Username: "alice"}

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := chiContext(t, newTestRequest(t, http.MethodDelete, "/api/v1/repos/alice/mine", nil), map[string]string{
		"owner": "alice",
		"name":  "mine",
	})
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 2, Username: "bob"}))
	w := httptest.NewRecorder()
	h.DeleteRepo(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAPIHandlerListRepos(t *testing.T) {
	store := &mockRepoStore{countReturn: 2}
	_, _ = store.Create(repo.OwnerTypeUser, 1, "a", "", "main", false)
	_, _ = store.Create(repo.OwnerTypeUser, 1, "b", "", "main", false)

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := newTestRequest(t, http.MethodGet, "/api/v1/repos", nil)
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 1, Username: "alice"}))
	w := httptest.NewRecorder()
	h.ListRepos(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Items      []models.RepoDTO `json:"items"`
		Pagination Pagination       `json:"pagination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 2 {
		t.Errorf("items = %d, want 2", len(body.Items))
	}
	if body.Pagination.Total != 2 {
		t.Errorf("pagination.total = %d, want 2", body.Pagination.Total)
	}
}

func TestAPIHandlerListReposCountError(t *testing.T) {
	store := &mockRepoStore{countErr: errors.New("boom")}
	_, _ = store.Create(repo.OwnerTypeUser, 1, "a", "", "main", false)

	h := NewAPIHandlerWithStore(&config.Config{}, store)
	req := newTestRequest(t, http.MethodGet, "/api/v1/repos", nil)
	req = req.WithContext(middleware.WithUser(req.Context(), &models.User{ID: 1, Username: "alice"}))
	w := httptest.NewRecorder()
	h.ListRepos(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on count error, got %d", w.Code)
	}
}

func TestRepoPathSanity(t *testing.T) {
	// ValidateName is the single source of truth for safe path components.
	for _, name := range []string{"../evil", "hello/world", "repo.git"} {
		if gitutil.ValidateName(name) {
			t.Errorf("ValidateName(%q) should be false", name)
		}
	}
	for _, name := range []string{"hello", "repo-1", "my_repo", "repo123"} {
		if !gitutil.ValidateName(name) {
			t.Errorf("ValidateName(%q) should be true", name)
		}
	}
}

func TestAPIHandlerUnauthorized(t *testing.T) {
	h := NewAPIHandlerWithStore(&config.Config{}, &mockRepoStore{})
	req := newTestRequest(t, http.MethodPost, "/api/v1/repos", CreateRepoRequest{Name: "x"})
	w := httptest.NewRecorder()
	h.CreateRepo(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
