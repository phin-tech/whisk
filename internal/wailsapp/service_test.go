package wailsapp_test

import (
	"context"
	"testing"

	"github.com/phin-tech/whisk/internal/appmenu"
	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/wailsapp"
)

func TestServiceDelegatesToRuntimeClient(t *testing.T) {
	fake := &runtimeClientFake{
		sessions: []session.Session{{ID: "sess_01"}},
		created: protocol.CreatedSession{
			Session:   session.Session{ID: "sess_02"},
			MainPtyID: "pty_01",
		},
		split:  protocol.SplitPaneResult{Session: session.Session{ID: "sess_02"}, PaneID: "pane_02", PtyID: "pty_02"},
		output: protocol.OutputSnapshot{PtyID: "pty_01", Offset: 12, Output: "hello"},
		ptys:   []protocol.PTYInfo{{ID: "pty_01", SessionID: "sess_01", PaneID: "pane_01"}},
		ptyHistory: []protocol.PTYHistorySummary{{
			PTYID:      "pty_01",
			SessionID:  "sess_01",
			WorkingDir: "/repo",
		}},
		selectedPTYHistory: protocol.PTYHistory{PTYID: "pty_01", Output: "hello history"},
		event:              protocol.NextEventResponse{Event: protocol.RuntimeEvent{Seq: 3, Type: "pty.changed", PtyID: "pty_01"}},
		worktrunk: protocol.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      protocol.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []protocol.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature"}},
		createdWorktree: protocol.CreatedWorktree{
			Path: "/repo/.worktrees/created",
		},
		projects:          []protocol.Project{{ID: "proj_01", Name: "App", RootDir: "/repo"}},
		projectDetail:     protocol.ProjectDetail{Project: protocol.Project{ID: "proj_01", Name: "App"}},
		workflowTemplates: []protocol.WorkflowTemplate{{ID: "default", Name: "Default"}},
		workflowDefinitions: []protocol.WorkflowDefinitionRecord{{
			ID:      workitem.WorkflowPlanExecuteReview,
			Version: 1,
		}},
		promptTemplates: []protocol.PromptTemplate{{ID: "implement", Name: "Implement"}},
		agentProfiles:   []protocol.AgentProfile{{ID: "claude", Label: "Claude"}},
		workItems:       []protocol.WorkItem{{ID: "wi_01", ProjectID: "proj_01", Number: 1, Title: "Task"}},
		workItemLinks: []protocol.WorkItemLink{{
			ID:               "link_01",
			ProjectID:        "proj_01",
			SourceWorkItemID: "wi_02",
			TargetWorkItemID: "wi_01",
			Type:             workitem.WorkItemLinkBlocks,
		}},
		readyWork: protocol.ReadyWorkExplanation{
			Summary: protocol.ReadyWorkSummary{TotalReady: 1, TotalBlocked: 1},
		},
		runs:              []protocol.WorkItemRun{{ID: "run_01", WorkItemID: "wi_01", Status: "queued", Preset: "writer"}},
		httpForwards:      []protocol.HTTPForward{{ID: "fwd_01", TargetURL: "http://127.0.0.1:4966"}},
		agentPrompts:      []protocol.AgentPrompt{{ID: "prompt_01", Status: "pending", Message: "Continue?"}},
		agentIntegrations: []protocol.AgentHookIntegration{{Provider: "claude", Status: "current"}},
		clearResponse:     protocol.ClearDaemonResponse{SessionsCleared: 1},
	}
	service := wailsapp.NewService(fake)
	ctx := context.Background()

	cleared, err := service.ClearDaemon(ctx, protocol.ClearDaemonRequest{})
	if err != nil || cleared.SessionsCleared != 1 || !fake.clearCalled {
		t.Fatalf("clear daemon = %#v, called = %v, err = %v", cleared, fake.clearCalled, err)
	}
	fake.onboardingStatus = protocol.OnboardingStatus{ShouldShow: true}
	onboardingStatus, err := service.OnboardingStatus(ctx)
	if err != nil || !onboardingStatus.ShouldShow {
		t.Fatalf("onboarding status = %#v, err = %v", onboardingStatus, err)
	}
	if _, err := service.ApplyOnboarding(ctx, protocol.OnboardingApplyRequest{ItemIDs: []string{"skill:codex"}}); err != nil || fake.onboardingApplyReq.ItemIDs[0] != "skill:codex" {
		t.Fatalf("apply onboarding req = %#v, err = %v", fake.onboardingApplyReq, err)
	}
	sessions, err := service.ListSessions(ctx)
	if err != nil || sessions[0].ID != "sess_01" {
		t.Fatalf("list sessions = %#v, %v", sessions, err)
	}
	created, err := service.CreateSession(ctx, protocol.CreateSessionRequest{Name: "created"})
	if err != nil || created.MainPtyID != "pty_01" || fake.createReq.Name != "created" {
		t.Fatalf("create = %#v, req = %#v, err = %v", created, fake.createReq, err)
	}
	split, err := service.SplitPane(ctx, protocol.SplitPaneRequest{SessionID: "sess_02"})
	if err != nil || split.PaneID != "pane_02" || fake.splitReq.SessionID != "sess_02" {
		t.Fatalf("split = %#v, req = %#v, err = %v", split, fake.splitReq, err)
	}
	if _, err := service.SetSessionRootDir(ctx, protocol.SetSessionRootDirRequest{SessionID: "sess_02", RootDir: "/repo"}); err != nil || fake.setRootReq.RootDir != "/repo" {
		t.Fatalf("set root req = %#v, err = %v", fake.setRootReq, err)
	}
	if _, err := service.SetSessionProject(ctx, protocol.SetSessionProjectRequest{SessionID: "sess_02", ProjectID: "proj_01"}); err != nil || fake.setProjectReq.ProjectID != "proj_01" {
		t.Fatalf("set project req = %#v, err = %v", fake.setProjectReq, err)
	}
	if _, err := service.SetPaneWorkingDir(ctx, protocol.SetPaneWorkingDirRequest{SessionID: "sess_02", PaneID: "pane_02", WorkingDir: "/repo/frontend"}); err != nil || fake.setPaneDirReq.WorkingDir != "/repo/frontend" {
		t.Fatalf("set pane working dir req = %#v, err = %v", fake.setPaneDirReq, err)
	}
	started, err := service.StartPanePTY(ctx, protocol.StartPanePTYRequest{SessionID: "sess_02", PaneID: "pane_02", Options: protocol.StartPTYOptions{Cols: 80, Rows: 24}})
	if err != nil || started.PTYID != "pty_03" || fake.startPanePTYReq.PaneID != "pane_02" {
		t.Fatalf("start pane pty = %#v, req = %#v, err = %v", started, fake.startPanePTYReq, err)
	}
	restarted, err := service.RestartPanePTY(ctx, protocol.RestartPanePTYRequest{SessionID: "sess_02", PaneID: "pane_02", Options: protocol.StartPTYOptions{Cols: 80, Rows: 24}})
	if err != nil || restarted.PTYID != "pty_04" || restarted.OldPTYID != "pty_03" || fake.restartPanePTYReq.PaneID != "pane_02" {
		t.Fatalf("restart pane pty = %#v, req = %#v, err = %v", restarted, fake.restartPanePTYReq, err)
	}
	detached, err := service.DetachPanePTY(ctx, protocol.DetachPanePTYRequest{SessionID: "sess_02", PaneID: "pane_02"})
	if err != nil || detached.PTYID != "pty_03" || fake.detachPanePTYReq.PaneID != "pane_02" {
		t.Fatalf("detach pane pty = %#v, req = %#v, err = %v", detached, fake.detachPanePTYReq, err)
	}
	remaining, err := service.CloseSession(ctx, protocol.CloseSessionRequest{SessionID: "sess_02"})
	if err != nil || len(remaining) != 1 || fake.closeSessionReq.SessionID != "sess_02" {
		t.Fatalf("close session = %#v, req = %#v, err = %v", remaining, fake.closeSessionReq, err)
	}
	if _, err := service.ClosePane(ctx, protocol.ClosePaneRequest{SessionID: "sess_02", WindowID: "win_01", PaneID: "pane_02"}); err != nil || fake.closePaneReq.PaneID != "pane_02" {
		t.Fatalf("close pane req = %#v, err = %v", fake.closePaneReq, err)
	}
	if err := service.WritePTY(ctx, protocol.WritePTYRequest{PtyID: "pty_01", Data: "x"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if fake.writeReq.Data != "x" {
		t.Fatalf("write req = %#v", fake.writeReq)
	}
	if err := service.ResizePTY(ctx, protocol.ResizePTYRequest{PtyID: "pty_01", Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if fake.resizeReq.Cols != 80 || fake.resizeReq.Rows != 24 {
		t.Fatalf("resize req = %#v", fake.resizeReq)
	}
	killed, err := service.KillPTY(ctx, protocol.KillPTYRequest{PTYID: "pty_01"})
	if err != nil || killed.ID != "pty_01" || fake.killReq.PTYID != "pty_01" {
		t.Fatalf("kill = %#v, req = %#v, err = %v", killed, fake.killReq, err)
	}
	if err := service.DeletePTY(ctx, protocol.DeletePTYRequest{PTYID: "pty_01"}); err != nil || fake.deletePTYReq.PTYID != "pty_01" {
		t.Fatalf("delete pty req = %#v, err = %v", fake.deletePTYReq, err)
	}
	output, err := service.Output(ctx, protocol.OutputRequest{PtyID: "pty_01", FromOffset: 7})
	if err != nil || output.Offset != 12 || fake.outputReq.FromOffset != 7 {
		t.Fatalf("output = %#v, req = %#v, err = %v", output, fake.outputReq, err)
	}
	ptys, err := service.ListPTYs(ctx)
	if err != nil || ptys[0].ID != "pty_01" {
		t.Fatalf("ptys = %#v, err = %v", ptys, err)
	}
	history, err := service.ListPTYHistory(ctx)
	if err != nil || history[0].PTYID != "pty_01" {
		t.Fatalf("pty history = %#v, err = %v", history, err)
	}
	selectedHistory, err := service.ReadPTYHistory(ctx, "pty_01")
	if err != nil || selectedHistory.Output != "hello history" || fake.readPTYHistoryID != "pty_01" {
		t.Fatalf("selected pty history = %#v, id = %q, err = %v", selectedHistory, fake.readPTYHistoryID, err)
	}
	event, err := service.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 25, AfterSeq: 2})
	if err != nil || event.Event.Type != "pty.changed" || event.Event.Seq != 3 || fake.nextEventReq.TimeoutMs != 25 || fake.nextEventReq.AfterSeq != 2 {
		t.Fatalf("event = %#v, req = %#v, err = %v", event, fake.nextEventReq, err)
	}

	worktrunk, err := service.DetectWorktrunk(ctx, protocol.DetectWorktrunkRequest{RepoPath: "/repo"})
	if err != nil || !worktrunk.Available || fake.detectWorktrunkReq.RepoPath != "/repo" {
		t.Fatalf("detect worktrunk = %#v, req = %#v, err = %v", worktrunk, fake.detectWorktrunkReq, err)
	}
	worktrees, err := service.ListWorktrees(ctx, protocol.ListWorktreesRequest{RepoPath: "/repo"})
	if err != nil || len(worktrees) != 1 || worktrees[0].Branch != "feature" || fake.listWorktreesReq.RepoPath != "/repo" {
		t.Fatalf("list worktrees = %#v, req = %#v, err = %v", worktrees, fake.listWorktreesReq, err)
	}
	createdWorktree, err := service.CreateWorktree(ctx, protocol.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	})
	if err != nil || createdWorktree.Path != "/repo/.worktrees/created" || fake.createWorktreeReq.Base != "main" {
		t.Fatalf("create worktree = %#v, req = %#v, err = %v", createdWorktree, fake.createWorktreeReq, err)
	}
	if err := service.RemoveWorktree(ctx, protocol.RemoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/created"}); err != nil {
		t.Fatalf("remove worktree: %v", err)
	}
	if fake.removeWorktreeReq.WorktreePath != "/repo/.worktrees/created" || fake.removeWorktreeReq.AlsoBranch {
		t.Fatalf("remove worktree req = %#v", fake.removeWorktreeReq)
	}
	projects, err := service.ListProjects(ctx)
	if err != nil || len(projects) != 1 || projects[0].ID != "proj_01" {
		t.Fatalf("list projects = %#v, err = %v", projects, err)
	}
	project, err := service.CreateProject(ctx, protocol.CreateProjectRequest{Name: "App", RootDir: "/repo"})
	if err != nil || project.ID != "proj_02" || fake.createProjectReq.Name != "App" {
		t.Fatalf("create project = %#v, req = %#v, err = %v", project, fake.createProjectReq, err)
	}
	description := "Daemon owned"
	project, err = service.UpdateProject(ctx, "proj_01", protocol.UpdateProjectRequest{Description: &description})
	if err != nil || project.Description != "Daemon owned" || fake.updateProjectID != "proj_01" {
		t.Fatalf("update project = %#v, id = %q, err = %v", project, fake.updateProjectID, err)
	}
	deletedProject, err := service.DeleteProject(ctx, "proj_delete", protocol.DeleteProjectRequest{Actor: "human"})
	if err != nil || deletedProject.ID != "proj_delete" || fake.deleteProjectID != "proj_delete" || fake.deleteProjectReq.Actor != "human" {
		t.Fatalf("delete project = %#v, id = %q, req = %#v, err = %v", deletedProject, fake.deleteProjectID, fake.deleteProjectReq, err)
	}
	detail, err := service.ProjectDetail(ctx, "proj_01")
	if err != nil || detail.Project.ID != "proj_01" || fake.projectDetailID != "proj_01" {
		t.Fatalf("project detail = %#v, id = %q, err = %v", detail, fake.projectDetailID, err)
	}
	project, err = service.AddProjectAttachment(ctx, protocol.AddProjectAttachmentRequest{ProjectID: "proj_01", Kind: "note", Title: "Context", Note: "Read this", IncludeInContext: true})
	if err != nil || len(project.Attachments) != 1 || fake.addProjectAttachmentReq.Note != "Read this" {
		t.Fatalf("add project attachment = %#v, req = %#v, err = %v", project, fake.addProjectAttachmentReq, err)
	}
	attachmentTitle := "Updated context"
	project, err = service.UpdateProjectAttachment(ctx, "att_01", protocol.UpdateProjectAttachmentRequest{Title: &attachmentTitle})
	if err != nil || fake.updateProjectAttachmentID != "att_01" || fake.updateProjectAttachmentReq.Title == nil || *fake.updateProjectAttachmentReq.Title != attachmentTitle {
		t.Fatalf("update project attachment = %#v, id = %q, req = %#v, err = %v", project, fake.updateProjectAttachmentID, fake.updateProjectAttachmentReq, err)
	}
	contextBundle, err := service.ProjectContext(ctx, "proj_01")
	if err != nil || contextBundle.ProjectID != "proj_01" || fake.projectContextID != "proj_01" {
		t.Fatalf("project context = %#v, id = %q, err = %v", contextBundle, fake.projectContextID, err)
	}
	project, err = service.DeleteProjectAttachment(ctx, "att_01", protocol.DeleteProjectAttachmentRequest{ProjectID: "proj_01"})
	if err != nil || len(project.Attachments) != 0 || fake.deleteProjectAttachmentID != "att_01" {
		t.Fatalf("delete project attachment = %#v, id = %q, err = %v", project, fake.deleteProjectAttachmentID, err)
	}
	templates, err := service.ListWorkflowTemplates(ctx)
	if err != nil || len(templates) != 1 || templates[0].ID != "default" {
		t.Fatalf("list templates = %#v, err = %v", templates, err)
	}
	definitions, err := service.ListWorkflowDefinitions(ctx)
	if err != nil || len(definitions) != 1 || definitions[0].ID != workitem.WorkflowPlanExecuteReview {
		t.Fatalf("list workflow definitions = %#v, err = %v", definitions, err)
	}
	validation, err := service.ValidateWorkflowDefinition(ctx, protocol.ValidateWorkflowDefinitionRequest{Definition: workitem.DefaultWorkflowDefinition()})
	if err != nil || !validation.Valid || fake.validateWorkflowDefinitionReq.Definition.ID != workitem.WorkflowPlanExecuteReview {
		t.Fatalf("validate workflow definition = %#v, req = %#v, err = %v", validation, fake.validateWorkflowDefinitionReq, err)
	}
	imported, err := service.ImportWorkflowDefinition(ctx, protocol.ImportWorkflowDefinitionRequest{Definition: workitem.DefaultWorkflowDefinition(), Source: "test"})
	if err != nil || imported.ID != workitem.WorkflowPlanExecuteReview || fake.importWorkflowDefinitionReq.Source != "test" {
		t.Fatalf("import workflow definition = %#v, req = %#v, err = %v", imported, fake.importWorkflowDefinitionReq, err)
	}
	fileValidation, err := service.ValidateWorkflowDefinitionFile(ctx, protocol.ValidateWorkflowDefinitionFileRequest{Path: "/tmp/workflow.json"})
	if err != nil || !fileValidation.Valid || fake.validateWorkflowFileReq.Path != "/tmp/workflow.json" {
		t.Fatalf("validate workflow file = %#v, req = %#v, err = %v", fileValidation, fake.validateWorkflowFileReq, err)
	}
	importedFile, err := service.ImportWorkflowDefinitionFile(ctx, protocol.ImportWorkflowDefinitionFileRequest{Path: "/tmp/workflow.json"})
	if err != nil || importedFile.SourcePath != "/tmp/workflow.json" || fake.importWorkflowFileReq.Path != "/tmp/workflow.json" {
		t.Fatalf("import workflow file = %#v, req = %#v, err = %v", importedFile, fake.importWorkflowFileReq, err)
	}
	if err := service.ExportWorkflowDefinitionFile(ctx, protocol.ExportWorkflowDefinitionFileRequest{ID: "file", Version: 1, Path: "/tmp/out.json"}); err != nil || fake.exportWorkflowFileReq.Path != "/tmp/out.json" {
		t.Fatalf("export workflow file req = %#v, err = %v", fake.exportWorkflowFileReq, err)
	}
	deletedWorkflow, err := service.DeleteWorkflowDefinition(ctx, "file", 1)
	if err != nil || deletedWorkflow.ID != "file" || fake.deleteWorkflowDefinitionID != "file" || fake.deleteWorkflowDefinitionVersion != 1 {
		t.Fatalf("delete workflow definition = %#v, id = %q, version = %d, err = %v", deletedWorkflow, fake.deleteWorkflowDefinitionID, fake.deleteWorkflowDefinitionVersion, err)
	}
	migrationPlan, err := service.PlanProjectWorkflowMigration(ctx, "proj_01", protocol.PlanProjectWorkflowMigrationRequest{ID: workitem.WorkflowPlanExecuteReview, Version: 1})
	if err != nil || migrationPlan.ProjectID != "proj_01" || fake.workflowMigrationProjectID != "proj_01" || fake.workflowMigrationReq.ID != workitem.WorkflowPlanExecuteReview {
		t.Fatalf("workflow migration = %#v, id = %q, req = %#v, err = %v", migrationPlan, fake.workflowMigrationProjectID, fake.workflowMigrationReq, err)
	}
	project, err = service.SetProjectWorkflowDefinition(ctx, "proj_01", protocol.SetProjectWorkflowDefinitionRequest{
		ID:      workitem.WorkflowPlanExecuteReview,
		Version: 1,
	})
	if err != nil || project.Workflow.DefinitionID != workitem.WorkflowPlanExecuteReview || fake.setProjectWorkflowDefinitionID != "proj_01" {
		t.Fatalf("set workflow definition = %#v, id = %q, req = %#v, err = %v", project, fake.setProjectWorkflowDefinitionID, fake.setProjectWorkflowDefinitionReq, err)
	}
	promptTemplates, err := service.ListPromptTemplates(ctx)
	if err != nil || len(promptTemplates) != 1 || promptTemplates[0].ID != "implement" {
		t.Fatalf("list prompt templates = %#v, err = %v", promptTemplates, err)
	}
	agentProfiles, err := service.ListAgentProfiles(ctx)
	if err != nil || len(agentProfiles) != 1 || agentProfiles[0].ID != "claude" {
		t.Fatalf("list agent profiles = %#v, err = %v", agentProfiles, err)
	}
	items, err := service.ListWorkItems(ctx, "proj_01")
	if err != nil || len(items) != 1 || items[0].ID != "wi_01" || fake.listWorkItemsProjectID != "proj_01" {
		t.Fatalf("list work items = %#v, project = %q, err = %v", items, fake.listWorkItemsProjectID, err)
	}
	item, err := service.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{ProjectID: "proj_01", Title: "Task"})
	if err != nil || item.ID != "wi_02" || fake.createWorkItemReq.Title != "Task" {
		t.Fatalf("create work item = %#v, req = %#v, err = %v", item, fake.createWorkItemReq, err)
	}
	title := "Updated task"
	body := ""
	item, err = service.UpdateWorkItem(ctx, protocol.UpdateWorkItemRequest{ID: "wi_02", Title: &title, BodyMarkdown: &body})
	if err != nil || item.Title != title || fake.updateWorkItemReq.BodyMarkdown == nil {
		t.Fatalf("update work item = %#v, req = %#v, err = %v", item, fake.updateWorkItemReq, err)
	}
	item, err = service.MoveWorkItem(ctx, protocol.MoveWorkItemRequest{ID: "wi_02", StageID: "ready"})
	if err != nil || item.StageID != "ready" || fake.moveWorkItemReq.StageID != "ready" {
		t.Fatalf("move work item = %#v, req = %#v, err = %v", item, fake.moveWorkItemReq, err)
	}
	workflowActions, err := service.ListWorkItemWorkflowActions(ctx, "wi_02")
	if err != nil || len(workflowActions) != 1 || fake.listWorkflowActionsWorkItemID != "wi_02" {
		t.Fatalf("workflow actions = %#v, id = %q, err = %v", workflowActions, fake.listWorkflowActionsWorkItemID, err)
	}
	link, err := service.AddWorkItemLink(ctx, protocol.AddWorkItemLinkRequest{
		SourceWorkItemID: "wi_02",
		TargetWorkItemID: "wi_01",
		Type:             workitem.WorkItemLinkBlocks,
		Actor:            "agent",
	})
	if err != nil || link.ID != "link_01" || fake.addWorkItemLinkReq.SourceWorkItemID != "wi_02" {
		t.Fatalf("add work item link = %#v, req = %#v, err = %v", link, fake.addWorkItemLinkReq, err)
	}
	links, err := service.ListWorkItemLinks(ctx, "wi_02")
	if err != nil || len(links) != 1 || links[0].ID != "link_01" || fake.listWorkItemLinksID != "wi_02" {
		t.Fatalf("list work item links = %#v, id = %q, err = %v", links, fake.listWorkItemLinksID, err)
	}
	readyWork, err := service.ReadyWork(ctx, protocol.ReadyWorkRequest{ProjectID: "proj_01"})
	if err != nil || readyWork.Summary.TotalReady != 1 || fake.readyWorkReq.ProjectID != "proj_01" {
		t.Fatalf("ready work = %#v, req = %#v, err = %v", readyWork, fake.readyWorkReq, err)
	}
	item, err = service.BindWorkItemWorktree(ctx, protocol.BindWorkItemWorktreeRequest{ID: "wi_02", Branch: "whisk/app-2-task", WorktreePath: "/repo/.worktrees/task"})
	if err != nil || item.Worktree == nil || fake.bindWorkItemReq.Branch != "whisk/app-2-task" {
		t.Fatalf("bind work item = %#v, req = %#v, err = %v", item, fake.bindWorkItemReq, err)
	}
	item, err = service.AddWorkItemAttachment(ctx, protocol.AddWorkItemAttachmentRequest{WorkItemID: "wi_02", Kind: "file", Path: "docs/spec.md"})
	if err != nil || len(item.Attachments) != 1 || fake.addWorkItemAttachmentReq.Path != "docs/spec.md" {
		t.Fatalf("add attachment = %#v, req = %#v, err = %v", item, fake.addWorkItemAttachmentReq, err)
	}
	deleted, err := service.DeleteWorkItem(ctx, protocol.DeleteWorkItemRequest{ID: "wi_02"})
	if err != nil || deleted.ID != "wi_02" || fake.deleteWorkItemReq.ID != "wi_02" {
		t.Fatalf("delete work item = %#v, req = %#v, err = %v", deleted, fake.deleteWorkItemReq, err)
	}
	runs, err := service.ListWorkItemRuns(ctx, "wi_01")
	if err != nil || len(runs) != 1 || runs[0].ID != "run_01" || fake.listRunsWorkItemID != "wi_01" {
		t.Fatalf("list runs = %#v, work item = %q, err = %v", runs, fake.listRunsWorkItemID, err)
	}
	run, err := service.StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{WorkItemID: "wi_01", Preset: "writer", PromptTemplateID: "implement"})
	if err != nil || run.WorkItemID != "wi_01" || fake.startRunReq.Preset != "writer" {
		t.Fatalf("start run = %#v, req = %#v, err = %v", run, fake.startRunReq, err)
	}
	run, err = service.CancelWorkItemRun(ctx, protocol.CancelWorkItemRunRequest{ID: "run_01", Actor: "agent"})
	if err != nil || run.Status != "cancelled" || fake.cancelRunReq.Actor != "agent" {
		t.Fatalf("cancel run = %#v, req = %#v, err = %v", run, fake.cancelRunReq, err)
	}
	planning, err := service.StartPlanning(ctx, protocol.StartPlanningRequest{WorkItemID: "wi_01", Actor: "agent"})
	if err != nil || planning.PromptTemplateID != "plan" {
		t.Fatalf("start planning = %#v, err = %v", planning, err)
	}
	draft, err := service.SubmitDraftPlan(ctx, protocol.SubmitDraftPlanRequest{WorkItemID: "wi_01", RunID: planning.ID, Body: "Do it.", Actor: "agent"})
	if err != nil || draft.Status != "draft" {
		t.Fatalf("submit draft = %#v, err = %v", draft, err)
	}
	item, err = service.ApprovePlan(ctx, protocol.ApprovePlanRequest{WorkItemID: "wi_01", ArtifactID: draft.ID, Actor: "human"})
	if err != nil || item.StageID != "ready" {
		t.Fatalf("approve plan = %#v, err = %v", item, err)
	}
	run, err = service.StartExecution(ctx, protocol.StartExecutionRequest{WorkItemID: "wi_01", Actor: "agent"})
	if err != nil || run.PromptTemplateID != "implement" {
		t.Fatalf("start execution = %#v, err = %v", run, err)
	}
	run, err = service.QueueExecution(ctx, protocol.QueueExecutionRequest{WorkItemID: "wi_01", Actor: "human"})
	if err != nil || run.Status != "queued" {
		t.Fatalf("queue execution = %#v, err = %v", run, err)
	}
	run, err = service.LaunchExecution(ctx, protocol.LaunchExecutionRequest{WorkItemID: "wi_01", Actor: "agent"})
	if err != nil || run.Status != "running" {
		t.Fatalf("launch execution = %#v, err = %v", run, err)
	}
	run, err = service.LaunchWorkItemRun(ctx, protocol.LaunchWorkItemRunRequest{ID: "run_01", Actor: "agent"})
	if err != nil || run.Status != "running" {
		t.Fatalf("launch run = %#v, err = %v", run, err)
	}
	question, err := service.AskQuestion(ctx, protocol.AskQuestionRequest{WorkItemID: "wi_01", RunID: "run_01", Prompt: "Which key?", Actor: "agent"})
	if err != nil || question.Status != "open" {
		t.Fatalf("ask question = %#v, err = %v", question, err)
	}
	question, err = service.AnswerQuestion(ctx, protocol.AnswerQuestionRequest{ID: question.ID, Answer: "Staging.", Actor: "human"})
	if err != nil || question.Status != "answered" {
		t.Fatalf("answer question = %#v, err = %v", question, err)
	}
	item, err = service.CompleteExecution(ctx, protocol.CompleteExecutionRequest{WorkItemID: "wi_01", RunID: "run_01", Actor: "agent"})
	if err != nil || item.StageID != "review" {
		t.Fatalf("complete execution = %#v, err = %v", item, err)
	}
	feedback, err := service.SubmitReviewFeedback(ctx, protocol.SubmitReviewFeedbackRequest{WorkItemID: "wi_01", RunID: "run_01", Body: "Fix validation.", Actor: "human"})
	if err != nil || feedback.Kind != "feedback" {
		t.Fatalf("submit feedback = %#v, err = %v", feedback, err)
	}
	item, err = service.ApproveDone(ctx, protocol.ApproveDoneRequest{WorkItemID: "wi_01", Actor: "human"})
	if err != nil || item.StageID != "done" {
		t.Fatalf("approve done = %#v, err = %v", item, err)
	}
	artifacts, err := service.ListArtifacts(ctx, "wi_01")
	if err != nil || len(artifacts) != 1 || artifacts[0].Kind != "plan" {
		t.Fatalf("artifacts = %#v, err = %v", artifacts, err)
	}
	questions, err := service.ListQuestions(ctx, "wi_01")
	if err != nil || len(questions) != 1 || questions[0].Status != "open" {
		t.Fatalf("questions = %#v, err = %v", questions, err)
	}
	gates, err := service.ListGateReports(ctx, "wi_01")
	if err != nil || len(gates) != 1 || gates[0].Status != "pending" {
		t.Fatalf("gates = %#v, err = %v", gates, err)
	}
	gate, err := service.CompleteGate(ctx, protocol.CompleteGateRequest{ID: "gate_01", Status: workitem.GateStatusPassed, Actor: "agent"})
	if err != nil || gate.Status != workitem.GateStatusPassed {
		t.Fatalf("complete gate = %#v, err = %v", gate, err)
	}
	workflowEvents, err := service.ListWorkflowEvents(ctx, "wi_01")
	if err != nil || len(workflowEvents) != 1 || workflowEvents[0].Type != "planning_started" {
		t.Fatalf("workflow events = %#v, err = %v", workflowEvents, err)
	}
	status, err := service.ReportStatus(ctx, protocol.ReportStatusRequest{Kind: workitem.StatusKindQuestion, Message: "Need input.", WorkItemID: "wi_01", RunID: "run_01"})
	if err != nil || status.Event.Message != "Need input." || fake.reportStatusReq.Kind != workitem.StatusKindQuestion {
		t.Fatalf("report status = %#v, req = %#v, err = %v", status, fake.reportStatusReq, err)
	}
	statusEvents, err := service.ListStatusEvents(ctx, protocol.ListStatusEventsRequest{SessionID: "sess_01", UnreadOnly: true})
	if err != nil || len(statusEvents) != 1 || fake.listStatusEventsReq.SessionID != "sess_01" {
		t.Fatalf("status events = %#v, req = %#v, err = %v", statusEvents, fake.listStatusEventsReq, err)
	}
	statusEvent, err := service.MarkStatusEventRead(ctx, protocol.MarkStatusEventReadRequest{ID: "status_01"})
	if err != nil || statusEvent.ID != "status_01" || fake.markStatusReadReq.ID != "status_01" {
		t.Fatalf("mark status read = %#v, req = %#v, err = %v", statusEvent, fake.markStatusReadReq, err)
	}
	approvals, err := service.ListAgentBridgeApprovals(ctx, protocol.ListAgentBridgeApprovalsRequest{Status: "pending"})
	if err != nil || len(approvals) != 0 || fake.listAgentApprovalsReq.Status != "pending" {
		t.Fatalf("agent approvals = %#v, req = %#v, err = %v", approvals, fake.listAgentApprovalsReq, err)
	}
	approval, err := service.ResolveAgentBridgeApproval(ctx, "approval_01", protocol.ResolveAgentBridgeApprovalRequest{Action: "allow"})
	if err != nil || approval.ID != "approval_01" || fake.resolveAgentApprovalID != "approval_01" || fake.resolveAgentApprovalReq.Action != "allow" {
		t.Fatalf("resolve agent approval = %#v, id = %q, req = %#v, err = %v", approval, fake.resolveAgentApprovalID, fake.resolveAgentApprovalReq, err)
	}
	agentPrompts, err := service.ListAgentPrompts(ctx, protocol.ListAgentPromptsRequest{Status: "pending"})
	if err != nil || len(agentPrompts) != 1 || agentPrompts[0].ID != "prompt_01" || fake.listAgentPromptsReq.Status != "pending" {
		t.Fatalf("agent prompts = %#v, req = %#v, err = %v", agentPrompts, fake.listAgentPromptsReq, err)
	}
	agentPrompt, err := service.ResolveAgentPrompt(ctx, "prompt_01", protocol.ResolveAgentPromptRequest{Answer: "yes"})
	if err != nil || agentPrompt.ID != "prompt_01" || agentPrompt.Answer != "yes" || fake.resolveAgentPromptID != "prompt_01" {
		t.Fatalf("resolve agent prompt = %#v, id = %q, req = %#v, err = %v", agentPrompt, fake.resolveAgentPromptID, fake.resolveAgentPromptReq, err)
	}
	agentEvent, err := service.MarkAgentBridgeEventRead(ctx, protocol.MarkAgentBridgeEventReadRequest{ID: "event_01"})
	if err != nil || agentEvent.ID != "event_01" || fake.markAgentEventReadReq.ID != "event_01" {
		t.Fatalf("mark agent event read = %#v, req = %#v, err = %v", agentEvent, fake.markAgentEventReadReq, err)
	}
	events, err := service.ListAgentBridgeEvents(ctx, protocol.ListAgentBridgeEventsRequest{Status: "open"})
	if err != nil || fake.listAgentEventsReq.Status != "open" {
		t.Fatalf("agent events = %#v, req = %#v, err = %v", events, fake.listAgentEventsReq, err)
	}
	if _, err := service.AgentHookLogStatus(ctx); err != nil {
		t.Fatalf("agent hook log status: %v", err)
	}
	hookLogEnabled := true
	logStatus, err := service.SetAgentHookLogSettings(ctx, protocol.SetAgentHookLogSettingsRequest{Enabled: &hookLogEnabled})
	if err != nil || !logStatus.Enabled || fake.setAgentHookLogReq.Enabled == nil || !*fake.setAgentHookLogReq.Enabled {
		t.Fatalf("set agent hook log = %#v, req = %#v, err = %v", logStatus, fake.setAgentHookLogReq, err)
	}
	if _, err := service.ClearAgentHookLog(ctx); err != nil || !fake.clearAgentHookLogCalled {
		t.Fatalf("clear agent hook log: called = %v, err = %v", fake.clearAgentHookLogCalled, err)
	}
	if _, err := service.OpenAgentHookLog(ctx); err != nil || !fake.openAgentHookLogCalled {
		t.Fatalf("open agent hook log: called = %v, err = %v", fake.openAgentHookLogCalled, err)
	}
	integrations, err := service.ListAgentHookIntegrations(ctx)
	if err != nil || len(integrations) != 1 || integrations[0].Provider != "claude" {
		t.Fatalf("agent hook integrations = %#v, err = %v", integrations, err)
	}
	checkedIntegration, err := service.CheckAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || checkedIntegration.Provider != "claude" || fake.checkAgentHookReq.Provider != "claude" {
		t.Fatalf("check agent hook integration = %#v, req = %#v, err = %v", checkedIntegration, fake.checkAgentHookReq, err)
	}
	installedIntegration, err := service.InstallAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "codex"})
	if err != nil || installedIntegration.Provider != "codex" || fake.installAgentHookReq.Provider != "codex" {
		t.Fatalf("install agent hook integration = %#v, req = %#v, err = %v", installedIntegration, fake.installAgentHookReq, err)
	}
	removedIntegration, err := service.RemoveAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "codex"})
	if err != nil || removedIntegration.Provider != "codex" || fake.removeAgentHookReq.Provider != "codex" {
		t.Fatalf("remove agent hook integration = %#v, req = %#v, err = %v", removedIntegration, fake.removeAgentHookReq, err)
	}
	plugins, err := service.ListPlugins(ctx)
	if err != nil || len(plugins) != 1 || plugins[0].ID != "github" {
		t.Fatalf("list plugins = %#v, err = %v", plugins, err)
	}
	plugins, err = service.RescanPlugins(ctx)
	if err != nil || len(plugins) != 1 || !fake.rescanPluginsCalled {
		t.Fatalf("rescan plugins = %#v, called = %v, err = %v", plugins, fake.rescanPluginsCalled, err)
	}
	plugin, err := service.TrustPlugin(ctx, "github")
	if err != nil || !plugin.Trusted || fake.trustPluginID != "github" {
		t.Fatalf("trust plugin = %#v, id = %q, err = %v", plugin, fake.trustPluginID, err)
	}
	plugin, err = service.UntrustPlugin(ctx, "github")
	if err != nil || plugin.Trusted || fake.untrustPluginID != "github" {
		t.Fatalf("untrust plugin = %#v, id = %q, err = %v", plugin, fake.untrustPluginID, err)
	}
	registryPlugins, err := service.ListRegistryPlugins(ctx)
	if err != nil || len(registryPlugins) != 1 || registryPlugins[0].Registry != "phin-tech" {
		t.Fatalf("registry plugins = %#v, err = %v", registryPlugins, err)
	}
	plugin, err = service.InstallPlugin(ctx, "phin-tech", "github")
	if err != nil || fake.installPluginRegistry != "phin-tech" || fake.installPluginID != "github" {
		t.Fatalf("install plugin = %#v, registry = %q, id = %q, err = %v", plugin, fake.installPluginRegistry, fake.installPluginID, err)
	}
	project, err = service.RunPluginProjectAttachmentTemplate(ctx, "github", "issue", protocol.RunPluginProjectAttachmentTemplateRequest{ProjectID: "proj_01", Values: map[string]string{"issue": "1"}})
	if err != nil || len(project.Attachments) != 1 || fake.runPluginTemplateID != "issue" {
		t.Fatalf("run plugin template = %#v, template = %q, err = %v", project, fake.runPluginTemplateID, err)
	}
	httpForwards, err := service.ListHTTPForwards(ctx)
	if err != nil || len(httpForwards) != 1 || httpForwards[0].ID != "fwd_01" {
		t.Fatalf("list http forwards = %#v, err = %v", httpForwards, err)
	}
	if _, err := service.StartHTTPForward(ctx, protocol.StartHTTPForwardRequest{TargetURL: "http://127.0.0.1:4966"}); err == nil {
		t.Fatalf("expected start error without HTTP client")
	}
	if err := service.StopHTTPForward(ctx, "fwd_01"); err == nil {
		t.Fatalf("expected stop error without HTTP client")
	}
}

