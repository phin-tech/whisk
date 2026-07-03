package workitem

import (
	"testing"
	"time"
)

func TestValidateWorkflowTemplateRejectsInvalidInputs(t *testing.T) {
	invalid := []WorkflowTemplate{
		{},
		{ID: "wf"},
		{ID: "wf", Name: "WF"},
		{ID: "wf", Name: "WF", Stages: []WorkflowStage{{}}},
		{ID: "wf", Name: "WF", Stages: []WorkflowStage{{ID: "s1"}}},
		{ID: "wf", Name: "WF", Stages: []WorkflowStage{{ID: "s1", Name: "S1"}}},
		{ID: "wf", Name: "WF", Stages: []WorkflowStage{
			{ID: "s1", Name: "S1", Kind: "plan"},
			{ID: "s1", Name: "S2", Kind: "plan"},
		}},
	}
	for i, tpl := range invalid {
		if err := validateWorkflowTemplate(tpl); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
	if err := validateWorkflowTemplate(WorkflowTemplate{
		ID:     "wf",
		Name:   "WF",
		Stages: []WorkflowStage{{ID: "s1", Name: "S1", Kind: "plan"}},
	}); err != nil {
		t.Fatalf("valid template: %v", err)
	}
}

func TestValidatePromptTemplateRejectsInvalidInputs(t *testing.T) {
	for i, tpl := range []PromptTemplate{{}, {ID: "p"}, {ID: "p", Name: "P"}} {
		if err := validatePromptTemplate(tpl); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
	if err := validatePromptTemplate(PromptTemplate{ID: "p", Name: "P", Body: "do it"}); err != nil {
		t.Fatalf("valid prompt: %v", err)
	}
}

func TestValidateWorkflowDefinitionRecordRejectsInvalidInputs(t *testing.T) {
	valid := DefaultWorkflowDefinitionRecord(time.Now())
	if err := validateWorkflowDefinitionRecord(valid); err != nil {
		t.Fatalf("valid record: %v", err)
	}

	invalidDefinition := valid
	invalidDefinition.Definition.Stages = nil

	identityMismatch := valid
	identityMismatch.ID = "other-workflow"

	hashMismatch := valid
	hashMismatch.ContentHash = "wrong"

	invalid := []WorkflowDefinitionRecord{
		{},
		{ID: valid.ID},
		{ID: valid.ID, Version: valid.Version},
		identityMismatch,
		invalidDefinition,
		hashMismatch,
	}
	for i, record := range invalid {
		if err := validateWorkflowDefinitionRecord(record); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestWorkflowHelperFormattingAndRequirements(t *testing.T) {
	if got := stageNameFromID("qa_review-ready"); got != "Qa Review Ready" {
		t.Fatalf("stage name = %q", got)
	}
	if got := stageNameFromID(""); got != "" {
		t.Fatalf("empty stage name = %q", got)
	}

	reasons := map[WorkflowArtifactRequirement]string{
		{Kind: ArtifactKindPlan, Status: ArtifactStatusDraft}:     "plan draft required",
		{Kind: ArtifactKindPlan, Status: ArtifactStatusApproved}:  "approved plan required",
		{Kind: ArtifactKindFeedback, Status: ArtifactStatusDraft}: "draft feedback artifact required",
	}
	for requirement, want := range reasons {
		if got := workflowArtifactRequirementReason(requirement); got != want {
			t.Fatalf("reason for %#v = %q, want %q", requirement, got, want)
		}
	}

	if action := mustDefaultWorkflowAction(WorkflowActionStartPlanning); action.ID != WorkflowActionStartPlanning {
		t.Fatalf("default action = %#v", action)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic for missing default action")
			}
		}()
		_ = mustDefaultWorkflowAction("missing-action")
	}()
}

func TestStateListHelpersSortAndFilterSeededCollections(t *testing.T) {
	state := NewState()
	projectB := mustProject(t, state, "proj_b", "Beta")
	projectA := mustProject(t, state, "proj_a", "Alpha")
	itemB := mustWorkItem(t, state, "wi_b", projectB.ID)
	itemA := mustWorkItem(t, state, "wi_a", projectA.ID)
	now := time.Now()
	name, description, slug, interactive := "  Alpha Updated  ", "  Useful context  ", "Alpha Updated", true
	updatedProject, err := state.UpdateProject(UpdateProject{
		ID:                       projectA.ID,
		Name:                     &name,
		Description:              &description,
		Slug:                     &slug,
		UseInteractiveAgentShell: &interactive,
		DefaultPhaseAgents:       map[string]string{RunPresetWriter: "claude", RunPresetReader: ""},
		Now:                      now,
	})
	if err != nil || updatedProject.Name != "Alpha Updated" || updatedProject.Slug != "alpha-updated" || !updatedProject.Preferences.UseInteractiveAgentShell || updatedProject.Preferences.DefaultPhaseAgents[RunPresetWriter] != "claude" {
		t.Fatalf("updated project = %#v, err = %v", updatedProject, err)
	}

	definition := DefaultWorkflowDefinition()
	definition.Version = 2
	record, err := NewWorkflowDefinitionRecord(definition, "test", "", now)
	if err != nil {
		t.Fatalf("new workflow definition record: %v", err)
	}
	state.workflowDefinitions[workflowDefinitionKey{id: record.ID, version: record.Version}] = record
	if definitions := state.ListWorkflowDefinitions(); len(definitions) < 2 || definitions[0].Version >= definitions[1].Version {
		t.Fatalf("workflow definitions = %#v", definitions)
	}

	state.links["link_b"] = WorkItemLink{ID: "link_b", ProjectID: projectB.ID, SourceWorkItemID: itemB.ID, TargetWorkItemID: itemA.ID, Type: WorkItemLinkBlocks}
	state.links["link_a"] = WorkItemLink{ID: "link_a", ProjectID: projectA.ID, SourceWorkItemID: itemA.ID, TargetWorkItemID: itemB.ID, Type: WorkItemLinkParentChild}
	if links := state.ListWorkItemLinks(""); len(links) != 2 || links[0].ID != "link_a" {
		t.Fatalf("links = %#v", links)
	}
	if links := state.ListWorkItemLinks(itemB.ID); len(links) != 2 {
		t.Fatalf("filtered links = %#v", links)
	}

	state.runs["run_b"] = WorkItemRun{ID: "run_b", ProjectID: projectB.ID, WorkItemID: itemB.ID, CreatedAt: now.Add(time.Second)}
	state.runs["run_a"] = WorkItemRun{ID: "run_a", ProjectID: projectA.ID, WorkItemID: itemA.ID, CreatedAt: now}
	if runs := state.ListRuns(""); len(runs) != 2 || runs[0].ID != "run_a" {
		t.Fatalf("runs = %#v", runs)
	}
	if runs := state.ListRuns(itemB.ID); len(runs) != 1 || runs[0].ID != "run_b" {
		t.Fatalf("filtered runs = %#v", runs)
	}

	state.artifacts["artifact_b"] = Artifact{ID: "artifact_b", ProjectID: projectB.ID, WorkItemID: itemB.ID, CreatedAt: now.Add(time.Second)}
	state.artifacts["artifact_a"] = Artifact{ID: "artifact_a", ProjectID: projectA.ID, WorkItemID: itemA.ID, CreatedAt: now}
	if artifacts := state.ListArtifacts(""); len(artifacts) != 2 || artifacts[0].ID != "artifact_a" {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if artifacts := state.ListArtifacts(itemB.ID); len(artifacts) != 1 || artifacts[0].ID != "artifact_b" {
		t.Fatalf("filtered artifacts = %#v", artifacts)
	}

	state.questions["question_b"] = Question{ID: "question_b", ProjectID: projectB.ID, WorkItemID: itemB.ID, Prompt: "B?", CreatedAt: now.Add(time.Second)}
	state.questions["question_a"] = Question{ID: "question_a", ProjectID: projectA.ID, WorkItemID: itemA.ID, Prompt: "A?", CreatedAt: now}
	if questions := state.ListQuestions(""); len(questions) != 2 || questions[0].ID != "question_a" {
		t.Fatalf("questions = %#v", questions)
	}

	state.gateReports["gate_b"] = GateReport{ID: "gate_b", ProjectID: projectB.ID, WorkItemID: itemB.ID, Name: "B", CreatedAt: now.Add(time.Second)}
	state.gateReports["gate_a"] = GateReport{ID: "gate_a", ProjectID: projectA.ID, WorkItemID: itemA.ID, Name: "A", CreatedAt: now}
	if gates := state.ListGateReports(""); len(gates) != 2 || gates[0].ID != "gate_a" {
		t.Fatalf("gates = %#v", gates)
	}
	if !state.hasGateReport(itemA.ID, "A") || state.hasGateReport(itemA.ID, "missing") {
		t.Fatalf("gate lookup did not match seeded reports")
	}
	if action, err := state.workflowActionForItem(itemA, WorkflowActionStartPlanning); err != nil || action.ID != WorkflowActionStartPlanning {
		t.Fatalf("workflow action = %#v, err = %v", action, err)
	}
	if _, err := state.workflowActionForItem(itemA, "missing-action"); err == nil {
		t.Fatalf("expected missing workflow action error")
	}
	missingWorkflowItem := itemA
	missingWorkflowItem.WorkflowID = "missing-workflow"
	if _, err := state.workflowDefinitionForItem(missingWorkflowItem); err == nil {
		t.Fatalf("expected missing workflow definition error")
	}

	state.workflowEvents["event_b"] = WorkflowEvent{ID: "event_b", ProjectID: projectB.ID, WorkItemID: itemB.ID, At: now.Add(time.Second)}
	state.workflowEvents["event_a"] = WorkflowEvent{ID: "event_a", ProjectID: projectA.ID, WorkItemID: itemA.ID, At: now}
	if events := state.ListWorkflowEvents(""); len(events) != 2 || events[0].ID != "event_a" {
		t.Fatalf("workflow events = %#v", events)
	}

	readAt := now
	state.statusEvents["status_read"] = StatusEvent{ID: "status_read", ProjectID: projectA.ID, WorkItemID: itemA.ID, RunID: "run_a", SessionID: "sess_01", PTYID: "pty_01", CreatedAt: now, ReadAt: &readAt}
	state.statusEvents["status_unread"] = StatusEvent{ID: "status_unread", ProjectID: projectA.ID, WorkItemID: itemA.ID, RunID: "run_a", SessionID: "sess_01", PTYID: "pty_01", CreatedAt: now.Add(time.Second)}
	state.statusEvents["status_other"] = StatusEvent{ID: "status_other", ProjectID: projectB.ID, WorkItemID: itemB.ID, RunID: "run_b", SessionID: "sess_02", PTYID: "pty_02", CreatedAt: now.Add(2 * time.Second)}
	statuses := state.ListStatusEvents(ListStatusEvents{ProjectID: projectA.ID, WorkItemID: itemA.ID, RunID: "run_a", SessionID: "sess_01", PTYID: "pty_01", UnreadOnly: true})
	if len(statuses) != 1 || statuses[0].ID != "status_unread" {
		t.Fatalf("status events = %#v", statuses)
	}
}

func TestValidateProjectRejectsInvalidInputs(t *testing.T) {
	invalid := []Project{
		{},
		{ID: "x"},
		{ID: "x", Name: "N"},
		{ID: "x", Name: "N", Slug: "n"}, // missing/empty root dir
		{ID: "x", Name: "N", Slug: "n", RootDir: "relative/path"},               // not absolute
		{ID: "x", Name: "N", Slug: "n", RootDir: "/abs"},                        // next number not positive
		{ID: "x", Name: "N", Slug: "n", RootDir: "/abs", NextWorkItemNumber: 1}, // empty workflow cascade
	}
	for i, project := range invalid {
		if err := validateProject(project); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestValidateWorkItemRejectsInvalidInputs(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_01", "One")
	valid := mustWorkItem(t, state, "wi_01", "proj_01")
	if err := state.validateWorkItem(valid); err != nil {
		t.Fatalf("valid item should pass: %v", err)
	}

	mutators := []func(WorkItem) WorkItem{
		func(w WorkItem) WorkItem { w.ID = ""; return w },
		func(w WorkItem) WorkItem { w.ProjectID = "missing"; return w },
		func(w WorkItem) WorkItem { w.Number = 0; return w },
		func(w WorkItem) WorkItem { w.Title = "   "; return w },
		func(w WorkItem) WorkItem { w.WorkflowID = ""; return w },
		func(w WorkItem) WorkItem { w.WorkflowVersion = 0; return w },
		func(w WorkItem) WorkItem { w.WorkflowID = "other-workflow"; return w },
		func(w WorkItem) WorkItem { w.StageID = "missing-stage"; return w },
		func(w WorkItem) WorkItem { w.RunState = ""; return w },
	}
	for i, mutate := range mutators {
		if err := state.validateWorkItem(mutate(valid)); err == nil {
			t.Fatalf("mutator %d: expected error", i)
		}
	}
}
