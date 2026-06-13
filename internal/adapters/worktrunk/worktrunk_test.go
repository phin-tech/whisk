package worktrunk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseVersionLine(t *testing.T) {
	tests := map[string]string{
		"wt 0.44.0\n":                    "0.44.0",
		"worktrunk v0.45.1\n":            "0.45.1",
		"worktrunk 0.46.2 (abcdef)\n":    "0.46.2",
		"prefix not-semver 10.20.30 end": "10.20.30",
	}
	for input, want := range tests {
		got, ok := ParseVersionLine(input)
		if !ok {
			t.Fatalf("ParseVersionLine(%q) did not parse", input)
		}
		if got.String() != want {
			t.Fatalf("ParseVersionLine(%q) = %s, want %s", input, got, want)
		}
	}
	if _, ok := ParseVersionLine("not a version"); ok {
		t.Fatalf("ParseVersionLine parsed invalid input")
	}
}

func TestDetectWtConfig(t *testing.T) {
	repo := t.TempDir()
	if DetectWTConfig(repo) {
		t.Fatalf("DetectWTConfig returned true before config existed")
	}
	if err := os.MkdirAll(filepath.Join(repo, ".config"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".config", "wt.toml"), []byte(""), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if !DetectWTConfig(repo) {
		t.Fatalf("DetectWTConfig returned false after config existed")
	}
}

func TestDetectWtUsesOverrideAndVersionFloor(t *testing.T) {
	ctx := context.Background()
	bin := filepath.Join(t.TempDir(), "wt")
	if err := os.WriteFile(bin, []byte(""), 0o755); err != nil {
		t.Fatalf("write wt: %v", err)
	}
	runner := &fakeRunner{
		outputs: []Output{{StatusCode: 0, Stdout: []byte("wt 0.44.0\n")}},
	}

	detected, ok, err := Detect(ctx, runner, DetectOptions{OverridePath: " " + bin + " "})
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if !ok {
		t.Fatalf("Detect did not find wt")
	}
	if detected.Path != bin || detected.Version != "0.44.0" {
		t.Fatalf("detected = %#v", detected)
	}
	if len(runner.lookups) != 0 {
		t.Fatalf("override path should skip PATH lookup: %#v", runner.lookups)
	}
	if !reflect.DeepEqual(runner.commands[0].Args, []string{"--version"}) {
		t.Fatalf("version command args = %#v", runner.commands[0].Args)
	}

	runner = &fakeRunner{outputs: []Output{{StatusCode: 0, Stdout: []byte("wt 0.43.9\n")}}}
	_, ok, err = Detect(ctx, runner, DetectOptions{OverridePath: bin})
	if err != nil {
		t.Fatalf("Detect below floor error: %v", err)
	}
	if ok {
		t.Fatalf("Detect accepted version below floor")
	}

	dir := t.TempDir()
	runner = &fakeRunner{}
	_, ok, err = Detect(ctx, runner, DetectOptions{OverridePath: dir})
	if err != nil {
		t.Fatalf("Detect directory override error: %v", err)
	}
	if ok || len(runner.commands) != 0 {
		t.Fatalf("Detect accepted directory override, ok=%v commands=%#v", ok, runner.commands)
	}

	runner = &fakeRunner{outputs: []Output{{StatusCode: 1, Stderr: []byte("bad version")}}}
	_, ok, err = Detect(ctx, runner, DetectOptions{OverridePath: bin})
	if err != nil {
		t.Fatalf("Detect nonzero version error: %v", err)
	}
	if ok {
		t.Fatalf("Detect accepted nonzero version command")
	}
}

func TestDetectWtUsesPathLookup(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		lookupPath: "/usr/local/bin/wt",
		outputs:    []Output{{StatusCode: 0, Stdout: []byte("worktrunk v0.45.0\n")}},
	}

	detected, ok, err := Detect(ctx, runner, DetectOptions{})
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if !ok || detected.Path != "/usr/local/bin/wt" || detected.Version != "0.45.0" {
		t.Fatalf("detected = %#v ok=%v", detected, ok)
	}
	if !reflect.DeepEqual(runner.lookups, []string{"wt"}) {
		t.Fatalf("lookups = %#v", runner.lookups)
	}
}