func TestServiceLoadsAndSavesAppSettings(t *testing.T) {
	store := &appSettingsStoreFake{settings: appsettings.Settings{StartupView: appsettings.StartupViewKanban}}
	service := wailsapp.NewServiceWithSettings(&runtimeClientFake{}, store)
	ctx := context.Background()

	loaded, err := service.LoadAppSettings(ctx)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !store.loaded || loaded.StartupView != appsettings.StartupViewKanban {
		t.Fatalf("loaded = %#v, store loaded = %v", loaded, store.loaded)
	}

	saved, err := service.SaveAppSettings(ctx, appsettings.Settings{StartupView: appsettings.StartupViewSessions})
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if !store.saved || saved.StartupView != appsettings.StartupViewSessions || store.settings.StartupView != appsettings.StartupViewSessions {
		t.Fatalf("saved = %#v, store = %#v, saved flag = %v", saved, store.settings, store.saved)
	}
}

func TestServiceLoadKeybindingsReportsEffectiveAccelerators(t *testing.T) {
	store := &appSettingsStoreFake{settings: appsettings.Settings{
		Keybindings: map[string]string{appmenu.CommandOpenPreferences: "Cmd+Shift+P"},
	}}
	service := wailsapp.NewServiceWithSettings(&runtimeClientFake{}, store)

	view, err := service.LoadKeybindings(context.Background())
	if err != nil {
		t.Fatalf("load keybindings: %v", err)
	}
	var found bool
	for _, cmd := range view.Commands {
		if cmd.ID == appmenu.CommandOpenPreferences {
			found = true
			if cmd.Accelerator != "Cmd+Shift+P" {
				t.Fatalf("accelerator = %q, want override", cmd.Accelerator)
			}
		}
	}
	if !found {
		t.Fatalf("view missing open-preferences")
	}
}

