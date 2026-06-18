package onboarding

import "testing"

func TestSelectDefaultsKeepsRiskyRowsUnchecked(t *testing.T) {
	items := SelectDefaults([]Item{
		{ID: "hook:codex", Kind: KindHook, Status: StatusMissing},
		{ID: "hook:claude", Kind: KindHook, Status: StatusModified},
		{ID: "plugin:github", Kind: KindPlugin, Status: StatusUntrusted},
	})

	if !items[0].SelectedByDefault {
		t.Fatalf("missing hook was not selected")
	}
	if items[1].SelectedByDefault {
		t.Fatalf("modified hook was selected")
	}
	if items[2].SelectedByDefault {
		t.Fatalf("plugin trust was selected")
	}
}

func TestShouldShowFirstRunAndAppliedDriftOnly(t *testing.T) {
	items := []Item{
		{ID: "skill:codex", Kind: KindSkill, Status: StatusMissing},
		{ID: "skill:claude", Kind: KindSkill, Status: StatusMissing},
	}
	if !ShouldShow(items, State{}) {
		t.Fatalf("first run should show")
	}

	state := State{
		Applied: map[string]Record{"skill:codex": {Status: StatusCurrent, Hash: "old"}},
		Skipped: map[string]bool{"skill:claude": true},
	}
	items[0].Hash = "new"
	if !ShouldShow(items, state) {
		t.Fatalf("applied drift should show")
	}
	state.Applied["skill:codex"] = Record{Status: StatusMissing}
	items[0].Hash = ""
	if ShouldShow(items, state) {
		t.Fatalf("unchanged applied status should not show")
	}
	if ShouldShow([]Item{{ID: "skill:claude", Kind: KindSkill, Status: StatusMissing}}, state) {
		t.Fatalf("skipped drift should not show")
	}
}

func TestNextStateRecordsSelectedAndSkipped(t *testing.T) {
	got := NextState(State{}, []Item{
		{ID: "skill:codex", Status: StatusCurrent, LatestVersion: "1", Hash: "abc"},
		{ID: "plugin:github", Status: StatusUntrusted},
	}, map[string]bool{"skill:codex": true}, 18, "sha")

	if got.Applied["skill:codex"].Hash != "abc" || got.Applied["skill:codex"].Version != "1" {
		t.Fatalf("applied = %#v", got.Applied)
	}
	if !got.Skipped["plugin:github"] {
		t.Fatalf("skipped = %#v", got.Skipped)
	}
	if got.DaemonAPIVersion != 18 || got.DaemonGitSHA != "sha" {
		t.Fatalf("daemon version = %d %q", got.DaemonAPIVersion, got.DaemonGitSHA)
	}
}