func TestListRunsFullJSONList(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{{
			StatusCode: 0,
			Stdout: []byte(`[
				{"branch":"main","path":"/repo","kind":"main","is_current":true},
				{"branch":"feature","path":"/repo/.worktrees/feature","kind":"worktree"}
			]`),
		}},
	}
	client := NewClient(Binary{Path: "/bin/wt", Version: "0.44.0"}, runner)

	items, err := client.List(ctx, "/repo")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 2 || items[1].Branch != "feature" || items[1].Path != "/repo/.worktrees/feature" {
		t.Fatalf("items = %#v", items)
	}
	wantArgs := []string{"list", "--full", "--format=json"}
	if !reflect.DeepEqual(runner.commands[0].Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.commands[0].Args, wantArgs)
	}
	if runner.commands[0].Dir != "/repo" {
		t.Fatalf("dir = %q", runner.commands[0].Dir)
	}
}

func TestListAcceptsBooleanWorkingTreeCounts(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{{
			StatusCode: 0,
			Stdout: []byte(`[
				{
					"branch":"feature",
					"path":"/repo/.worktrees/feature",
					"kind":"worktree",
					"working_tree":{
						"dirty":true,
						"untracked":false,
						"modified":true,
						"deleted":true
					}
				}
			]`),
		}},
	}
	client := NewClient(Binary{Path: "/bin/wt", Version: "0.44.0"}, runner)

	items, err := client.List(ctx, "/repo")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 || !items[0].WorkingTree.Dirty || items[0].WorkingTree.Modified != 1 || items[0].WorkingTree.Deleted != 1 || items[0].WorkingTree.Untracked != 0 {
		t.Fatalf("items = %#v", items)
	}
}

func TestListAcceptsStringSymbols(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{{
			StatusCode: 0,
			Stdout: []byte(`[
				{
					"branch":"feature",
					"path":"/repo/.worktrees/feature",
					"kind":"worktree",
					"symbols":"*"
				}
			]`),
		}},
	}
	client := NewClient(Binary{Path: "/bin/wt", Version: "0.44.0"}, runner)

	items, err := client.List(ctx, "/repo")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 || !reflect.DeepEqual([]string(items[0].Symbols), []string{"*"}) {
		t.Fatalf("items = %#v", items)
	}
}

func TestListAcceptsPartialWorktreeJSON(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{{
			StatusCode: 0,
			Stdout: []byte(`[
				{
					"branch":"feature",
					"path":"/repo/.worktrees/feature",
					"kind":"worktree",
					"worktree":null,
					"unknown_future_field":{"nested":true}
				},
				{
					"branch":"missing-path",
					"kind":"worktree",
					"working_tree":{"dirty":false}
				}
			]`),
		}},
	}
	client := NewClient(Binary{Path: "/bin/wt", Version: "0.44.0"}, runner)

	items, err := client.List(ctx, "/repo")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 2 || items[0].Branch != "feature" || items[0].Path != "/repo/.worktrees/feature" {
		t.Fatalf("items = %#v", items)
	}
	if items[0].Worktree.Locked || items[1].Path != "" {
		t.Fatalf("partial items parsed incorrectly: %#v", items)
	}
}

func TestListMapsExitAndParseErrors(t *testing.T) {
	ctx := context.Background()
	client := NewClient(Binary{Path: "/bin/wt"}, &fakeRunner{
		outputs: []Output{{StatusCode: 2, Stderr: []byte("boom")}},
	})
	_, err := client.List(ctx, "/repo")
	var exitError *ExitError
	if !errors.As(err, &exitError) || exitError.StatusCode != 2 || exitError.Stderr != "boom" {
		t.Fatalf("List exit error = %T %[1]v", err)
	}

	client = NewClient(Binary{Path: "/bin/wt"}, &fakeRunner{
		outputs: []Output{{StatusCode: 0, Stdout: []byte(`{`)}},
	})
	_, err = client.List(ctx, "/repo")
	var parseError *ParseError
	if !errors.As(err, &parseError) {
		t.Fatalf("List parse error = %T %[1]v", err)
	}
	if parseError.Unwrap() == nil || parseError.Error() == "" {
		t.Fatalf("parse error helpers returned empty values")
	}
}

