package workitem

import "testing"

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