func TestServiceSaveKeybindingsPersistsAndUpdatesMenu(t *testing.T) {
	store := &appSettingsStoreFake{settings: appsettings.Settings{StartupView: appsettings.StartupViewSessions}}
	menu := &menuControllerFake{}
	service := wailsapp.NewServiceWithSettings(&runtimeClientFake{}, store)
	wailsapp.AttachMenuController(service, menu)

	view, err := service.SaveKeybindings(context.Background(), map[string]string{appmenu.CommandOpenPreferences: "Cmd+Shift+P"})
	if err != nil {
		t.Fatalf("save keybindings: %v", err)
	}
	if !store.saved || store.settings.Keybindings[appmenu.CommandOpenPreferences] != "Cmd+Shift+P" {
		t.Fatalf("store = %#v, saved = %v", store.settings, store.saved)
	}
	if menu.keybindingsCalls != 1 || menu.lastSettings.Keybindings[appmenu.CommandOpenPreferences] != "Cmd+Shift+P" {
		t.Fatalf("menu calls = %d, last = %#v", menu.keybindingsCalls, menu.lastSettings)
	}
	if len(view.Commands) == 0 {
		t.Fatalf("view should report commands")
	}
}

func TestServiceSaveKeybindingsRejectsInvalidOverride(t *testing.T) {
	store := &appSettingsStoreFake{}
	menu := &menuControllerFake{}
	service := wailsapp.NewServiceWithSettings(&runtimeClientFake{}, store)
	wailsapp.AttachMenuController(service, menu)

	if _, err := service.SaveKeybindings(context.Background(), map[string]string{"std-quit": "Cmd+Escape"}); err == nil {
		t.Fatalf("expected error for non-editable command")
	}
	if store.saved {
		t.Fatalf("invalid override must not persist")
	}
	if menu.keybindingsCalls != 0 {
		t.Fatalf("invalid override must not touch the menu")
	}
}

