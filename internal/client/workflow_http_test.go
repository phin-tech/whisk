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
	execution, err := daemon.StartExecution(ctx, protocol.StartExecutionRequest{
		WorkItemID: item.ID,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
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
}
