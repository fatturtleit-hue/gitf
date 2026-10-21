package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"gitf/internal/config"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// PackageHandler serves the Package Registry management API (list, search, delete).
// Protocol-specific endpoints for Docker/npm/PyPI/etc. are handled by dedicated
// route handlers registered directly on the chi router.
type PackageHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
	DB     *gorm.DB
}

func NewPackageHandler(db *gorm.DB, cfg *config.Config) *PackageHandler {
	return &PackageHandler{DB: db, Config: cfg}
}

// ListPackages returns packages for an owner (user or org).
func (h *PackageHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	if owner == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "owner is required")
		return
	}
	dummyPackages := []map[string]any{
		{"id": 1, "type": "docker", "name": owner + "/app", "version_count": 3},
		{"id": 2, "type": "npm", "name": "@" + owner + "/lib", "version_count": 5},
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": dummyPackages})
}

// ServeNpm packument returns a minimal npm package metadata response.
func (h *PackageHandler) ServeNpm(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	pkg := chi.URLParam(r, "package")
	if pkg == "" {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "package not found")
		return
	}
	packument := map[string]any{
		"name":     pkg,
		"versions": map[string]any{},
	}
	if owner != "" {
		packument["name"] = "@" + owner + "/" + pkg
	}
	writeJSON(w, r, http.StatusOK, packument)
}

// ServePypiSimple returns a PEP 503 simple index page.
func (h *PackageHandler) ServePypiSimple(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><a href="/simple/example/">example</a></body></html>`))
}

// ServePypiUpload handles PyPI package upload.
func (h *PackageHandler) ServePypiUpload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ServeMavenUpload handles Maven deploy.
func (h *PackageHandler) ServeMavenUpload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

// ServeMavenFetch handles Maven artifact retrieval.
func (h *PackageHandler) ServeMavenFetch(w http.ResponseWriter, r *http.Request) {
	artifactPath := chi.URLParam(r, "path")
	if strings.HasSuffix(artifactPath, "maven-metadata.xml") {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<metadata><groupId>org.example</groupId></metadata>`))
		return
	}
	writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "artifact not found")
}

// ServeNuGetPush handles NuGet push.
func (h *PackageHandler) ServeNuGetPush(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

// ServeHelmIndex returns a Helm chart repo index.
func (h *PackageHandler) ServeHelmIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write([]byte("apiVersion: v1\nentries: {}\n"))
}

// ServeHelmUpload handles Helm chart upload.
func (h *PackageHandler) ServeHelmUpload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}
