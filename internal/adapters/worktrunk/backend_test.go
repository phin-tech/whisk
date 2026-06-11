package worktrunk

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
)

func TestBackendDetectWorktrunk(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".config"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".config", "wt.toml"), []byte(""), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	bin := filepath.Join(t.TempDir(), "wt")
	if err := os.WriteFile(bin, []byte(""), 0o755); err != nil {
		t.Fatalf("write wt: %v", err)
	}
	runner := &fakeRunner{
		outputs: []Output{{StatusCode: 0, Stdout: []byte("wt 0.44.0\n")}},
	}
	backend := NewBackend(runner)

	status, err := backend.DetectWorktrunk(ctx, app.DetectWorktrunkRequest{RepoPath: repo, OverridePath: bin})
	if err != nil {
		t.Fatalf("DetectWorktrunk error: %v", err)
	}
	if !status.Available || !status.ConfigFound || status.Binary.Path != bin || status.Binary.Version != "0.44.0" {
		t.Fatalf("status = %#v", status)
	}
}

func TestBackendListWorktrees(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		lookupPath: "/bin/wt",
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte("wt 0.44.0\n")},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature","kind":"worktree","is_main":false,"is_current":true,"working_tree":{"dirty":true},"worktree":{"locked":true}}]`)},
		},
	}
	backend := NewBackend(runner)

	worktrees, err := backend.ListWorktrees(ctx, app.ListWorktreesRequest{RepoPath: "/repo"})
	if err != nil {
		t.Fatalf("ListWorktrees error: %v", err)
	}
	if len(worktrees) != 1 ||
		worktrees[0].Branch != "feature" ||
		worktrees[0].Path != "/repo/.worktrees/feature" ||
		worktrees[0].Kind != "worktree" ||
		worktrees[0].IsMain ||
		!worktrees[0].IsCurrent ||
		!worktrees[0].Dirty ||
		!worktrees[0].Locked {
		t.Fatalf("worktrees = %#v", worktrees)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"list", "--full", "--format=json"}) {
		t.Fatalf("list args = %#v", runner.commands[1].Args)
	}
}

func TestBackendCreateAndRemoveWorktree(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		lookupPath: "/bin/wt",
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte("wt 0.44.0\n")},
			{StatusCode: 0, Stdout: []byte(`[]`)},
			{StatusCode: 0},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"created","path":"/repo/.worktrees/created"}]`)},
			{StatusCode: 0, Stdout: []byte("wt 0.44.0\n")},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"created","path":"/repo/.worktrees/created"}]`)},
			{StatusCode: 0},
		},
	}
	backend := NewBackend(runner)

	created, err := backend.CreateWorktree(ctx, app.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	})
	if err != nil {
		t.Fatalf("CreateWorktree error: %v", err)
	}
	if created.Path != "/repo/.worktrees/created" {
		t.Fatalf("created = %#v", created)
	}
	if !reflect.DeepEqual(runner.commands[2].Args, []string{"switch", "--create", "--no-cd", "--base", "main", "created"}) {
		t.Fatalf("create args = %#v", runner.commands[2].Args)
	}

	err = backend.RemoveWorktree(ctx, app.RemoveWorktreeRequest{
		RepoPath:     "/repo",
		WorktreePath: "/repo/.worktrees/created",
	})
	if err != nil {
		t.Fatalf("RemoveWorktree error: %v", err)
	}
	if !reflect.DeepEqual(runner.commands[6].Args, []string{"remove", "--no-delete-branch", "created"}) {
		t.Fatalf("remove args = %#v", runner.commands[6].Args)
	}
}