func TestCreateReturnsExistingBranchWorktree(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{{
			StatusCode: 0,
			Stdout:     []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`),
		}},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	path, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if path != "/repo/.worktrees/feature" {
		t.Fatalf("path = %q", path)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %#v", runner.commands)
	}
}

func TestCreateIgnoresListedBranchWithoutPath(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature"}]`)},
			{StatusCode: 0},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	path, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if path != "/repo/.worktrees/feature" {
		t.Fatalf("path = %q", path)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"switch", "--create", "--no-cd", "feature"}) {
		t.Fatalf("create args = %#v", runner.commands[1].Args)
	}
}

func TestCreateValidatesBranchAndReportsMissingAfterSuccess(t *testing.T) {
	ctx := context.Background()
	client := NewClient(Binary{Path: "/bin/wt"}, &fakeRunner{})
	if _, err := client.Create(ctx, CreateRequest{RepoPath: "/repo"}); err == nil {
		t.Fatalf("Create accepted empty branch")
	}

	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[]`)},
			{StatusCode: 0},
			{StatusCode: 0, Stdout: []byte(`[]`)},
		},
	}
	client = NewClient(Binary{Path: "/bin/wt"}, runner)
	_, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Create missing branch error = %T %[1]v", err)
	}
	if notFound.Error() == "" {
		t.Fatalf("not found error string is empty")
	}
}

func TestCreateSwitchesAndRelistsForCreatedPath(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
			{StatusCode: 0, Stdout: []byte("created feature")},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	path, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature", Base: "main"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if path != "/repo/.worktrees/feature" {
		t.Fatalf("path = %q", path)
	}
	wantArgs := []string{"switch", "--create", "--no-cd", "--base", "main", "feature"}
	if !reflect.DeepEqual(runner.commands[1].Args, wantArgs) {
		t.Fatalf("switch args = %#v, want %#v", runner.commands[1].Args, wantArgs)
	}
}

func TestCreateSwitchesExistingBranchWhenCreateReportsBranchExists(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
			{StatusCode: 1, Stderr: []byte("✗ Branch whisk/test-project-1-test already exists\n↳ To switch to the existing branch, run without --create: wt switch whisk/test-project-1-test")},
			{StatusCode: 0, Stdout: []byte("switched")},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"whisk/test-project-1-test","path":"/repo/.worktrees/test"}]`)},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	path, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "whisk/test-project-1-test", Base: "main"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if path != "/repo/.worktrees/test" {
		t.Fatalf("path = %q", path)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"switch", "--create", "--no-cd", "--base", "main", "whisk/test-project-1-test"}) {
		t.Fatalf("create args = %#v", runner.commands[1].Args)
	}
	if !reflect.DeepEqual(runner.commands[2].Args, []string{"switch", "--no-cd", "whisk/test-project-1-test"}) {
		t.Fatalf("fallback args = %#v", runner.commands[2].Args)
	}
}

func TestCreateExistingBranchFallbackReportsRelistFailure(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
			{StatusCode: 1, Stderr: []byte("Branch feature already exists")},
			{StatusCode: 0, Stdout: []byte("switched")},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	_, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Create error = %T %[1]v, want NotFoundError", err)
	}
	if !reflect.DeepEqual(runner.commands[2].Args, []string{"switch", "--no-cd", "feature"}) {
		t.Fatalf("fallback args = %#v", runner.commands[2].Args)
	}
}

func TestCreateExistingBranchFallbackPreservesFallbackFailure(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
			{StatusCode: 1, Stderr: []byte("Branch feature already exists")},
			{StatusCode: 2, Stderr: []byte("cannot switch: dirty worktree")},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	_, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	var exitError *ExitError
	if !errors.As(err, &exitError) || exitError.StatusCode != 2 || exitError.Stderr != "cannot switch: dirty worktree" {
		t.Fatalf("Create error = %T %[1]v", err)
	}
	if len(runner.commands) != 3 {
		t.Fatalf("commands = %#v", runner.commands)
	}
}

