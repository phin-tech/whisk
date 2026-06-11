package client_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesDaemonWorktreeAPI(t *testing.T) {
	backend := &fakeWorktreeBackend{
		status: app.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      app.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []app.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature", Kind: "worktree"}},
		created:   app.CreatedWorktree{Path: "/repo/.worktrees/created"},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Worktrees: backend})

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := daemon.DetectWorktrunk(ctx, protocol.DetectWorktrunkRequest{RepoPath: "/repo", OverridePath: "/custom/wt"})
	if err != nil {
		t.Fatalf("detect worktrunk: %v", err)
	}
	if !status.Available || !status.ConfigFound || status.Binary.Path != "/bin/wt" {
		t.Fatalf("status = %#v", status)
	}
	if backend.detectReq.RepoPath != "/repo" || backend.detectReq.OverridePath != "/custom/wt" {
		t.Fatalf("detect req = %#v", backend.detectReq)
	}

	worktrees, err := daemon.ListWorktrees(ctx, protocol.ListWorktreesRequest{RepoPath: "/repo"})
	if err != nil {
		t.Fatalf("list worktrees: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0].Branch != "feature" {
		t.Fatalf("worktrees = %#v", worktrees)
	}
	if backend.listReq.RepoPath != "/repo" {
		t.Fatalf("list req = %#v", backend.listReq)
	}

	created, err := daemon.CreateWorktree(ctx, protocol.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	})
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}
	if created.Path != "/repo/.worktrees/created" {
		t.Fatalf("created = %#v", created)
	}
	if backend.createReq.Branch != "created" || backend.createReq.Base != "main" {
		t.Fatalf("create req = %#v", backend.createReq)
	}

	if err := daemon.RemoveWorktree(ctx, protocol.RemoveWorktreeRequest{
		RepoPath:     "/repo",
		WorktreePath: "/repo/.worktrees/created",
		AlsoBranch:   false,
		Force:        false,
	}); err != nil {
		t.Fatalf("remove worktree: %v", err)
	}
	if backend.removeReq.WorktreePath != "/repo/.worktrees/created" || backend.removeReq.AlsoBranch || backend.removeReq.Force {
		t.Fatalf("remove req = %#v", backend.removeReq)
	}
}

type fakeWorktreeBackend struct {
	status    app.WorktrunkStatus
	worktrees []app.Worktree
	created   app.CreatedWorktree

	detectReq app.DetectWorktrunkRequest
	listReq   app.ListWorktreesRequest
	createReq app.CreateWorktreeRequest
	removeReq app.RemoveWorktreeRequest
}

func (b *fakeWorktreeBackend) DetectWorktrunk(_ context.Context, req app.DetectWorktrunkRequest) (app.WorktrunkStatus, error) {
	b.detectReq = req
	return b.status, nil
}

func (b *fakeWorktreeBackend) ListWorktrees(_ context.Context, req app.ListWorktreesRequest) ([]app.Worktree, error) {
	b.listReq = req
	return b.worktrees, nil
}

func (b *fakeWorktreeBackend) CreateWorktree(_ context.Context, req app.CreateWorktreeRequest) (app.CreatedWorktree, error) {
	b.createReq = req
	return b.created, nil
}

func (b *fakeWorktreeBackend) RemoveWorktree(_ context.Context, req app.RemoveWorktreeRequest) error {
	b.removeReq = req
	return nil
}
