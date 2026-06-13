package hooks

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseCommandsSupportsFlatAndSectionForms(t *testing.T) {
	commands, err := ParseCommands(`
post_agent = "echo user"

[pre_agent]
lint = "task lint"
`, "/config/hooks.toml", "user")
	if err != nil {
		t.Fatalf("ParseCommands error: %v", err)
	}
	if len(commands) != 2 || commands[0].Event != "post_agent" || commands[1].Name != "lint" || commands[1].Command != "task lint" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestRunUsesApprovedProjectHooksAndWorktreeCwd(t *testing.T) {
	root := t.TempDir()
	repo := t.TempDir()
	writeFile(t, filepath.Join(root, "hooks.toml"), `post_agent = "echo user"`)
	writeFile(t, filepath.Join(repo, ".config", "whisk", "hooks.toml"), `post_agent = "echo project"`)
	service := NewService(root, &fakeRunner{})
	projectCommand := Command{Event: "post_agent", Source: "project", ConfigPath: filepath.Join(repo, ".config", "whisk", "hooks.toml"), Name: "default", Command: "echo project"}
	if err := service.Approve(ApprovalID(projectCommand)); err != nil {
		t.Fatalf("Approve error: %v", err)
	}

	summary, err := service.Run(context.Background(), RunRequest{Event: "post_agent", RepoPath: repo, WorktreePath: repo})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if summary.Ran != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	runner := service.runner.(*fakeRunner)
	gotCommands := []string{runner.commands[0].Command, runner.commands[1].Command}
	if !reflect.DeepEqual(gotCommands, []string{"echo user", "echo project"}) {
		t.Fatalf("commands = %#v", runner.commands)
	}
	if runner.commands[1].Dir != repo {
		t.Fatalf("project cwd = %q", runner.commands[1].Dir)
	}
}

func TestProjectHooksAreSkippedUntilApproved(t *testing.T) {
	root := t.TempDir()
	repo := t.TempDir()
	writeFile(t, filepath.Join(root, "hooks.toml"), `post_agent = "echo user"`)
	writeFile(t, filepath.Join(repo, ".config", "whisk", "hooks.toml"), `post_agent = "echo project"`)
	service := NewService(root, &fakeRunner{})

	summary, err := service.Run(context.Background(), RunRequest{Event: "post_agent", RepoPath: repo})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	runner := service.runner.(*fakeRunner)
	if summary.Ran != 1 || runner.commands[0].Command != "echo user" {
		t.Fatalf("summary=%#v commands=%#v", summary, runner.commands)
	}
}

func TestHookValidationAndApprovalIdempotence(t *testing.T) {
	root := t.TempDir()
	service := NewService(root, nil)
	if service.runner == nil {
		t.Fatalf("default runner missing")
	}
	if _, err := service.Run(context.Background(), RunRequest{Event: "unknown"}); err == nil {
		t.Fatalf("expected unknown event error")
	}
	command := Command{Event: "post_agent", Source: "project", ConfigPath: "/repo/.config/whisk/hooks.toml", Name: "default", Command: "echo ok"}
	id := ApprovalID(command)
	if err := service.Approve(id); err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if err := service.Approve(id); err != nil {
		t.Fatalf("Approve duplicate error: %v", err)
	}
	if approvals, err := service.approvals(); err != nil || len(approvals) != 1 {
		t.Fatalf("approvals = %#v, %v", approvals, err)
	}
	if err := service.Approve(""); err == nil {
		t.Fatalf("expected empty approval error")
	}
}

func TestParseCommandsSkipsCommentsInvalidEventsAndInvalidStrings(t *testing.T) {
	commands, err := ParseCommands(`
# comment
unknown = "nope"
post_agent = bad
post_agent = "ok"
`, "/hooks.toml", "user")
	if err != nil {
		t.Fatalf("ParseCommands error: %v", err)
	}
	if len(commands) != 1 || commands[0].Command != "ok" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestShellRunnerRunsCommandInDirectoryAndReportsFailures(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	if err := (ShellRunner{}).Run(ctx, RunCommand{
		Command: "printf ok > marker",
		Dir:     dir,
	}); err != nil {
		t.Fatalf("run command: %v", err)
	}
	bytes, err := os.ReadFile(filepath.Join(dir, "marker"))
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if string(bytes) != "ok" {
		t.Fatalf("marker = %q", string(bytes))
	}
	err = (ShellRunner{}).Run(ctx, RunCommand{Command: "printf nope && exit 7", Dir: dir})
	if err == nil || !strings.Contains(err.Error(), "hook command failed") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected command failure with output, got %v", err)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := ensureParent(path); err != nil {
		t.Fatalf("ensure parent: %v", err)
	}
	if err := writeText(path, body); err != nil {
		t.Fatalf("write: %v", err)
	}
}

type fakeRunner struct {
	commands []RunCommand
}

func (r *fakeRunner) Run(_ context.Context, command RunCommand) error {
	r.commands = append(r.commands, command)
	return nil
}