func TestCreateDoesNotFallbackForUnrelatedAlreadyExistsError(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"main","path":"/repo"}]`)},
			{StatusCode: 1, Stderr: []byte("Branch other already exists")},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	_, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature"})
	var exitError *ExitError
	if !errors.As(err, &exitError) || exitError.Stderr != "Branch other already exists" {
		t.Fatalf("Create error = %T %[1]v", err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("unexpected fallback commands = %#v", runner.commands)
	}
}

func TestCreatePassesEnvironmentToSwitchAndRelist(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 1, Stderr: []byte("list unavailable")},
			{StatusCode: 0},
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	env := map[string]string{"HOME": "/tmp/whisk-home"}
	_, err := client.Create(ctx, CreateRequest{RepoPath: "/repo", Branch: "feature", Env: env})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	env["HOME"] = "/mutated"
	if runner.commands[1].Env["HOME"] != "/tmp/whisk-home" || runner.commands[2].Env["HOME"] != "/tmp/whisk-home" {
		t.Fatalf("env was not cloned into commands: %#v", runner.commands)
	}
}

func TestRemoveMapsPathToBranchAndKeepsBranchByDefault(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
			{StatusCode: 0},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	if err := client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/feature"}); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	wantArgs := []string{"remove", "--no-delete-branch", "feature"}
	if !reflect.DeepEqual(runner.commands[1].Args, wantArgs) {
		t.Fatalf("remove args = %#v, want %#v", runner.commands[1].Args, wantArgs)
	}
}

func TestRemoveReportsMissingAndDetachedWorktrees(t *testing.T) {
	ctx := context.Background()
	client := NewClient(Binary{Path: "/bin/wt"}, &fakeRunner{
		outputs: []Output{{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)}},
	})
	err := client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/missing"})
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Remove missing error = %T %[1]v", err)
	}

	client = NewClient(Binary{Path: "/bin/wt"}, &fakeRunner{
		outputs: []Output{{StatusCode: 0, Stdout: []byte(`[{"path":"/repo/.worktrees/detached"}]`)}},
	})
	err = client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/detached"})
	if !errors.As(err, &notFound) {
		t.Fatalf("Remove detached error = %T %[1]v", err)
	}
}

func TestRemoveRejectsMainAndCurrentWorktreesBeforeCallingWtRemove(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		json string
	}{
		{
			name: "main",
			json: `[{"branch":"main","path":"/repo","is_main":true}]`,
		},
		{
			name: "current",
			json: `[{"branch":"feature","path":"/repo/.worktrees/feature","is_current":true}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				outputs: []Output{{StatusCode: 0, Stdout: []byte(tt.json)}},
			}
			client := NewClient(Binary{Path: "/bin/wt"}, runner)

			err := client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: firstPathForRemove(tt.name)})
			var protected *ProtectedWorktreeError
			if !errors.As(err, &protected) {
				t.Fatalf("Remove error = %T %[1]v, want ProtectedWorktreeError", err)
			}
			if len(runner.commands) != 1 {
				t.Fatalf("remove command should not run: %#v", runner.commands)
			}
		})
	}
}

func firstPathForRemove(name string) string {
	if name == "main" {
		return "/repo"
	}
	return "/repo/.worktrees/feature"
}

func TestRemoveMatchesCleanedWorktreePath(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
			{StatusCode: 0},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	if err := client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/feature/."}); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"remove", "--no-delete-branch", "feature"}) {
		t.Fatalf("remove args = %#v", runner.commands[1].Args)
	}
}