func TestServiceSyncSessionMenuForwardsToController(t *testing.T) {
	menu := &menuControllerFake{}
	service := wailsapp.NewService(&runtimeClientFake{})
	wailsapp.AttachMenuController(service, menu)

	sessions := []appmenu.SessionRef{{ID: "sess_01", Name: "alpha"}}
	if err := service.SyncSessionMenu(context.Background(), sessions); err != nil {
		t.Fatalf("sync session menu: %v", err)
	}
	if menu.sessionCalls != 1 || len(menu.lastSessions) != 1 || menu.lastSessions[0].Name != "alpha" {
		t.Fatalf("menu sessions = %#v, calls = %d", menu.lastSessions, menu.sessionCalls)
	}
}

func TestServiceSyncSessionMenuWithoutControllerIsNoOp(t *testing.T) {
	service := wailsapp.NewService(&runtimeClientFake{})
	if err := service.SyncSessionMenu(context.Background(), []appmenu.SessionRef{{ID: "s"}}); err != nil {
		t.Fatalf("sync session menu without controller: %v", err)
	}
}

type menuControllerFake struct {
	keybindingsCalls int
	sessionCalls     int
	lastSettings     appsettings.Settings
	lastSessions     []appmenu.SessionRef
}

func (f *menuControllerFake) SetKeybindings(settings appsettings.Settings) {
	f.keybindingsCalls++
	f.lastSettings = settings
}

