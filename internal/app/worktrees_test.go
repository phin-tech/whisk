package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
)

func TestRuntimeWorktreeMethodsDelegateToBackend(t *testing.T) {
	backend := &worktreeBackendFake{
		status: app.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      app.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []app.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature"}},
		created:   app.CreatedWorktree{Path: "/repo/.worktrees/created"},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Worktrees: backend})
	ctx := context.Background()

	status, err := runtime.DetectWorktrunk(ctx, app.DetectWorktrunkRequest{RepoPath: "/repo", OverridePath: "/custom/wt"})
	if err != nil || !status.Available {
		t.Fatalf("detect = %#v, %v", status, err)
	}
	if backend.detectReq.OverridePath != "/custom/wt" {
		t.Fatalf("detect req = %#v", backend.detectReq)
	}

	worktrees, err := runtime.ListWorktrees(ctx, app.ListWorktreesRequest{RepoPath: "/repo"})
	if err != nil || len(worktrees) != 1 || worktrees[0].Branch != "feature" {
		t.Fatalf("list = %#v, %v", worktrees, err)
	}

	created, err := runtime.CreateWorktree(ctx, app.CreateWorktreeRequest{RepoPath: "/repo", Branch: "created", Base: "main"})
	if err != nil || created.Path != "/repo/.worktrees/created" {
		t.Fatalf("create = %#v, %v", created, err)
	}
	if backend.createReq.Base != "main" {
		t.Fatalf("create req = %#v", backend.createReq)
	}

	if err := runtime.RemoveWorktree(ctx, app.RemoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/created"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if backend.removeReq.WorktreePath != "/repo/.worktrees/created" {
		t.Fatalf("remove req = %#v", backend.removeReq)
	}
}

func TestRuntimeWorktreeMethodsRequireBackend(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	ctx := context.Background()

	_, err := runtime.DetectWorktrunk(ctx, app.DetectWorktrunkRequest{})
	assertWorktreeBackendRequired(t, err)
	_, err = runtime.ListWorktrees(ctx, app.ListWorktreesRequest{})
	assertWorktreeBackendRequired(t, err)
	_, err = runtime.CreateWorktree(ctx, app.CreateWorktreeRequest{})
	assertWorktreeBackendRequired(t, err)
	assertWorktreeBackendRequired(t, runtime.RemoveWorktree(ctx, app.RemoveWorktreeRequest{}))
}

func assertWorktreeBackendRequired(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "worktree backend required") {
		t.Fatalf("error = %v", err)
	}
}

type worktreeBackendFake struct {
	status    app.WorktrunkStatus
	worktrees []app.Worktree
	created   app.CreatedWorktree

	detectReq app.DetectWorktrunkRequest
	listReq   app.ListWorktreesRequest
	createReq app.CreateWorktreeRequest
	removeReq app.RemoveWorktreeRequest
}

func (b *worktreeBackendFake) DetectWorktrunk(_ context.Context, req app.DetectWorktrunkRequest) (app.WorktrunkStatus, error) {
	b.detectReq = req
	return b.status, nil
}

func (b *worktreeBackendFake) ListWorktrees(_ context.Context, req app.ListWorktreesRequest) ([]app.Worktree, error) {
	b.listReq = req
	return b.worktrees, nil
}

func (b *worktreeBackendFake) CreateWorktree(_ context.Context, req app.CreateWorktreeRequest) (app.CreatedWorktree, error) {
	b.createReq = req
	return b.created, nil
}

func (b *worktreeBackendFake) RemoveWorktree(_ context.Context, req app.RemoveWorktreeRequest) error {
	b.removeReq = req
	return nil
}
