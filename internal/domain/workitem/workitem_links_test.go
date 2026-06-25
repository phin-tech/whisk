package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestStateAddsWorkItemLinksAsSeparateGraphRecords(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	dependent := mustWorkItem(t, state, "wi_dependent", project.ID)
	blocker := mustWorkItem(t, state, "wi_blocker", project.ID)

	link, err := state.AddWorkItemLink(AddWorkItemLink{
		ID:               "link_01",
		SourceWorkItemID: dependent.ID,
		TargetWorkItemID: blocker.ID,
		Type:             WorkItemLinkBlocks,
		Actor:            "agent",
		Now:              now,
	})
	if err != nil {
		t.Fatalf("add link: %v", err)
	}
	if link.ProjectID != project.ID || link.SourceWorkItemID != dependent.ID || link.TargetWorkItemID != blocker.ID || link.Type != WorkItemLinkBlocks {
		t.Fatalf("link = %#v", link)
	}
	if link.CreatedBy != "agent" || !link.CreatedAt.Equal(now) {
		t.Fatalf("link audit fields = %#v", link)
	}

	links := state.ListWorkItemLinks(dependent.ID)
	if len(links) != 1 || links[0].ID != link.ID {
		t.Fatalf("links = %#v", links)
	}
	if stored, ok := state.GetWorkItem(dependent.ID); !ok || len(stored.History) != 1 {
		t.Fatalf("link mutated work item instead of separate graph state: %#v", stored)
	}

	restored, err := NewStateFromSnapshot(state.Snapshot())
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	restoredLinks := restored.ListWorkItemLinks(dependent.ID)
	if len(restoredLinks) != 1 || restoredLinks[0].ID != link.ID {
		t.Fatalf("restored links = %#v", restoredLinks)
	}
}

