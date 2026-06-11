package gitcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStatusBuildsRepoReadModel(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{
		{StatusCode: 0, Stdout: []byte("true\n")},
		{StatusCode: 0, Stdout: []byte("feature\n")},
		{StatusCode: 0, Stdout: []byte(" M main.go\n")},
		{StatusCode: 0, Stdout: []byte("origin/feature\n")},
		{StatusCode: 0, Stdout: []byte("3\t7\n")},
		{StatusCode: 0, Stdout: []byte("origin/main\n")},
		{StatusCode: 0, Stdout: []byte("12\n")},
	}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	status, err := client.Status(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.Branch != "feature" || !status.Dirty || status.Ahead != 3 || status.Behind != 7 || status.BehindDefault != 12 {
		t.Fatalf("status = %#v", status)
	}
	if status.RemoteState != RemoteDiverged || status.DefaultBranch != "main" || status.TrackingBranch != "origin/feature" {
		t.Fatalf("remote status = %#v", status)
	}
	want := []string{"rev-list", "--left-right", "--count", "HEAD...@{u}"}
	if !reflect.DeepEqual(runner.commands[4].Args, want) {
		t.Fatalf("ahead/behind args = %#v", runner.commands[4].Args)
	}
}

func TestStatusWithoutTrackingBranchIsUnknown(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{
		{StatusCode: 0, Stdout: []byte("true\n")},
		{StatusCode: 0, Stdout: []byte("main\n")},
		{StatusCode: 0, Stdout: []byte("")},
		{StatusCode: 1, Stderr: []byte("no upstream")},
		{StatusCode: 0, Stdout: []byte("origin/main\n")},
	}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	status, err := client.Status(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.RemoteState != RemoteUnknown || status.Ahead != 0 || status.Behind != 0 || status.Dirty {
		t.Fatalf("status = %#v", status)
	}
}

func TestCloneRepoBuildsSingleBranchCloneAndCleansFailedTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "checkout")
	runner := &fakeRunner{outputs: []Output{{StatusCode: 42, Stderr: []byte("nope")}}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	err := client.CloneRepo(context.Background(), CloneRequest{
		URL:    "git@example.com:repo.git",
		Branch: "main",
		Target: target,
	})
	var exit *ExitError
	if !errors.As(err, &exit) || exit.StatusCode != 42 {
		t.Fatalf("CloneRepo error = %T %[1]v", err)
	}
	want := []string{"clone", "--branch", "main", "--single-branch", "git@example.com:repo.git", target}
	if !reflect.DeepEqual(runner.commands[0].Args, want) {
		t.Fatalf("clone args = %#v", runner.commands[0].Args)
	}
}

func TestWorktreeListParsesPorcelain(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{{StatusCode: 0, Stdout: []byte("worktree /repo\nHEAD abc\nbranch refs/heads/main\n\nworktree /repo/.worktrees/feature\nHEAD def\nbranch refs/heads/feature\n")}}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	worktrees, err := client.WorktreeList(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("WorktreeList error: %v", err)
	}
	if len(worktrees) != 2 || worktrees[1].Branch != "feature" || worktrees[1].Path != "/repo/.worktrees/feature" {
		t.Fatalf("worktrees = %#v", worktrees)
	}
	if !reflect.DeepEqual(runner.commands[0].Args, []string{"worktree", "list", "--porcelain"}) {
		t.Fatalf("worktree list args = %#v", runner.commands[0].Args)
	}
}

func TestWorktreeAddAndRemoveUseConservativeDefaults(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{{StatusCode: 0}, {StatusCode: 0}}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	if err := client.WorktreeAdd(context.Background(), WorktreeAddRequest{RepoPath: "/repo", Path: "/repo/.worktrees/card", Branch: "card", StartPoint: "origin/main"}); err != nil {
		t.Fatalf("WorktreeAdd error: %v", err)
	}
	if err := client.WorktreeRemove(context.Background(), WorktreeRemoveRequest{RepoPath: "/repo", Path: "/repo/.worktrees/card"}); err != nil {
		t.Fatalf("WorktreeRemove error: %v", err)
	}
	if !reflect.DeepEqual(runner.commands[0].Args, []string{"worktree", "add", "-b", "card", "/repo/.worktrees/card", "origin/main"}) {
		t.Fatalf("add args = %#v", runner.commands[0].Args)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"worktree", "remove", "/repo/.worktrees/card"}) {
		t.Fatalf("remove args = %#v", runner.commands[1].Args)
	}
}

