package client_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesDaemonWorkItemAPI(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	templates, err := daemon.ListWorkflowTemplates(ctx)
	if err != nil || len(templates) == 0 || templates[0].ID != "default" {
		t.Fatalf("templates = %#v, err = %v", templates, err)
	}
	prompts, err := daemon.ListPromptTemplates(ctx)
	if err != nil || len(prompts) == 0 || prompts[0].ID == "" {
		t.Fatalf("prompts = %#v, err = %v", prompts, err)
	}

	root := t.TempDir()
	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{
		Name:    "Agent App",
		RootDir: root,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID == "" || project.Slug != "agent-app" || project.Workflow.TemplateID != "default" {
		t.Fatalf("project = %#v", project)
	}

	projects, err := daemon.ListProjects(ctx)
	if err != nil || len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("projects = %#v, err = %v", projects, err)
	}

	item, err := daemon.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{
		ProjectID:    project.ID,
		Title:        "Implement CLI contract",
		BodyMarkdown: "Use daemon-owned state.",
		Actor:        "agent",
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	if item.Number != 1 || item.StageID != "backlog" || item.RunState != workitem.RunStateIdle {
		t.Fatalf("item = %#v", item)
	}

	if _, err := daemon.MoveWorkItem(ctx, protocol.MoveWorkItemRequest{ID: item.ID, StageID: "execution"}); err == nil {
		t.Fatalf("expected execution move without worktree to fail")
	}

	item, err = daemon.BindWorkItemWorktree(ctx, protocol.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/agent-app-1-cli-contract",
		Base:         "main",
		WorktreePath: root,
		Actor:        "agent",
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}
	if item.Worktree == nil || item.Worktree.Branch != "whisk/agent-app-1-cli-contract" {
		t.Fatalf("bound item = %#v", item)
	}

	item, err = daemon.MoveWorkItem(ctx, protocol.MoveWorkItemRequest{ID: item.ID, StageID: "execution", Actor: "agent"})
	if err != nil {
		t.Fatalf("move ready: %v", err)
	}
	if item.StageID != "execution" {
		t.Fatalf("moved item = %#v", item)
	}

	item, err = daemon.AddWorkItemAttachment(ctx, protocol.AddWorkItemAttachmentRequest{
		WorkItemID: item.ID,
		Kind:       workitem.AttachmentKindFile,
		Path:       "docs/spec.md",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}
	if len(item.Attachments) != 1 || item.Attachments[0].Path != "docs/spec.md" {
		t.Fatalf("attachments = %#v", item.Attachments)
	}

	items, err := daemon.ListWorkItems(ctx, project.ID)
	if err != nil || len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("items = %#v, err = %v", items, err)
	}

	run, err := daemon.StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		SessionID:        "sess_01",
		PTYID:            "pty_01",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.ID == "" || run.Status != workitem.RunStateQueued || run.PromptSnapshot == "" {
		t.Fatalf("run = %#v", run)
	}
	report, err := daemon.ReportStatus(ctx, protocol.ReportStatusRequest{
		RunID:   run.ID,
		Kind:    workitem.StatusKindQuestion,
		Message: "Need the staging API key.",
		Actor:   "agent",
	})
	if err != nil {
		t.Fatalf("report question: %v", err)
	}
	if report.Event.Kind != workitem.StatusKindQuestion || !report.Event.RequiresAttention {
		t.Fatalf("report = %#v", report)
	}
	if report.Run == nil || report.Run.Status != workitem.RunStateAwaitingInput || report.Run.History[len(report.Run.History)-1].Message != "Need the staging API key." {
		t.Fatalf("report run = %#v", report.Run)
	}
	items, err = daemon.ListWorkItems(ctx, project.ID)
	if err != nil || len(items) != 1 || items[0].RunState != workitem.RunStateAwaitingInput {
		t.Fatalf("items after question = %#v, err = %v", items, err)
	}
	events, err := daemon.ListStatusEvents(ctx, protocol.ListStatusEventsRequest{SessionID: report.Event.SessionID, UnreadOnly: true})
	if err != nil || len(events) != 1 || events[0].ID != report.Event.ID {
		t.Fatalf("status events = %#v, err = %v", events, err)
	}
	read, err := daemon.MarkStatusEventRead(ctx, protocol.MarkStatusEventReadRequest{ID: report.Event.ID})
	if err != nil || read.ReadAt == nil {
		t.Fatalf("mark read = %#v, err = %v", read, err)
	}
	runs, err := daemon.ListWorkItemRuns(ctx, item.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("runs = %#v, err = %v", runs, err)
	}
	completed, err := daemon.ReportStatus(ctx, protocol.ReportStatusRequest{
		RunID:   run.ID,
		Kind:    workitem.StatusKindDone,
		Message: "Implementation complete and tests pass.",
		Actor:   "agent",
	})
	if err != nil || completed.Run == nil || completed.Run.Status != workitem.RunStateCompleted || completed.Run.CompletedAt == nil || completed.WorkItem == nil || completed.WorkItem.StageID != "review" {
		t.Fatalf("done = %#v, err = %v", completed, err)
	}

	deleted, err := daemon.DeleteWorkItem(ctx, protocol.DeleteWorkItemRequest{ID: item.ID, Actor: "agent"})
	if err != nil || deleted.ID != item.ID {
		t.Fatalf("delete = %#v, err = %v", deleted, err)
	}
	if got := deleted.History[len(deleted.History)-1].Actor; got != "agent" {
		t.Fatalf("deleted actor = %q", got)
	}
	items, err = daemon.ListWorkItems(ctx, project.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("items after delete = %#v, err = %v", items, err)
	}
}