func (f *menuControllerFake) SetSessions(sessions []appmenu.SessionRef) {
	f.sessionCalls++
	f.lastSessions = sessions
}

type appSettingsStoreFake struct {
	settings appsettings.Settings
	loaded   bool
	saved    bool
}

func (f *appSettingsStoreFake) Load(context.Context) (appsettings.Settings, error) {
	f.loaded = true
	return f.settings, nil
}

func (f *appSettingsStoreFake) Save(_ context.Context, settings appsettings.Settings) (appsettings.Settings, error) {
	f.saved = true
	f.settings = settings
	return settings, nil
}

type runtimeClientFake struct {
	sessions            []session.Session
	created             protocol.CreatedSession
	split               protocol.SplitPaneResult
	output              protocol.OutputSnapshot
	ptys                []protocol.PTYInfo
	ptyHistory          []protocol.PTYHistorySummary
	selectedPTYHistory  protocol.PTYHistory
	event               protocol.NextEventResponse
	worktrunk           protocol.WorktrunkStatus
	worktrees           []protocol.Worktree
	createdWorktree     protocol.CreatedWorktree
	projects            []protocol.Project
	projectDetail       protocol.ProjectDetail
	workflowTemplates   []protocol.WorkflowTemplate
	workflowDefinitions []protocol.WorkflowDefinitionRecord
	promptTemplates     []protocol.PromptTemplate
	agentProfiles       []protocol.AgentProfile
	workItems           []protocol.WorkItem
	workItemLinks       []protocol.WorkItemLink
	readyWork           protocol.ReadyWorkExplanation
	runs                []protocol.WorkItemRun
	httpForwards        []protocol.HTTPForward
	agentApprovals      []protocol.AgentBridgeApproval
	agentPrompts        []protocol.AgentPrompt
	agentEvents         []protocol.AgentBridgeEvent
	agentIntegrations   []protocol.AgentHookIntegration
	agentHookLog        protocol.AgentHookLogStatus
	clearResponse       protocol.ClearDaemonResponse

	clearCalled       bool
	createReq         protocol.CreateSessionRequest
	splitReq          protocol.SplitPaneRequest
	setRootReq        protocol.SetSessionRootDirRequest
	setProjectReq     protocol.SetSessionProjectRequest
	setPaneDirReq     protocol.SetPaneWorkingDirRequest
	startPanePTYReq   protocol.StartPanePTYRequest
	restartPanePTYReq protocol.RestartPanePTYRequest
	detachPanePTYReq  protocol.DetachPanePTYRequest
	closeSessionReq   protocol.CloseSessionRequest
	closePaneReq      protocol.ClosePaneRequest
	writeReq          protocol.WritePTYRequest
	resizeReq         protocol.ResizePTYRequest
	killReq           protocol.KillPTYRequest
	deletePTYReq      protocol.DeletePTYRequest
	outputReq         protocol.OutputRequest
	readPTYHistoryID  string
	nextEventReq      protocol.NextEventRequest

	detectWorktrunkReq              protocol.DetectWorktrunkRequest
	listWorktreesReq                protocol.ListWorktreesRequest
	createWorktreeReq               protocol.CreateWorktreeRequest
	removeWorktreeReq               protocol.RemoveWorktreeRequest
	createProjectReq                protocol.CreateProjectRequest
	updateProjectID                 string
	updateProjectReq                protocol.UpdateProjectRequest
	deleteProjectID                 string
	deleteProjectReq                protocol.DeleteProjectRequest
	projectDetailID                 string
	addProjectAttachmentReq         protocol.AddProjectAttachmentRequest
	updateProjectAttachmentID       string
	updateProjectAttachmentReq      protocol.UpdateProjectAttachmentRequest
	deleteProjectAttachmentID       string
	projectContextID                string
	validateWorkflowDefinitionReq   protocol.ValidateWorkflowDefinitionRequest
	validateWorkflowFileReq         protocol.ValidateWorkflowDefinitionFileRequest
	importWorkflowDefinitionReq     protocol.ImportWorkflowDefinitionRequest
	importWorkflowFileReq           protocol.ImportWorkflowDefinitionFileRequest
	exportWorkflowFileReq           protocol.ExportWorkflowDefinitionFileRequest
	deleteWorkflowDefinitionID      string
	deleteWorkflowDefinitionVersion int
	workflowMigrationProjectID      string
	workflowMigrationReq            protocol.PlanProjectWorkflowMigrationRequest
	setProjectWorkflowDefinitionID  string
	setProjectWorkflowDefinitionReq protocol.SetProjectWorkflowDefinitionRequest
	listWorkItemsProjectID          string
	listWorkflowActionsWorkItemID   string
	createWorkItemReq               protocol.CreateWorkItemRequest
	updateWorkItemReq               protocol.UpdateWorkItemRequest
	moveWorkItemReq                 protocol.MoveWorkItemRequest
	addWorkItemLinkReq              protocol.AddWorkItemLinkRequest
	listWorkItemLinksID             string
	readyWorkReq                    protocol.ReadyWorkRequest
	bindWorkItemReq                 protocol.BindWorkItemWorktreeRequest
	addWorkItemAttachmentReq        protocol.AddWorkItemAttachmentRequest
	deleteWorkItemReq               protocol.DeleteWorkItemRequest
	listRunsWorkItemID              string
	startRunReq                     protocol.StartWorkItemRunRequest
	cancelRunReq                    protocol.CancelWorkItemRunRequest
	reportStatusReq                 protocol.ReportStatusRequest
	listStatusEventsReq             protocol.ListStatusEventsRequest
	markStatusReadReq               protocol.MarkStatusEventReadRequest
	agentBridgeHookID               string
	agentBridgeHookReq              protocol.AgentBridgeHookRequest
	recordAgentHookReq              protocol.AgentBridgeHookRequest
	listAgentApprovalsReq           protocol.ListAgentBridgeApprovalsRequest
	listAgentPromptsReq             protocol.ListAgentPromptsRequest
	listAgentEventsReq              protocol.ListAgentBridgeEventsRequest
	markAgentEventReadReq           protocol.MarkAgentBridgeEventReadRequest
	resolveAgentApprovalID          string
	resolveAgentApprovalReq         protocol.ResolveAgentBridgeApprovalRequest
	resolveAgentPromptID            string
	resolveAgentPromptReq           protocol.ResolveAgentPromptRequest
	checkAgentHookReq               protocol.AgentHookIntegrationRequest
	installAgentHookReq             protocol.AgentHookIntegrationRequest
	removeAgentHookReq              protocol.AgentHookIntegrationRequest
	setAgentHookLogReq              protocol.SetAgentHookLogSettingsRequest
	clearAgentHookLogCalled         bool
	openAgentHookLogCalled          bool
	rescanPluginsCalled             bool
	trustPluginID                   string
	untrustPluginID                 string
	installPluginRegistry           string
	installPluginID                 string
	runPluginTemplateID             string
	onboardingApplyReq              protocol.OnboardingApplyRequest
	onboardingStatus                protocol.OnboardingStatus
	createForwardReq                protocol.CreateHTTPForwardRequest
	deleteForwardID                 string
}