func TestValidationAndRemoteStateBranches(t *testing.T) {
	client := NewClient(Binary{}, &fakeRunner{})
	if client.binary.Path != "git" {
		t.Fatalf("default binary = %#v", client.binary)
	}
	if err := client.CloneRepo(context.Background(), CloneRequest{}); err == nil {
		t.Fatalf("expected missing url error")
	}
	if err := client.CloneRepo(context.Background(), CloneRequest{URL: "git@example.com:repo.git"}); err == nil {
		t.Fatalf("expected missing target error")
	}
	existing := t.TempDir()
	if err := client.CloneRepo(context.Background(), CloneRequest{URL: "git@example.com:repo.git", Target: existing}); err == nil {
		t.Fatalf("expected existing target error")
	}

	if (&ExitError{StatusCode: 1, Stderr: "x"}).Error() == "" || (&NotRepoError{Path: "/repo"}).Error() == "" {
		t.Fatalf("error strings must not be empty")
	}
	if remoteState(false, 0, 0) != RemoteUnknown || remoteState(true, 0, 0) != RemoteUpToDate ||
		remoteState(true, 1, 0) != RemoteAhead || remoteState(true, 0, 1) != RemoteBehind ||
		remoteState(true, 1, 1) != RemoteDiverged {
		t.Fatalf("remoteState branches regressed")
	}
}

func TestWorktreeAddExistingRefAndForcedRemove(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{{StatusCode: 0}, {StatusCode: 0}}}
	client := NewClient(Binary{Path: "/bin/git"}, runner)

	if err := client.WorktreeAdd(context.Background(), WorktreeAddRequest{RepoPath: "/repo", Path: "/repo/.worktrees/main", Branch: "main", ExistingRef: true}); err != nil {
		t.Fatalf("WorktreeAdd existing ref error: %v", err)
	}
	if err := client.WorktreeRemove(context.Background(), WorktreeRemoveRequest{RepoPath: "/repo", Path: "/repo/.worktrees/main", Force: true}); err != nil {
		t.Fatalf("WorktreeRemove force error: %v", err)
	}
	if !reflect.DeepEqual(runner.commands[0].Args, []string{"worktree", "add", "/repo/.worktrees/main", "main"}) {
		t.Fatalf("existing-ref add args = %#v", runner.commands[0].Args)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"worktree", "remove", "--force", "/repo/.worktrees/main"}) {
		t.Fatalf("force remove args = %#v", runner.commands[1].Args)
	}
}

func TestStatusRejectsNonRepo(t *testing.T) {
	client := NewClient(Binary{Path: "/bin/git"}, &fakeRunner{outputs: []Output{{StatusCode: 128, Stderr: []byte("not repo")}}})
	_, err := client.Status(context.Background(), "/repo")
	var notRepo *NotRepoError
	if !errors.As(err, &notRepo) {
		t.Fatalf("Status error = %T %[1]v", err)
	}
}

func TestOSRunnerRunCapturesSuccessAndExitError(t *testing.T) {
	ctx := context.Background()
	runner := OSRunner{}
	output, err := runner.Run(ctx, Command{
		Path: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "ok"},
	})
	if err != nil {
		t.Fatalf("Run success error: %v", err)
	}
	if output.StatusCode != 0 || string(output.Stdout) != "ok" {
		t.Fatalf("success output = %#v", output)
	}

	output, err = runner.Run(ctx, Command{
		Path: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "fail"},
	})
	if err != nil {
		t.Fatalf("Run failure error: %v", err)
	}
	if output.StatusCode != 7 || string(output.Stderr) != "bad" {
		t.Fatalf("failure output = %#v", output)
	}
}

func TestHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			switch os.Args[i+1] {
			case "ok":
				os.Stdout.WriteString("ok")
				os.Exit(0)
			case "fail":
				os.Stderr.WriteString("bad")
				os.Exit(7)
			}
		}
	}
}

type fakeRunner struct {
	outputs  []Output
	commands []Command
}

func (r *fakeRunner) Run(_ context.Context, command Command) (Output, error) {
	r.commands = append(r.commands, Command{Path: command.Path, Args: append([]string(nil), command.Args...), Dir: command.Dir})
	if len(r.outputs) == 0 {
		return Output{}, nil
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return output, nil
}