func TestRemoveCanDeleteBranchAndForceWithoutMappingDirtyError(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{
		outputs: []Output{
			{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
			{StatusCode: 1, Stderr: []byte("uncommitted changes")},
		},
	}
	client := NewClient(Binary{Path: "/bin/wt"}, runner)

	err := client.Remove(ctx, RemoveRequest{
		RepoPath:     "/repo",
		WorktreePath: "/repo/.worktrees/feature",
		AlsoBranch:   true,
		Force:        true,
	})
	var exitError *ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("Remove force error = %T %[1]v", err)
	}
	wantArgs := []string{"remove", "--force", "feature"}
	if !reflect.DeepEqual(runner.commands[1].Args, wantArgs) {
		t.Fatalf("remove args = %#v, want %#v", runner.commands[1].Args, wantArgs)
	}
}

func TestRemoveMapsLockedAndDirtyFailures(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		err  string
		want string
	}{
		{name: "locked", err: "worktree is locked by hook", want: "locked"},
		{name: "dirty", err: "refusing: uncommitted changes", want: "dirty"},
		{name: "dirty uppercase", err: "Refusing: DIRTY worktree", want: "dirty"},
		{name: "local changes", err: "cannot remove with local changes", want: "dirty"},
		{name: "locked by", err: "Locked by another process", want: "locked"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				outputs: []Output{
					{StatusCode: 0, Stdout: []byte(`[{"branch":"feature","path":"/repo/.worktrees/feature"}]`)},
					{StatusCode: 1, Stderr: []byte(tt.err)},
				},
			}
			client := NewClient(Binary{Path: "/bin/wt"}, runner)

			err := client.Remove(ctx, RemoveRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/feature"})
			switch tt.want {
			case "locked":
				var locked *LockedError
				if !errors.As(err, &locked) {
					t.Fatalf("Remove error = %T %[1]v, want LockedError", err)
				}
			case "dirty":
				var dirty *DirtyError
				if !errors.As(err, &dirty) {
					t.Fatalf("Remove error = %T %[1]v, want DirtyError", err)
				}
			}
		})
	}
}

func TestOSRunnerRunCapturesSuccessExitErrorAndEnv(t *testing.T) {
	ctx := context.Background()
	runner := OSRunner{}
	output, err := runner.Run(ctx, Command{
		Path: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "ok"},
		Env:  map[string]string{"WHISK_HELPER_VALUE": "env-ok"},
	})
	if err != nil {
		t.Fatalf("Run success error: %v", err)
	}
	if output.StatusCode != 0 || string(output.Stdout) != "env-ok" {
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
	if _, err := runner.LookPath(os.Args[0]); err != nil {
		t.Fatalf("LookPath error: %v", err)
	}
}

func TestErrorStrings(t *testing.T) {
	errs := []error{
		&ExitError{StatusCode: 1, Stderr: "x"},
		&LockedError{Reason: "locked"},
		&DirtyError{Reason: "dirty"},
		&ProtectedWorktreeError{Path: "/repo", Reason: "main worktree"},
	}
	for _, err := range errs {
		if err.Error() == "" {
			t.Fatalf("empty error string for %T", err)
		}
	}
}

func TestHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			switch os.Args[i+1] {
			case "ok":
				os.Stdout.WriteString(os.Getenv("WHISK_HELPER_VALUE"))
				os.Exit(0)
			case "fail":
				os.Stderr.WriteString("bad")
				os.Exit(7)
			}
		}
	}
}

type fakeRunner struct {
	lookupPath string
	lookupErr  error
	lookups    []string
	outputs    []Output
	commands   []Command
	runErr     error
}

func (r *fakeRunner) LookPath(file string) (string, error) {
	r.lookups = append(r.lookups, file)
	if r.lookupErr != nil {
		return "", r.lookupErr
	}
	return r.lookupPath, nil
}

func (r *fakeRunner) Run(_ context.Context, command Command) (Output, error) {
	r.commands = append(r.commands, cloneCommand(command))
	if r.runErr != nil {
		return Output{}, r.runErr
	}
	if len(r.outputs) == 0 {
		return Output{}, nil
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return output, nil
}

func cloneCommand(command Command) Command {
	cloned := Command{
		Path: command.Path,
		Dir:  command.Dir,
		Args: append([]string(nil), command.Args...),
	}
	if command.Env != nil {
		cloned.Env = map[string]string{}
		for key, value := range command.Env {
			cloned.Env[key] = value
		}
	}
	return cloned
}