func (f *runtimeClientFake) ClearDaemon(context.Context, protocol.ClearDaemonRequest) (protocol.ClearDaemonResponse, error) {
	f.clearCalled = true
	return f.clearResponse, nil
}

func (f *runtimeClientFake) OnboardingStatus(context.Context) (protocol.OnboardingStatus, error) {
	return f.onboardingStatus, nil
}

func (f *runtimeClientFake) ApplyOnboarding(_ context.Context, req protocol.OnboardingApplyRequest) (protocol.OnboardingStatus, error) {
	f.onboardingApplyReq = req
	return f.onboardingStatus, nil
}

func (f *runtimeClientFake) ListSessions(context.Context) ([]session.Session, error) {
	return f.sessions, nil
}

func (f *runtimeClientFake) CreateSession(_ context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	f.createReq = req
	return f.created, nil
}

func (f *runtimeClientFake) SplitPane(_ context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	f.splitReq = req
	return f.split, nil
}

func (f *runtimeClientFake) SetSessionRootDir(_ context.Context, req protocol.SetSessionRootDirRequest) (session.Session, error) {
	f.setRootReq = req
	return session.Session{ID: req.SessionID, RootDir: req.RootDir}, nil
}

func (f *runtimeClientFake) SetSessionProject(_ context.Context, req protocol.SetSessionProjectRequest) (session.Session, error) {
	f.setProjectReq = req
	return session.Session{ID: req.SessionID, ProjectID: req.ProjectID, RootDir: "/repo"}, nil
}

func (f *runtimeClientFake) SetPaneWorkingDir(_ context.Context, req protocol.SetPaneWorkingDirRequest) (session.Session, error) {
	f.setPaneDirReq = req
	return session.Session{ID: req.SessionID}, nil
}

func (f *runtimeClientFake) StartPanePTY(_ context.Context, req protocol.StartPanePTYRequest) (protocol.StartedPanePTY, error) {
	f.startPanePTYReq = req
	return protocol.StartedPanePTY{Session: session.Session{ID: req.SessionID}, PTYID: "pty_03"}, nil
}

func (f *runtimeClientFake) RestartPanePTY(_ context.Context, req protocol.RestartPanePTYRequest) (protocol.RestartedPanePTY, error) {
	f.restartPanePTYReq = req
	return protocol.RestartedPanePTY{Session: session.Session{ID: req.SessionID}, PTYID: "pty_04", OldPTYID: "pty_03"}, nil
}

func (f *runtimeClientFake) DetachPanePTY(_ context.Context, req protocol.DetachPanePTYRequest) (protocol.DetachedPanePTY, error) {
	f.detachPanePTYReq = req
	return protocol.DetachedPanePTY{Session: session.Session{ID: req.SessionID}, PTYID: "pty_03"}, nil
}

func (f *runtimeClientFake) CloseSession(_ context.Context, req protocol.CloseSessionRequest) ([]session.Session, error) {
	f.closeSessionReq = req
	return []session.Session{{ID: "sess_01"}}, nil
}

func (f *runtimeClientFake) ClosePane(_ context.Context, req protocol.ClosePaneRequest) (session.Session, error) {
	f.closePaneReq = req
	return session.Session{ID: req.SessionID}, nil
}

func (f *runtimeClientFake) WritePTY(_ context.Context, req protocol.WritePTYRequest) error {
	f.writeReq = req
	return nil
}

func (f *runtimeClientFake) ResizePTY(_ context.Context, req protocol.ResizePTYRequest) error {
	f.resizeReq = req
	return nil
}

func (f *runtimeClientFake) KillPTY(_ context.Context, req protocol.KillPTYRequest) (protocol.PTYInfo, error) {
	f.killReq = req
	return protocol.PTYInfo{ID: req.PTYID, Status: "killed"}, nil
}

func (f *runtimeClientFake) DeletePTY(_ context.Context, req protocol.DeletePTYRequest) error {
	f.deletePTYReq = req
	return nil
}

func (f *runtimeClientFake) Output(_ context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	f.outputReq = req
	return f.output, nil
}

func (f *runtimeClientFake) ListPTYs(context.Context) ([]protocol.PTYInfo, error) {
	return f.ptys, nil
}

func (f *runtimeClientFake) ListPTYHistory(context.Context) ([]protocol.PTYHistorySummary, error) {
	return f.ptyHistory, nil
}

func (f *runtimeClientFake) ReadPTYHistory(_ context.Context, ptyID string) (protocol.PTYHistory, error) {
	f.readPTYHistoryID = ptyID
	return f.selectedPTYHistory, nil
}

func (f *runtimeClientFake) NextEvent(_ context.Context, req protocol.NextEventRequest) (protocol.NextEventResponse, error) {
	f.nextEventReq = req
	return f.event, nil
}

func (f *runtimeClientFake) SendMail(_ context.Context, req protocol.SendMailRequest) (protocol.MailMessage, error) {
	return protocol.MailMessage{}, nil
}

func (f *runtimeClientFake) ListMail(_ context.Context, req protocol.ListMailRequest) ([]protocol.MailMessage, error) {
	return nil, nil
}

func (f *runtimeClientFake) NextMail(_ context.Context, req protocol.NextMailRequest) (protocol.NextMailResponse, error) {
	return protocol.NextMailResponse{}, nil
}

func (f *runtimeClientFake) MarkMailRead(_ context.Context, mailID string, req protocol.MarkMailReadRequest) (protocol.MailMessage, error) {
	return protocol.MailMessage{}, nil
}

func (f *runtimeClientFake) ReplyMail(_ context.Context, mailID string, req protocol.ReplyMailRequest) (protocol.MailMessage, error) {
	return protocol.MailMessage{}, nil
}

func (f *runtimeClientFake) DetectWorktrunk(_ context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error) {
	f.detectWorktrunkReq = req
	return f.worktrunk, nil
}

func (f *runtimeClientFake) ListWorktrees(_ context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error) {
	f.listWorktreesReq = req
	return f.worktrees, nil
}

func (f *runtimeClientFake) CreateWorktree(_ context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error) {
	f.createWorktreeReq = req
	return f.createdWorktree, nil
}

func (f *runtimeClientFake) RemoveWorktree(_ context.Context, req protocol.RemoveWorktreeRequest) error {
	f.removeWorktreeReq = req
	return nil
}

func (f *runtimeClientFake) ListProjects(context.Context) ([]protocol.Project, error) {
	return f.projects, nil
}

func (f *runtimeClientFake) CreateProject(_ context.Context, req protocol.CreateProjectRequest) (protocol.Project, error) {
	f.createProjectReq = req
	return protocol.Project{ID: "proj_02", Name: req.Name, RootDir: req.RootDir}, nil
}

func (f *runtimeClientFake) UpdateProject(_ context.Context, projectID string, req protocol.UpdateProjectRequest) (protocol.Project, error) {
	f.updateProjectID = projectID
	f.updateProjectReq = req
	project := f.projectDetail.Project
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	return project, nil
}

func (f *runtimeClientFake) DeleteProject(_ context.Context, projectID string, req protocol.DeleteProjectRequest) (protocol.Project, error) {
	f.deleteProjectID = projectID
	f.deleteProjectReq = req
	return protocol.Project{ID: projectID}, nil
}

func (f *runtimeClientFake) GetProjectDetail(_ context.Context, projectID string) (protocol.ProjectDetail, error) {
	f.projectDetailID = projectID
	return f.projectDetail, nil
}

func (f *runtimeClientFake) AddProjectAttachment(_ context.Context, req protocol.AddProjectAttachmentRequest) (protocol.Project, error) {
	f.addProjectAttachmentReq = req
	project := f.projectDetail.Project
	project.Attachments = append(project.Attachments, protocol.Attachment{ID: "att_01", Kind: req.Kind, Title: req.Title, Path: req.Path, URL: req.URL, Note: req.Note, Provider: req.Provider, Target: req.Target, IncludeInContext: req.IncludeInContext})
	return project, nil
}

func (f *runtimeClientFake) UpdateProjectAttachment(_ context.Context, attachmentID string, req protocol.UpdateProjectAttachmentRequest) (protocol.Project, error) {
	f.updateProjectAttachmentID = attachmentID
	f.updateProjectAttachmentReq = req
	project := f.projectDetail.Project
	project.Attachments = append(project.Attachments, protocol.Attachment{ID: attachmentID})
	if req.Title != nil {
		project.Attachments[0].Title = *req.Title
	}
	return project, nil
}

func (f *runtimeClientFake) DeleteProjectAttachment(_ context.Context, attachmentID string, _ protocol.DeleteProjectAttachmentRequest) (protocol.Project, error) {
	f.deleteProjectAttachmentID = attachmentID
	project := f.projectDetail.Project
	project.Attachments = nil
	return project, nil
}

func (f *runtimeClientFake) GetProjectContext(_ context.Context, projectID string) (protocol.ProjectContext, error) {
	f.projectContextID = projectID
	return protocol.ProjectContext{ProjectID: f.projectDetail.Project.ID}, nil
}

func (f *runtimeClientFake) ListWorkflowTemplates(context.Context) ([]protocol.WorkflowTemplate, error) {
	return f.workflowTemplates, nil
}

func (f *runtimeClientFake) ListWorkflowDefinitions(context.Context) ([]protocol.WorkflowDefinitionRecord, error) {
	return f.workflowDefinitions, nil
}

