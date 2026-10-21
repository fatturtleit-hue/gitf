package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os/exec"

	"gitf/internal/middleware"
)

// Server-side git smart-HTTP subprocess runner.
//
// Security invariant: the git argv is fully literal — the repository is
// selected via cmd.Dir, never as a command argument, and exec.Command never
// invokes a shell, so no request-derived string can introduce flags or
// metacharacters. Callers stream output straight to the ResponseWriter AFTER
// headers are sent; failures are logged with request_id + stderr.

var errGitRPC = errors.New("git rpc failed")

// gitSubprocess runs cmd, streams stdout to w and logs failures. The request
// context cancels the subprocess: when the client goes away the child git
// process is killed, mirroring exec.CommandContext semantics.
func gitSubprocess(ctx context.Context, w http.ResponseWriter, stdin io.Reader, cmd *exec.Cmd) error {
	cmd.Env = baseEnv
	var stderr bytes.Buffer
	if stdin != nil {
		cmd.Stdin = stdin
	}
	cmd.Stdout = w
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		slog.ErrorContext(ctx, "git rpc start failed", "error", err, "request_id", middleware.GetRequestID(ctx))
		return errGitRPC
	}
	watchDone := make(chan struct{})
	defer close(watchDone)
	go func() {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		case <-watchDone:
		}
	}()
	if err := cmd.Wait(); err != nil {
		slog.ErrorContext(ctx, "git rpc failed",
			"error", err,
			"stderr", stderr.String(),
			"request_id", middleware.GetRequestID(ctx),
		)
		return errGitRPC
	}
	return nil
}

func runGitUploadPack(r *http.Request, w http.ResponseWriter, repoPath string, advertiseRefs bool) error {
	// The stateless RPC variant streams the client's request body (want/have
	// lines) into upload-pack via stdin; the advertise variant needs none.
	var stdin io.Reader
	if !advertiseRefs {
		stdin = r.Body
	}
	var cmd *exec.Cmd
	if advertiseRefs {
		cmd = exec.Command("git", "upload-pack", "--stateless-rpc", "--advertise-refs", ".")
	} else {
		cmd = exec.Command("git", "upload-pack", "--stateless-rpc", ".")
	}
	cmd.Dir = repoPath
	return gitSubprocess(r.Context(), w, stdin, cmd)
}

func runGitReceivePack(r *http.Request, w http.ResponseWriter, repoPath string) error {
	cmd := exec.Command("git", "receive-pack", "--stateless-rpc", ".")
	cmd.Dir = repoPath
	return gitSubprocess(r.Context(), w, r.Body, cmd)
}