func TestStateRejectsInvalidWorkItemLinksAndBlockingCycles(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	otherProject := mustProject(t, state, "proj_02", "Two")
	first := mustWorkItem(t, state, "wi_01", project.ID)
	second := mustWorkItem(t, state, "wi_02", project.ID)
	third := mustWorkItem(t, state, "wi_03", project.ID)
	other := mustWorkItem(t, state, "wi_other", otherProject.ID)

	tests := []struct {
		name string
		req  AddWorkItemLink
		want string
	}{
		{
			name: "self link",
			req:  AddWorkItemLink{ID: "link_self", SourceWorkItemID: first.ID, TargetWorkItemID: first.ID, Type: WorkItemLinkBlocks, Now: now},
			want: "cannot link work item to itself",
		},
		{
			name: "cross project",
			req:  AddWorkItemLink{ID: "link_cross", SourceWorkItemID: first.ID, TargetWorkItemID: other.ID, Type: WorkItemLinkBlocks, Now: now},
			want: "same project",
		},
		{
			name: "unknown type",
			req:  AddWorkItemLink{ID: "link_type", SourceWorkItemID: first.ID, TargetWorkItemID: second.ID, Type: "unknown", Now: now},
			want: "unsupported link type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := state.AddWorkItemLink(test.req)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}

	if _, err := state.AddWorkItemLink(AddWorkItemLink{
		ID:               "link_01",
		SourceWorkItemID: first.ID,
		TargetWorkItemID: second.ID,
		Type:             WorkItemLinkBlocks,
		Now:              now,
	}); err != nil {
		t.Fatalf("add first link: %v", err)
	}
	if _, err := state.AddWorkItemLink(AddWorkItemLink{
		ID:               "link_02",
		SourceWorkItemID: second.ID,
		TargetWorkItemID: third.ID,
		Type:             WorkItemLinkBlocks,
		Now:              now,
	}); err != nil {
		t.Fatalf("add second link: %v", err)
	}
	_, err := state.AddWorkItemLink(AddWorkItemLink{
		ID:               "link_cycle",
		SourceWorkItemID: third.ID,
		TargetWorkItemID: first.ID,
		Type:             WorkItemLinkBlocks,
		Now:              now,
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestBuildReadyWorkExplanationHonorsBlockingLinks(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	blocked := mustWorkItem(t, state, "wi_blocked", project.ID)
	blocker := mustWorkItem(t, state, "wi_blocker", project.ID)
	blocked = mustMoveWorkItem(t, state, blocked.ID, StageReady, now)
	blocker = mustMoveWorkItem(t, state, blocker.ID, StageReady, now)

	if _, err := state.AddWorkItemLink(AddWorkItemLink{
		ID:               "link_01",
		SourceWorkItemID: blocked.ID,
		TargetWorkItemID: blocker.ID,
		Type:             WorkItemLinkBlocks,
		Now:              now,
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}

	explanation := BuildReadyWorkExplanation(ReadyWorkInput{
		ProjectID: project.ID,
		WorkItems: state.ListWorkItems(project.ID),
		Links:     state.ListWorkItemLinks(""),
	})
	if explanation.Summary.TotalReady != 1 || explanation.Summary.TotalBlocked != 1 {
		t.Fatalf("summary = %#v", explanation.Summary)
	}
	if len(explanation.Ready) != 1 || explanation.Ready[0].WorkItem.ID != blocker.ID {
		t.Fatalf("ready = %#v", explanation.Ready)
	}
	if explanation.Ready[0].Reason != "no blocking dependencies" {
		t.Fatalf("ready reason = %q", explanation.Ready[0].Reason)
	}
	if len(explanation.Blocked) != 1 || explanation.Blocked[0].WorkItem.ID != blocked.ID {
		t.Fatalf("blocked = %#v", explanation.Blocked)
	}
	if len(explanation.Blocked[0].BlockedBy) != 1 || explanation.Blocked[0].BlockedBy[0].ID != blocker.ID || explanation.Blocked[0].BlockedBy[0].Title != blocker.Title {
		t.Fatalf("blocked by = %#v", explanation.Blocked[0].BlockedBy)
	}

	mustMoveWorkItem(t, state, blocker.ID, StageDone, now.Add(time.Minute))
	explanation = BuildReadyWorkExplanation(ReadyWorkInput{
		ProjectID: project.ID,
		WorkItems: state.ListWorkItems(project.ID),
		Links:     state.ListWorkItemLinks(""),
	})
	if len(explanation.Ready) != 1 || explanation.Ready[0].WorkItem.ID != blocked.ID {
		t.Fatalf("ready after blocker done = %#v", explanation.Ready)
	}
	if len(explanation.Ready[0].ResolvedBlockers) != 1 || explanation.Ready[0].ResolvedBlockers[0] != blocker.ID {
		t.Fatalf("resolved blockers = %#v", explanation.Ready[0].ResolvedBlockers)
	}
	if len(explanation.Blocked) != 0 {
		t.Fatalf("blocked after blocker done = %#v", explanation.Blocked)
	}
}

func TestBuildReadyWorkExplanationTreatsParentChildAsHierarchyOnly(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	parent := mustWorkItem(t, state, "wi_parent", project.ID)
	child := mustWorkItem(t, state, "wi_child", project.ID)
	related := mustWorkItem(t, state, "wi_related", project.ID)
	parent = mustMoveWorkItem(t, state, parent.ID, StageReady, now)
	child = mustMoveWorkItem(t, state, child.ID, StageReady, now)
	related = mustMoveWorkItem(t, state, related.ID, StageReady, now)

	for _, req := range []AddWorkItemLink{
		{ID: "link_parent", SourceWorkItemID: child.ID, TargetWorkItemID: parent.ID, Type: WorkItemLinkParentChild, Now: now},
		{ID: "link_related", SourceWorkItemID: child.ID, TargetWorkItemID: related.ID, Type: WorkItemLinkRelated, Now: now},
		{ID: "link_duplicate", SourceWorkItemID: related.ID, TargetWorkItemID: child.ID, Type: WorkItemLinkDuplicates, Now: now},
		{ID: "link_supersedes", SourceWorkItemID: parent.ID, TargetWorkItemID: related.ID, Type: WorkItemLinkSupersedes, Now: now},
	} {
		if _, err := state.AddWorkItemLink(req); err != nil {
			t.Fatalf("add %s: %v", req.ID, err)
		}
	}

	explanation := BuildReadyWorkExplanation(ReadyWorkInput{
		ProjectID: project.ID,
		WorkItems: state.ListWorkItems(project.ID),
		Links:     state.ListWorkItemLinks(""),
	})
	if len(explanation.Blocked) != 0 {
		t.Fatalf("non-blocking links blocked work = %#v", explanation.Blocked)
	}
	childReady := findReadyWorkItem(explanation.Ready, child.ID)
	if childReady == nil {
		t.Fatalf("child missing from ready = %#v", explanation.Ready)
	}
	if childReady.ParentWorkItemID == nil || *childReady.ParentWorkItemID != parent.ID {
		t.Fatalf("child parent = %#v", childReady.ParentWorkItemID)
	}
}

func mustMoveWorkItem(t *testing.T, state *State, id string, stageID string, now time.Time) WorkItem {
	t.Helper()
	item, err := state.MoveWorkItem(MoveWorkItem{
		ID:        id,
		HistoryID: "hist_move_" + id + "_" + stageID,
		StageID:   stageID,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("move %s to %s: %v", id, stageID, err)
	}
	return item
}

func findReadyWorkItem(items []ReadyWorkItem, id string) *ReadyWorkItem {
	for i := range items {
		if items[i].WorkItem.ID == id {
			return &items[i]
		}
	}
	return nil
}
