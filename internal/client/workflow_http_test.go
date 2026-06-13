package client_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesExplicitWorkflowActions(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{
		Name:    "App",
		RootDir: t.TempDir(),
		Preferences: protocol.ProjectPreferences{
			AutoWorktree: true,
			AutoRun:      workitem.AutoRunNever,
			DefaultPhaseAgents: map[string]string{
				workitem.StagePlanning:  "codex",
				workitem.StageExecution: "codex",
			},
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := daemon.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Task",
		Actor:     "human",
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	planning, err := daemon.StartPlanning(ctx, protocol.StartPlanningRequest{
		WorkItemID: item.ID,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if planning.PromptTemplateID != workitem.PromptTemplatePlan {
		t.Fatalf("planning = %#v", planning)
	}
	draft, err := daemon.SubmitDraftPlan(ctx, protocol.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      planning.ID,
		Title:      "Plan",
		Body:       "Do it.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if draft.Kind != workitem.ArtifactKindPlan || draft.Status != workitem.ArtifactStatusDraft {
		t.Fatalf("draft = %#v", draft)
	}
	ready, err := daemon.ApprovePlan(ctx, protocol.ApprovePlanRequest{
		WorkItemID: item.ID,
		ArtifactID: draft.ID,
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if ready.StageID != workitem.StageReady {
		t.Fatalf("ready = %#v", ready)
	}
	execution, err := daemon.QueueExecution(ctx, protocol.QueueExecutionRequest{
		WorkItemID: item.ID,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("queue execution: %v", err)
	}
	question, err := daemon.AskQuestion(ctx, protocol.AskQuestionRequest{
		WorkItemID: item.ID,
		RunID:      execution.ID,
		Prompt:     "Which key?",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if question.Status != workitem.QuestionStatusOpen {
		t.Fatalf("question = %#v", question)
	}
	answered, err := daemon.AnswerQuestion(ctx, protocol.AnswerQuestionRequest{
		ID:     question.ID,
		Answer: "Staging.",
		Actor:  "human",
	})
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}
	if answered.Status != workitem.QuestionStatusAnswered {
		t.Fatalf("answered = %#v", answered)
	}
	questions, err := daemon.ListQuestions(ctx, item.ID)
	if err != nil {
		t.Fatalf("list questions: %v", err)
	}
	if len(questions) != 1 || questions[0].ID != question.ID {
		t.Fatalf("questions = %#v", questions)
	}
	review, err := daemon.CompleteExecution(ctx, protocol.CompleteExecutionRequest{
		RunID:   execution.ID,
		Message: "Done.",
		Actor:   "agent",
	})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if review.StageID != workitem.StageReview {
		t.Fatalf("review = %#v", review)
	}
	gates, err := daemon.ListGateReports(ctx, item.ID)
	if err != nil {
		t.Fatalf("list gates: %v", err)
	}
	if len(gates) != 1 || !gates[0].Blocking || gates[0].Status != workitem.GateStatusPending {
		t.Fatalf("gates = %#v", gates)
	}
	if _, err := daemon.ApproveDone(ctx, protocol.ApproveDoneRequest{WorkItemID: item.ID, Actor: "human"}); err == nil {
		t.Fatalf("approve done should fail with pending gate")
	}
	feedback, err := daemon.SubmitReviewFeedback(ctx, protocol.SubmitReviewFeedbackRequest{
		WorkItemID: item.ID,
		RunID:      execution.ID,
		Body:       "Fix validation.",
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("submit feedback: %v", err)
	}
	if feedback.Kind != workitem.ArtifactKindFeedback {
		t.Fatalf("feedback = %#v", feedback)
	}
	artifacts, err := daemon.ListArtifacts(ctx, item.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	passed, err := daemon.CompleteGate(ctx, protocol.CompleteGateRequest{
		ID:     gates[0].ID,
		Status: workitem.GateStatusPassed,
		Actor:  "agent",
	})
	if err != nil {
		t.Fatalf("complete gate: %v", err)
	}
	if passed.Status != workitem.GateStatusPassed {
		t.Fatalf("passed = %#v", passed)
	}
	done, err := daemon.ApproveDone(ctx, protocol.ApproveDoneRequest{
		WorkItemID: item.ID,
		Reason:     "review passed",
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("approve done: %v", err)
	}
	if done.StageID != workitem.StageDone {
		t.Fatalf("done = %#v", done)
	}
	events, err := daemon.ListWorkflowEvents(ctx, item.ID)
	if err != nil {
		t.Fatalf("list workflow events: %v", err)
	}
	if len(events) == 0 || events[len(events)-1].Type != workitem.WorkflowEventDoneApproved {
		t.Fatalf("events = %#v", events)
	}
	if _, err := daemon.StartExecution(ctx, protocol.StartExecutionRequest{WorkItemID: "missing", Actor: "agent"}); err == nil {
		t.Fatalf("expected start execution error for missing item")
	}
	if _, err := daemon.LaunchExecution(ctx, protocol.LaunchExecutionRequest{WorkItemID: "missing", Actor: "agent"}); err == nil {
		t.Fatalf("expected launch execution error for missing item")
	}
	runToLaunch, err := daemon.StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start launch run: %v", err)
	}
	if _, err := daemon.LaunchWorkItemRun(ctx, protocol.LaunchWorkItemRunRequest{ID: runToLaunch.ID, Actor: "agent"}); err == nil {
		t.Fatalf("expected launch run error without pty backend")
	}
	runOnly, err := daemon.StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start standalone run: %v", err)
	}
	cancelled, err := daemon.CancelWorkItemRun(ctx, protocol.CancelWorkItemRunRequest{ID: runOnly.ID, Actor: "human"})
	if err != nil {
		t.Fatalf("cancel standalone run: %v", err)
	}
	if cancelled.Status != workitem.RunStateCancelled {
		t.Fatalf("cancelled = %#v", cancelled)
	}
}
