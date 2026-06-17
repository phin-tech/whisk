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

	name := "Agent App Updated"
	description := "Project editor contract"
	project, err = daemon.UpdateProject(ctx, project.ID, protocol.UpdateProjectRequest{Name: &name, Description: &description})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if project.Name != name || project.Description != description {
		t.Fatalf("updated project = %#v", project)
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

func TestHTTPClientCreatesProjectSession(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	defer httpServer.Close()

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()
	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{
		Name:    "Agent App",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	created, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:      "Agent App",
		RootDir:   project.RootDir,
		ProjectID: project.ID,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.Session.ProjectID != project.ID {
		t.Fatalf("session project id = %q, want %q", created.Session.ProjectID, project.ID)
	}

	cleared, err := daemon.SetSessionProject(ctx, protocol.SetSessionProjectRequest{SessionID: created.Session.ID})
	if err != nil {
		t.Fatalf("clear session project: %v", err)
	}
	if cleared.ProjectID != "" {
		t.Fatalf("cleared session project id = %q", cleared.ProjectID)
	}
	updated, err := daemon.SetSessionProject(ctx, protocol.SetSessionProjectRequest{SessionID: created.Session.ID, ProjectID: project.ID})
	if err != nil {
		t.Fatalf("set session project: %v", err)
	}
	if updated.ProjectID != project.ID {
		t.Fatalf("updated session project id = %q", updated.ProjectID)
	}

	if _, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:      "Missing",
		RootDir:   project.RootDir,
		ProjectID: "missing",
	}); err == nil {
		t.Fatalf("expected missing project error")
	}
	if _, err := daemon.SetSessionProject(ctx, protocol.SetSessionProjectRequest{SessionID: created.Session.ID, ProjectID: "missing"}); err == nil {
		t.Fatalf("expected missing project assignment error")
	}
}

func TestHTTPClientGetsProjectDetail(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	defer httpServer.Close()

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()
	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{Name: "Agent App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	otherProject, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{Name: "Other", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}
	item, err := daemon.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{ProjectID: project.ID, Title: "Task", Actor: "agent"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	if _, err := daemon.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{ProjectID: otherProject.ID, Title: "Other", Actor: "agent"}); err != nil {
		t.Fatalf("create other work item: %v", err)
	}
	if _, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{Name: "Project session", RootDir: project.RootDir, ProjectID: project.ID}); err != nil {
		t.Fatalf("create project session: %v", err)
	}
	if _, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{Name: "Other session", RootDir: otherProject.RootDir, ProjectID: otherProject.ID}); err != nil {
		t.Fatalf("create other session: %v", err)
	}
	run, err := daemon.StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{WorkItemID: item.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	detail, err := daemon.GetProjectDetail(ctx, project.ID)
	if err != nil {
		t.Fatalf("project detail: %v", err)
	}
	if detail.Project.ID != project.ID {
		t.Fatalf("detail project = %#v", detail.Project)
	}
	if len(detail.WorkItems) != 1 || detail.WorkItems[0].ID != item.ID {
		t.Fatalf("detail items = %#v", detail.WorkItems)
	}
	if len(detail.Sessions) != 1 || detail.Sessions[0].ProjectID != project.ID {
		t.Fatalf("detail sessions = %#v", detail.Sessions)
	}
	if len(detail.Runs) != 1 || detail.Runs[0].ID != run.ID {
		t.Fatalf("detail runs = %#v", detail.Runs)
	}

	if _, err := daemon.GetProjectDetail(ctx, "missing"); err == nil {
		t.Fatalf("expected missing project error")
	}
}