func (f *runtimeClientFake) ValidateWorkflowDefinition(_ context.Context, req protocol.ValidateWorkflowDefinitionRequest) (protocol.WorkflowValidationReport, error) {
	f.validateWorkflowDefinitionReq = req
	return protocol.WorkflowValidationReport{Valid: req.Definition.ID != ""}, nil
}

func (f *runtimeClientFake) ValidateWorkflowDefinitionFile(_ context.Context, req protocol.ValidateWorkflowDefinitionFileRequest) (protocol.WorkflowValidationReport, error) {
	f.validateWorkflowFileReq = req
	return protocol.WorkflowValidationReport{Valid: req.Path != ""}, nil
}

func (f *runtimeClientFake) ImportWorkflowDefinition(_ context.Context, req protocol.ImportWorkflowDefinitionRequest) (protocol.WorkflowDefinitionRecord, error) {
	f.importWorkflowDefinitionReq = req
	record := protocol.WorkflowDefinitionRecord{
		ID:         req.Definition.ID,
		Version:    req.Definition.Version,
		Definition: req.Definition,
		SourcePath: req.SourcePath,
	}
	f.workflowDefinitions = append(f.workflowDefinitions, record)
	return record, nil
}

func (f *runtimeClientFake) ImportWorkflowDefinitionFile(_ context.Context, req protocol.ImportWorkflowDefinitionFileRequest) (protocol.WorkflowDefinitionRecord, error) {
	f.importWorkflowFileReq = req
	return protocol.WorkflowDefinitionRecord{ID: "file", Version: 1, SourcePath: req.Path}, nil
}

func (f *runtimeClientFake) ExportWorkflowDefinitionFile(_ context.Context, req protocol.ExportWorkflowDefinitionFileRequest) error {
	f.exportWorkflowFileReq = req
	return nil
}

func (f *runtimeClientFake) DeleteWorkflowDefinition(_ context.Context, id string, version int) (protocol.WorkflowDefinitionRecord, error) {
	f.deleteWorkflowDefinitionID = id
	f.deleteWorkflowDefinitionVersion = version
	return protocol.WorkflowDefinitionRecord{ID: id, Version: version}, nil
}

func (f *runtimeClientFake) SetProjectWorkflowDefinition(_ context.Context, projectID string, req protocol.SetProjectWorkflowDefinitionRequest) (protocol.Project, error) {
	f.setProjectWorkflowDefinitionID = projectID
	f.setProjectWorkflowDefinitionReq = req
	project := f.projectDetail.Project
	project.Workflow.DefinitionID = req.ID
	project.Workflow.DefinitionVersion = req.Version
	return project, nil
}

func (f *runtimeClientFake) PlanProjectWorkflowMigration(_ context.Context, projectID string, req protocol.PlanProjectWorkflowMigrationRequest) (protocol.WorkflowMigrationPlan, error) {
	f.workflowMigrationProjectID = projectID
	f.workflowMigrationReq = req
	return protocol.WorkflowMigrationPlan{ProjectID: projectID, TargetID: req.ID, TargetVersion: req.Version}, nil
}

func (f *runtimeClientFake) ListAgentProfiles(context.Context) ([]protocol.AgentProfile, error) {
	return f.agentProfiles, nil
}

func (f *runtimeClientFake) ListPromptTemplates(context.Context) ([]protocol.PromptTemplate, error) {
	return f.promptTemplates, nil
}

func (f *runtimeClientFake) ListWorkItems(_ context.Context, projectID string) ([]protocol.WorkItem, error) {
	f.listWorkItemsProjectID = projectID
	return f.workItems, nil
}

func (f *runtimeClientFake) CreateWorkItem(_ context.Context, req protocol.CreateWorkItemRequest) (protocol.WorkItem, error) {
	f.createWorkItemReq = req
	return protocol.WorkItem{ID: "wi_02", ProjectID: req.ProjectID, Number: 2, Title: req.Title}, nil
}

func (f *runtimeClientFake) UpdateWorkItem(_ context.Context, req protocol.UpdateWorkItemRequest) (protocol.WorkItem, error) {
	f.updateWorkItemReq = req
	item := protocol.WorkItem{ID: req.ID}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.BodyMarkdown != nil {
		item.BodyMarkdown = *req.BodyMarkdown
	}
	return item, nil
}

func (f *runtimeClientFake) MoveWorkItem(_ context.Context, req protocol.MoveWorkItemRequest) (protocol.WorkItem, error) {
	f.moveWorkItemReq = req
	return protocol.WorkItem{ID: req.ID, StageID: req.StageID}, nil
}

func (f *runtimeClientFake) ListWorkItemWorkflowActions(_ context.Context, workItemID string) ([]protocol.WorkflowActionAvailability, error) {
	f.listWorkflowActionsWorkItemID = workItemID
	return []protocol.WorkflowActionAvailability{{
		Action:    workitem.WorkflowActionDefinition{ID: workitem.WorkflowActionStartPlanning},
		Enabled:   true,
		InputKind: workitem.WorkflowActionInputRun,
	}}, nil
}

func (f *runtimeClientFake) AddWorkItemLink(_ context.Context, req protocol.AddWorkItemLinkRequest) (protocol.WorkItemLink, error) {
	f.addWorkItemLinkReq = req
	if len(f.workItemLinks) == 0 {
		return protocol.WorkItemLink{}, nil
	}
	return f.workItemLinks[0], nil
}

func (f *runtimeClientFake) ListWorkItemLinks(_ context.Context, workItemID string) ([]protocol.WorkItemLink, error) {
	f.listWorkItemLinksID = workItemID
	return f.workItemLinks, nil
}

func (f *runtimeClientFake) ReadyWork(_ context.Context, req protocol.ReadyWorkRequest) (protocol.ReadyWorkExplanation, error) {
	f.readyWorkReq = req
	return f.readyWork, nil
}

func (f *runtimeClientFake) StartPlanning(_ context.Context, req protocol.StartPlanningRequest) (protocol.WorkItemRun, error) {
	return protocol.WorkItemRun{ID: "run_plan", WorkItemID: req.WorkItemID, PromptTemplateID: "plan"}, nil
}

func (f *runtimeClientFake) SubmitDraftPlan(_ context.Context, req protocol.SubmitDraftPlanRequest) (protocol.Artifact, error) {
	return protocol.Artifact{ID: "artifact_plan", WorkItemID: req.WorkItemID, Kind: "plan", Status: "draft"}, nil
}

func (f *runtimeClientFake) ApprovePlan(_ context.Context, req protocol.ApprovePlanRequest) (protocol.WorkItem, error) {
	return protocol.WorkItem{ID: req.WorkItemID, StageID: "ready"}, nil
}

func (f *runtimeClientFake) StartExecution(_ context.Context, req protocol.StartExecutionRequest) (protocol.WorkItemRun, error) {
	return protocol.WorkItemRun{ID: "run_exec", WorkItemID: req.WorkItemID, PromptTemplateID: "implement"}, nil
}

func (f *runtimeClientFake) QueueExecution(_ context.Context, req protocol.QueueExecutionRequest) (protocol.WorkItemRun, error) {
	return protocol.WorkItemRun{ID: "run_exec", WorkItemID: req.WorkItemID, PromptTemplateID: "implement", Status: "queued"}, nil
}

func (f *runtimeClientFake) LaunchExecution(_ context.Context, req protocol.LaunchExecutionRequest) (protocol.WorkItemRun, error) {
	return protocol.WorkItemRun{ID: "run_exec", WorkItemID: req.WorkItemID, PromptTemplateID: "implement", Status: "running"}, nil
}

func (f *runtimeClientFake) AskQuestion(_ context.Context, req protocol.AskQuestionRequest) (protocol.Question, error) {
	return protocol.Question{ID: "question_01", WorkItemID: req.WorkItemID, RunID: req.RunID, Prompt: req.Prompt, Status: "open"}, nil
}

func (f *runtimeClientFake) AnswerQuestion(_ context.Context, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	return protocol.Question{ID: req.ID, Answer: req.Answer, Status: "answered"}, nil
}

func (f *runtimeClientFake) CompleteExecution(_ context.Context, req protocol.CompleteExecutionRequest) (protocol.WorkItem, error) {
	return protocol.WorkItem{ID: req.WorkItemID, StageID: "review"}, nil
}

func (f *runtimeClientFake) SubmitReviewFeedback(_ context.Context, req protocol.SubmitReviewFeedbackRequest) (protocol.Artifact, error) {
	return protocol.Artifact{ID: "feedback_01", WorkItemID: req.WorkItemID, RunID: req.RunID, Kind: "feedback", Status: "approved"}, nil
}

func (f *runtimeClientFake) BindWorkItemWorktree(_ context.Context, req protocol.BindWorkItemWorktreeRequest) (protocol.WorkItem, error) {
	f.bindWorkItemReq = req
	return protocol.WorkItem{ID: req.ID, Worktree: &protocol.WorktreeBinding{Branch: req.Branch, WorktreePath: req.WorktreePath}}, nil
}

func (f *runtimeClientFake) AddWorkItemAttachment(_ context.Context, req protocol.AddWorkItemAttachmentRequest) (protocol.WorkItem, error) {
	f.addWorkItemAttachmentReq = req
	return protocol.WorkItem{ID: req.WorkItemID, Attachments: []protocol.Attachment{{ID: "att_01", Kind: req.Kind, Path: req.Path}}}, nil
}

func (f *runtimeClientFake) DeleteWorkItem(_ context.Context, req protocol.DeleteWorkItemRequest) (protocol.WorkItem, error) {
	f.deleteWorkItemReq = req
	return protocol.WorkItem{ID: req.ID}, nil
}

func (f *runtimeClientFake) ListWorkItemRuns(_ context.Context, workItemID string) ([]protocol.WorkItemRun, error) {
	f.listRunsWorkItemID = workItemID
	return f.runs, nil
}

func (f *runtimeClientFake) StartWorkItemRun(_ context.Context, req protocol.StartWorkItemRunRequest) (protocol.WorkItemRun, error) {
	f.startRunReq = req
	return protocol.WorkItemRun{ID: "run_02", WorkItemID: req.WorkItemID, Status: "queued", Preset: req.Preset, PromptTemplateID: req.PromptTemplateID}, nil
}

