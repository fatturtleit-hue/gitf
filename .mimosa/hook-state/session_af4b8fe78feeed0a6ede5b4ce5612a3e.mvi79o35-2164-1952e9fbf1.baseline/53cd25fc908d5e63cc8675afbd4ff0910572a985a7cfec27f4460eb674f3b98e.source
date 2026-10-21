package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"gorm.io/gorm"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/db"
	"gitf/internal/gitutil"
	"gitf/internal/handlers"
	"gitf/internal/log"
	"gitf/internal/metrics"
	"gitf/internal/middleware"
	"gitf/internal/models"
	"gitf/internal/webhooks"
)

// version is set at build time via -ldflags. The default is "1.0.0".
var version = "1.0.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		// Minimal health probe used by Docker HEALTHCHECK.
		// Performs a TCP connect to the configured listen address.
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "healthcheck config load failed:", err)
			os.Exit(1)
		}
		conn, err := net.DialTimeout("tcp", cfg.Addr, 5*time.Second)
		if err != nil {
			os.Exit(1)
		}
		conn.Close()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Init(cfg.Log.Level, cfg.Log.Format)
	slog.Info("gitf starting",
		"version", version,
		"addr", cfg.Addr,
		"data_dir", cfg.DataDir,
	)

	database, err := db.New(cfg)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	sqlDB, err := database.DB()
	if err != nil {
		slog.Error("failed to get sql.DB handle", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Ensure a default admin account exists on first startup.
	if err := ensureDefaultAdmin(database, cfg); err != nil {
		slog.Error("failed to ensure default admin", "error", err)
		os.Exit(1)
	}

	authHandler := handlers.NewAuthHandler(database, cfg)
	apiHandler := handlers.NewAPIHandler(database, cfg)
	gitHandler := handlers.NewGitHTTPHandler(cfg, apiHandler.Repos)
	healthHandler := handlers.NewHealthHandler(database)
	adminHandler := handlers.NewAdminHandler(cfg, apiHandler.Repos)
	collabHandler := handlers.NewCollaboratorHandler(cfg, apiHandler.Repos)
	orgHandler := handlers.NewOrgHandler(database, cfg, apiHandler.Repos)
	releaseHandler := handlers.NewReleaseHandler(database, cfg, apiHandler.Repos)
	protectionHandler := handlers.NewProtectionHandler(database, apiHandler.Repos)
	issueHandler := handlers.NewIssueHandler(database, apiHandler.Repos)
	labelHandler := handlers.NewLabelHandler(database, apiHandler.Repos)
	milestoneHandler := handlers.NewMilestoneHandler(database, apiHandler.Repos)
	kanbanHandler := handlers.NewKanbanHandler(database, apiHandler.Repos)
	prHandler := handlers.NewPRHandler(database, cfg, apiHandler.Repos)
	userHandler := handlers.NewUserHandler(database, cfg)
	tokenStore := auth.NewTokenStore(database)

	// Webhook delivery workers must exist before the internal git hook
	// handler, which enqueues deliveries for post-receive.
	dispatcher := webhooks.NewDispatcher(database, cfg.WebhookWorkerCount, cfg.WebhookTimeoutSec, cfg.WebhookMaxRetries)
	defer dispatcher.Shutdown()
	internalHandler := handlers.NewGitInternalHandler(database, apiHandler.Repos, cfg, dispatcher)

	// Repair/reinstall server-side git hooks for repositories created by
	// older versions (missing owner headers or post-receive).
	if n := gitutil.SweepInstallHooks(cfg); n > 0 {
		slog.Info("reinstalled git hooks", "repositories", n)
	}

	patHandler := handlers.NewPATHandler(database)
	webhookHandler := handlers.NewWebhookHandler(database, apiHandler.Repos, dispatcher)

	// Phase 5: Actions
	actionsHandler := handlers.NewActionsHandler(database, apiHandler.Repos)
	pkgHandler := handlers.NewPackageHandler(database, cfg)

	r := buildRouter(cfg, database, authHandler, apiHandler, gitHandler, healthHandler, adminHandler, userHandler, collabHandler, orgHandler, releaseHandler, protectionHandler, internalHandler, issueHandler, labelHandler, milestoneHandler, kanbanHandler, prHandler, patHandler, webhookHandler, actionsHandler, pkgHandler, tokenStore)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if cfg.TLS.Enabled {
			slog.Info("listening with TLS", "addr", cfg.Addr)
			errCh <- srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			slog.Info("listening", "addr", cfg.Addr)
			errCh <- srv.ListenAndServe()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		if err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped", "version", version)
}

func buildRouter(
	cfg *config.Config,
	database *gorm.DB,
	authHandler *handlers.AuthHandler,
	apiHandler *handlers.APIHandler,
	gitHandler *handlers.GitHTTPHandler,
	healthHandler *handlers.HealthHandler,
	adminHandler *handlers.AdminHandler,
	userHandler *handlers.UserHandler,
	collabHandler *handlers.CollaboratorHandler,
	orgHandler *handlers.OrgHandler,
	releaseHandler *handlers.ReleaseHandler,
	protectionHandler *handlers.ProtectionHandler,
	internalHandler *handlers.GitInternalHandler,
	issueHandler *handlers.IssueHandler,
	labelHandler *handlers.LabelHandler,
	milestoneHandler *handlers.MilestoneHandler,
	kanbanHandler *handlers.KanbanHandler,
	prHandler *handlers.PRHandler,
	patHandler *handlers.PATHandler,
	webhookHandler *handlers.WebhookHandler,
	actionsHandler *handlers.ActionsHandler,
	pkgHandler *handlers.PackageHandler,
	tokenStore auth.TokenStore,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware (outermost first).
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(metrics.MetricsMiddleware)
	r.Use(middleware.SecurityHeaders)
	r.Use(chimw.Recoverer)
	r.Use(chimw.StripSlashes)
	// Compress web text payloads only — the git HTTP and archive endpoints
	// stream binary content types that are deliberately not in this list.
	r.Use(chimw.Compress(5,
		"text/html", "text/css", "text/plain",
		"application/javascript", "application/json",
		"image/svg+xml", "application/xml",
	))

	// Health checks (no auth required).
	r.Get("/health", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// Metrics. Exposed without auth by default because this app is designed to
	// run behind an ingress that can restrict /metrics. If you expose the
	// container directly to the internet, restrict this route with an allowlist.
	r.Get("/metrics", metrics.Handler().ServeHTTP)

	// pprof — enabled via env var in debug builds. Require an additional
	// ENABLE_PPROF env var; no auth by default (intended for local profiling).
	if os.Getenv("ENABLE_PPROF") == "true" {
		r.Get("/debug/pprof/", pprof.Index)
		r.Get("/debug/pprof/cmdline", pprof.Cmdline)
		r.Get("/debug/pprof/profile", pprof.Profile)
		r.Get("/debug/pprof/symbol", pprof.Symbol)
		r.Get("/debug/pprof/trace", pprof.Trace)
	}

	// Git HTTP protocol routes (optional auth for public repos, with rate limiting).
	r.Route("/{owner}/{name}.git", func(r chi.Router) {
		r.Use(middleware.OptionalAuth(authHandler.Users, cfg, tokenStore))
		r.Use(httprate.LimitByIP(120, time.Minute)) // bumped: 60/min is too easy for CI
		r.Get("/info/refs", gitHandler.InfoRefs)
		r.Post("/git-upload-pack", gitHandler.UploadPack)
		r.Post("/git-receive-pack", gitHandler.ReceivePack)
	})

	// Public auth endpoints with rate limiting.
	r.With(httprate.LimitByIP(5, time.Minute)).Post("/api/v1/auth/register", authHandler.Register)
	r.With(httprate.LimitByIP(10, time.Minute)).Post("/api/v1/auth/login", authHandler.Login)
	r.Get("/api/v1/auth/register/status", authHandler.RegistrationStatus)

	// Authenticated API v1.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(authHandler.Users, cfg, tokenStore))
		r.Use(httprate.LimitByIP(1000, time.Minute)) // bumped: 100/min is too low for pages + polling
		r.Get("/user", authHandler.Me)
		r.Post("/auth/refresh", authHandler.RefreshToken)
		r.Post("/auth/setup", authHandler.SetupDefaultAdmin)
		r.Get("/repos", apiHandler.ListRepos)
		r.Post("/repos", apiHandler.CreateRepo)
		r.Get("/repos/{owner}/{name}", apiHandler.GetRepo)
		r.Delete("/repos/{owner}/{name}", apiHandler.DeleteRepo)
		r.Get("/repos/{owner}/{name}/branches", apiHandler.GetRepoBranches)
		r.Get("/repos/{owner}/{name}/stats", apiHandler.GetRepoStats)
		r.Get("/repos/{owner}/{name}/commits", apiHandler.GetRepoCommits)
		r.Get("/repos/{owner}/{name}/commits/{hash}", apiHandler.GetRepoCommit)
		r.Get("/repos/{owner}/{name}/tree", apiHandler.GetRepoTree)
		r.Get("/repos/{owner}/{name}/blob", apiHandler.GetRepoBlob)
		r.Get("/repos/{owner}/{name}/raw", apiHandler.GetRepoRaw)

		// Collaborator management (owner only)
		r.Get("/repos/{owner}/{name}/collaborators", collabHandler.ListCollaborators)
		r.Post("/repos/{owner}/{name}/collaborators", collabHandler.AddCollaborator)
		r.Put("/repos/{owner}/{name}/collaborators/{collabID}", collabHandler.UpdateCollaborator)
		r.Delete("/repos/{owner}/{name}/collaborators/{userID}", collabHandler.RemoveCollaborator)

		// Admin
		// Phase 1 — Tags, Compare, Archive, Fork, Settings
	r.Get("/repos/{owner}/{name}/tags", apiHandler.GetRepoTags)
	r.Get("/repos/{owner}/{name}/compare/{base}...{head}", apiHandler.GetRepoCompare)
	r.Get("/repos/{owner}/{name}/archive/{ref}", apiHandler.GetRepoArchive)
	r.Post("/repos/{owner}/{name}/fork", apiHandler.CreateFork)
	r.Patch("/repos/{owner}/{name}", apiHandler.UpdateRepo)
	// Releases
	r.Get("/repos/{owner}/{name}/releases", releaseHandler.ListReleases)
	r.Post("/repos/{owner}/{name}/releases", releaseHandler.CreateRelease)
	r.Get("/repos/{owner}/{name}/releases/{releaseID}", releaseHandler.GetRelease)
	r.Delete("/repos/{owner}/{name}/releases/{releaseID}", releaseHandler.DeleteRelease)
	r.Post("/repos/{owner}/{name}/releases/{releaseID}/assets", releaseHandler.UploadAsset)
	// Branch protection
	r.Get("/repos/{owner}/{name}/protection", protectionHandler.ListProtections)
	r.Post("/repos/{owner}/{name}/protection", protectionHandler.CreateProtection)
	r.Delete("/repos/{owner}/{name}/protection/{protID}", protectionHandler.DeleteProtection)
// Organizations
	r.Post("/orgs", orgHandler.CreateOrg)
	r.Get("/orgs", orgHandler.ListOrgs)
	r.Get("/orgs/{org}", orgHandler.GetOrg)
	r.Put("/orgs/{org}", orgHandler.UpdateOrg)
	r.Delete("/orgs/{org}", orgHandler.DeleteOrg)
	r.Get("/orgs/{org}/members", orgHandler.ListMembers)
	r.Post("/orgs/{org}/members", orgHandler.AddMember)
	r.Patch("/orgs/{org}/members/{userID}", orgHandler.UpdateMember)
	r.Delete("/orgs/{org}/members/{userID}", orgHandler.RemoveMember)
	r.Post("/orgs/{org}/members/leave", orgHandler.LeaveOrg)
	r.Post("/orgs/{org}/teams", orgHandler.CreateTeam)
	r.Get("/orgs/{org}/teams", orgHandler.ListTeams)
	r.Delete("/orgs/{org}/teams/{teamID}", orgHandler.DeleteTeam)
	r.Post("/orgs/{org}/teams/{teamID}/members", orgHandler.AddTeamMember)
	r.Delete("/orgs/{org}/teams/{teamID}/members/{userID}", orgHandler.RemoveTeamMember)
	r.Get("/orgs/{org}/teams/{teamID}/members", orgHandler.ListTeamMembers)
	r.Post("/orgs/{org}/teams/{teamID}/repos", orgHandler.AddTeamRepo)
	r.Delete("/orgs/{org}/teams/{teamID}/repos/{teamRepoID}", orgHandler.RemoveTeamRepo)
	r.Get("/orgs/{org}/teams/{teamID}/repos", orgHandler.ListTeamRepos)
	// Admin (site-admin only)
	// Issues
	r.Get("/repos/{owner}/{name}/issues", issueHandler.ListIssues)
	r.Post("/repos/{owner}/{name}/issues", issueHandler.CreateIssue)
	r.Get("/repos/{owner}/{name}/issues/{number}", issueHandler.GetIssue)
	r.Patch("/repos/{owner}/{name}/issues/{number}", issueHandler.UpdateIssue)
	r.Post("/repos/{owner}/{name}/issues/{number}/close", issueHandler.CloseIssue)
	r.Post("/repos/{owner}/{name}/issues/{number}/reopen", issueHandler.ReopenIssue)
	r.Get("/repos/{owner}/{name}/issues/{number}/comments", issueHandler.ListComments)
	r.Post("/repos/{owner}/{name}/issues/{number}/comments", issueHandler.AddComment)
	// Labels
	r.Get("/repos/{owner}/{name}/labels", labelHandler.ListLabels)
	r.Post("/repos/{owner}/{name}/labels", labelHandler.CreateLabel)
	r.Delete("/repos/{owner}/{name}/labels/{labelID}", labelHandler.DeleteLabel)
	// Milestones
	r.Get("/repos/{owner}/{name}/milestones", milestoneHandler.ListMilestones)
	r.Post("/repos/{owner}/{name}/milestones", milestoneHandler.CreateMilestone)
	r.Post("/repos/{owner}/{name}/milestones/close", milestoneHandler.CloseMilestone)
	// Kanban
	r.Get("/repos/{owner}/{name}/kanban", kanbanHandler.GetColumns)
	r.Put("/repos/{owner}/{name}/kanban/move", kanbanHandler.MoveIssue)
	// User tokens (PAT)
	r.Get("/user/tokens", patHandler.ListTokens)
	r.Post("/user/tokens", patHandler.CreateToken)
	r.Delete("/user/tokens/{tokenID}", patHandler.DeleteToken)
	// Webhooks (repo)
	r.Get("/repos/{owner}/{name}/hooks", webhookHandler.ListWebhooks)
	r.Post("/repos/{owner}/{name}/hooks", webhookHandler.CreateWebhook)
	r.Delete("/repos/{owner}/{name}/hooks/{hookID}", webhookHandler.DeleteWebhook)
	r.Get("/repos/{owner}/{name}/hooks/{hookID}/deliveries", webhookHandler.ListDeliveries)
	r.Post("/repos/{owner}/{name}/hooks/{hookID}/deliveries/{deliveryID}", webhookHandler.Redeliver)
	// Actions
	r.Get("/repos/{owner}/{name}/actions/workflows", actionsHandler.ListWorkflows)
	r.Get("/repos/{owner}/{name}/actions/workflows/{workflowID}/runs", actionsHandler.ListRuns)
	r.Post("/repos/{owner}/{name}/actions/workflows/{workflowID}/dispatch", actionsHandler.DispatchWorkflow)
	r.Get("/repos/{owner}/{name}/actions/runs/{runID}", actionsHandler.GetRun)
	r.Get("/repos/{owner}/{name}/actions/runs/{runID}/logs", actionsHandler.GetRunLogs)
	r.Post("/repos/{owner}/{name}/actions/runs/{runID}/cancel", actionsHandler.CancelRun)

	// Pull Requests
	r.Get("/repos/{owner}/{name}/pulls", prHandler.ListPRs)
	r.Post("/repos/{owner}/{name}/pulls", prHandler.CreatePR)
	r.Get("/repos/{owner}/{name}/pulls/{number}", prHandler.GetPR)
	r.Get("/repos/{owner}/{name}/pulls/{number}/diff", prHandler.GetPRDiff)
	r.Post("/repos/{owner}/{name}/pulls/{number}/merge", prHandler.MergePR)
	r.Post("/repos/{owner}/{name}/pulls/{number}/close", prHandler.ClosePR)
	r.Get("/repos/{owner}/{name}/pulls/{number}/reviews", prHandler.ListReviews)
	r.Post("/repos/{owner}/{name}/pulls/{number}/reviews", prHandler.CreateReview)

	// Admin stats — available to all authenticated users; handler restricts sensitive fields.
	r.Get("/admin/stats", adminHandler.GetStats)
	// Admin user management (site-admin only).
	r.Route("/admin", func(r chi.Router) {
		r.Use(handlers.RequireAdmin)
		r.Get("/users", userHandler.ListUsers)
		r.Post("/users", userHandler.CreateUser)
		r.Get("/users/{id}", userHandler.GetUser)
		r.Patch("/users/{id}", userHandler.UpdateUser)
		r.Delete("/users/{id}", userHandler.DeleteUser)
	})
	})
	// Runner API (external — runners authenticate via runner token).
	r.Post("/api/v1/actions/runners/register", actionsHandler.RunnerRegister)
	r.Post("/api/v1/actions/runners/poll", actionsHandler.RunnerPoll)
	r.Post("/api/v1/actions/runs/{runID}/status", actionsHandler.RunnerUpdateStatus)
	r.Post("/api/v1/actions/runs/{runID}/logs", actionsHandler.RunnerPostLog)
	r.Post("/api/v1/actions/runners/heartbeat", actionsHandler.RunnerHeartbeat)

	// Docker/OCI Package Registry (v2 protocol stubs — Phase 6).
	r.Route("/v2", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
		})
		r.Head("/{owner}/{name}/blobs/{digest}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		r.Head("/{owner}/{name}/manifests/{ref}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	})

	// Other Package Registry protocol stubs.
	r.Get("/{owner}/-/packages", pkgHandler.ListPackages)
	r.Get("/{owner}/npm/{package}", pkgHandler.ServeNpm)
	r.Post("/{owner}/pypi/", pkgHandler.ServePypiUpload)
	r.Get("/{owner}/pypi/simple/", pkgHandler.ServePypiSimple)
	r.Put("/{owner}/maven2/{path:.*}", pkgHandler.ServeMavenUpload)
	r.Get("/{owner}/maven2/{path:.*}", pkgHandler.ServeMavenFetch)
	r.Put("/{owner}/nuget/", pkgHandler.ServeNuGetPush)
	r.Get("/{owner}/helm/index.yaml", pkgHandler.ServeHelmIndex)
	r.Post("/{owner}/helm/api/charts", pkgHandler.ServeHelmUpload)

	// Internal endpoints (hook IPC, loopback-protected).
	r.Post("/internal/pre-receive", internalHandler.PreReceive)
	r.Post("/internal/post-receive", internalHandler.PostReceive)

	r.NotFound(handlers.StaticHandler().ServeHTTP)

	return r
}

// ensureDefaultAdmin creates a bootstrap admin account on the first run
// when no users exist in the database. The password must be provided via
// the DEFAULT_ADMIN_PASSWORD environment variable.
func ensureDefaultAdmin(db *gorm.DB, cfg *config.Config) error {
	store := handlers.NewGormUserStore(db)
	exists, err := store.Exists(context.Background())
	if err != nil {
		return fmt.Errorf("check users: %w", err)
	}
	if exists {
		return nil
	}

	if cfg.DefaultAdminPassword == "" {
		return fmt.Errorf("DEFAULT_ADMIN_PASSWORD is required on first startup (no users in database)")
	}

	hash, err := auth.HashPassword(cfg.DefaultAdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin := &models.User{
		Username:       cfg.DefaultAdminUsername,
		Email:          cfg.DefaultAdminUsername + "@localhost",
		PasswordHash:   hash,
		IsSiteAdmin:    true,
		IsDefaultAdmin: true,
		SetupCompleted: false,
	}

	if err := store.Create(context.Background(), admin); err != nil {
		return fmt.Errorf("create default admin: %w", err)
	}

	slog.Info("default admin account created", "username", admin.Username)
	return nil
}