func (f *runtimeClientFake) LaunchWorkItemRun(_ context.Context, req protocol.LaunchWorkItemRunRequest) (protocol.WorkItemRun, error) {
	return protocol.WorkItemRun{ID: req.ID, Status: "running"}, nil
}

func (f *runtimeClientFake) CancelWorkItemRun(_ context.Context, req protocol.CancelWorkItemRunRequest) (protocol.WorkItemRun, error) {
	f.cancelRunReq = req
	return protocol.WorkItemRun{ID: req.ID, Status: "cancelled"}, nil
}

func (f *runtimeClientFake) ApproveDone(_ context.Context, req protocol.ApproveDoneRequest) (protocol.WorkItem, error) {
	return protocol.WorkItem{ID: req.WorkItemID, StageID: "done"}, nil
}

func (f *runtimeClientFake) ListArtifacts(_ context.Context, workItemID string) ([]protocol.Artifact, error) {
	return []protocol.Artifact{{ID: "artifact_01", WorkItemID: workItemID, Kind: "plan"}}, nil
}

func (f *runtimeClientFake) ListQuestions(_ context.Context, workItemID string) ([]protocol.Question, error) {
	return []protocol.Question{{ID: "question_01", WorkItemID: workItemID, Status: "open"}}, nil
}

func (f *runtimeClientFake) ListGateReports(_ context.Context, workItemID string) ([]protocol.GateReport, error) {
	return []protocol.GateReport{{ID: "gate_01", WorkItemID: workItemID, Status: "pending"}}, nil
}

func (f *runtimeClientFake) CompleteGate(_ context.Context, req protocol.CompleteGateRequest) (protocol.GateReport, error) {
	return protocol.GateReport{ID: req.ID, Status: req.Status, OverrideReason: req.OverrideReason}, nil
}

func (f *runtimeClientFake) ListWorkflowEvents(_ context.Context, workItemID string) ([]protocol.WorkflowEvent, error) {
	return []protocol.WorkflowEvent{{ID: "event_01", WorkItemID: workItemID, Type: "planning_started"}}, nil
}

func (f *runtimeClientFake) ReportStatus(_ context.Context, req protocol.ReportStatusRequest) (protocol.ReportStatusResponse, error) {
	f.reportStatusReq = req
	return protocol.ReportStatusResponse{Event: protocol.StatusEvent{ID: "status_01", Kind: req.Kind, Message: req.Message}}, nil
}

func (f *runtimeClientFake) ListStatusEvents(_ context.Context, req protocol.ListStatusEventsRequest) ([]protocol.StatusEvent, error) {
	f.listStatusEventsReq = req
	return []protocol.StatusEvent{{ID: "status_01", SessionID: req.SessionID}}, nil
}

func (f *runtimeClientFake) MarkStatusEventRead(_ context.Context, req protocol.MarkStatusEventReadRequest) (protocol.StatusEvent, error) {
	f.markStatusReadReq = req
	return protocol.StatusEvent{ID: req.ID}, nil
}

func (f *runtimeClientFake) AgentBridgeHook(_ context.Context, bridgeID string, req protocol.AgentBridgeHookRequest) (protocol.AgentBridgeHookResponse, error) {
	f.agentBridgeHookID = bridgeID
	f.agentBridgeHookReq = req
	return protocol.AgentBridgeHookResponse{}, nil
}

func (f *runtimeClientFake) RecordAgentHookEvent(_ context.Context, req protocol.AgentBridgeHookRequest) (protocol.AgentBridgeEvent, error) {
	f.recordAgentHookReq = req
	return protocol.AgentBridgeEvent{ID: "event_01", Provider: req.Provider, EventName: req.EventName, Status: "pending"}, nil
}

func (f *runtimeClientFake) ListAgentBridgeApprovals(_ context.Context, req protocol.ListAgentBridgeApprovalsRequest) ([]protocol.AgentBridgeApproval, error) {
	f.listAgentApprovalsReq = req
	return f.agentApprovals, nil
}

func (f *runtimeClientFake) ResolveAgentBridgeApproval(_ context.Context, id string, req protocol.ResolveAgentBridgeApprovalRequest) (protocol.AgentBridgeApproval, error) {
	f.resolveAgentApprovalID = id
	f.resolveAgentApprovalReq = req
	return protocol.AgentBridgeApproval{ID: id, Status: "resolved", Decision: protocol.AgentBridgeHookDecision{Action: req.Action, Reason: req.Reason}}, nil
}

func (f *runtimeClientFake) ListAgentPrompts(_ context.Context, req protocol.ListAgentPromptsRequest) ([]protocol.AgentPrompt, error) {
	f.listAgentPromptsReq = req
	return f.agentPrompts, nil
}

func (f *runtimeClientFake) ResolveAgentPrompt(_ context.Context, id string, req protocol.ResolveAgentPromptRequest) (protocol.AgentPrompt, error) {
	f.resolveAgentPromptID = id
	f.resolveAgentPromptReq = req
	return protocol.AgentPrompt{ID: id, Status: "resolved", Answer: req.Answer}, nil
}

func (f *runtimeClientFake) ListAgentBridgeEvents(_ context.Context, req protocol.ListAgentBridgeEventsRequest) ([]protocol.AgentBridgeEvent, error) {
	f.listAgentEventsReq = req
	return f.agentEvents, nil
}

func (f *runtimeClientFake) MarkAgentBridgeEventRead(_ context.Context, req protocol.MarkAgentBridgeEventReadRequest) (protocol.AgentBridgeEvent, error) {
	f.markAgentEventReadReq = req
	return protocol.AgentBridgeEvent{ID: req.ID, Status: "read"}, nil
}

func (f *runtimeClientFake) ListAgentHookIntegrations(context.Context) ([]protocol.AgentHookIntegration, error) {
	return f.agentIntegrations, nil
}

func (f *runtimeClientFake) CheckAgentHookIntegration(_ context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	f.checkAgentHookReq = req
	return protocol.AgentHookIntegration{Provider: req.Provider, Status: "current"}, nil
}

func (f *runtimeClientFake) InstallAgentHookIntegration(_ context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	f.installAgentHookReq = req
	return protocol.AgentHookIntegration{Provider: req.Provider, Status: "current"}, nil
}

func (f *runtimeClientFake) RemoveAgentHookIntegration(_ context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	f.removeAgentHookReq = req
	return protocol.AgentHookIntegration{Provider: req.Provider, Status: "missing"}, nil
}

func (f *runtimeClientFake) AgentHookLogStatus(context.Context) (protocol.AgentHookLogStatus, error) {
	return f.agentHookLog, nil
}

func (f *runtimeClientFake) SetAgentHookLogSettings(_ context.Context, req protocol.SetAgentHookLogSettingsRequest) (protocol.AgentHookLogStatus, error) {
	f.setAgentHookLogReq = req
	if req.Enabled != nil {
		f.agentHookLog.Enabled = *req.Enabled
	}
	if req.ClearAfterSession != nil {
		f.agentHookLog.ClearAfterSession = *req.ClearAfterSession
	}
	return f.agentHookLog, nil
}

func (f *runtimeClientFake) ClearAgentHookLog(context.Context) (protocol.AgentHookLogStatus, error) {
	f.clearAgentHookLogCalled = true
	f.agentHookLog.SizeBytes = 0
	return f.agentHookLog, nil
}

func (f *runtimeClientFake) OpenAgentHookLog(context.Context) (protocol.AgentHookLogStatus, error) {
	f.openAgentHookLogCalled = true
	return f.agentHookLog, nil
}

func (f *runtimeClientFake) ListPlugins(context.Context) ([]protocol.PluginStatus, error) {
	return []protocol.PluginStatus{{ID: "github", Name: "GitHub", Trusted: true, Valid: true}}, nil
}

func (f *runtimeClientFake) RescanPlugins(context.Context) ([]protocol.PluginStatus, error) {
	f.rescanPluginsCalled = true
	return []protocol.PluginStatus{{ID: "github", Name: "GitHub", Trusted: true, Valid: true}}, nil
}

func (f *runtimeClientFake) TrustPlugin(_ context.Context, id string) (protocol.PluginStatus, error) {
	f.trustPluginID = id
	return protocol.PluginStatus{ID: id, Trusted: true, Valid: true}, nil
}

func (f *runtimeClientFake) UntrustPlugin(_ context.Context, id string) (protocol.PluginStatus, error) {
	f.untrustPluginID = id
	return protocol.PluginStatus{ID: id, Trusted: false, Valid: true}, nil
}

func (f *runtimeClientFake) ListRegistryPlugins(context.Context) ([]protocol.RegistryPlugin, error) {
	return []protocol.RegistryPlugin{{Registry: "phin-tech", ID: "github", Name: "GitHub Issues", SourceType: "path"}}, nil
}

func (f *runtimeClientFake) InstallPlugin(_ context.Context, registry, id string) (protocol.PluginStatus, error) {
	f.installPluginRegistry = registry
	f.installPluginID = id
	return protocol.PluginStatus{ID: id, Registry: registry, Valid: true}, nil
}

func (f *runtimeClientFake) RunPluginProjectAttachmentTemplate(_ context.Context, _ string, templateID string, _ protocol.RunPluginProjectAttachmentTemplateRequest) (protocol.Project, error) {
	f.runPluginTemplateID = templateID
	return protocol.Project{ID: "proj_01", Attachments: []protocol.Attachment{{ID: "att_01", Kind: "external", Provider: "github"}}}, nil
}

func (f *runtimeClientFake) CreateHTTPForward(_ context.Context, req protocol.CreateHTTPForwardRequest) (protocol.HTTPForward, error) {
	f.createForwardReq = req
	return protocol.HTTPForward{ID: "fwd_02", TargetURL: req.TargetURL}, nil
}

func (f *runtimeClientFake) ListHTTPForwards(context.Context) ([]protocol.HTTPForward, error) {
	return f.httpForwards, nil
}

func (f *runtimeClientFake) DeleteHTTPForward(_ context.Context, id string) error {
	f.deleteForwardID = id
	return nil
}
