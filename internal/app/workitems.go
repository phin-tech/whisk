package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	bridgeinstaller "github.com/phin-tech/whisk/internal/adapters/agentbridge"
	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type CreateProjectRequest struct {
	Name        string
	Description string
	Slug        string
	RootDir     string
	WorkflowID  string
	Preferences workitem.ProjectPreferences
}

type UpdateProjectRequest struct {
	ID                       string
	Name                     *string
	Description              *string
	Slug                     *string
	UseInteractiveAgentShell *bool
	DefaultPhaseAgents       map[string]string
}

type DeleteProjectRequest struct {
	ID    string
	Actor string
}

type AddProjectAttachmentRequest struct {
	ProjectID        string
	Kind             string
	Scope            string
	Title            string
	Path             string
	URL              string
	Note             string
	Provider         string
	Target           string
	IncludeInContext bool
	Meta             map[string]workitem.MetadataValue
}

type UpdateProjectAttachmentRequest struct {
	ID               string
	ProjectID        string
	Title            *string
	Path             *string
	URL              *string
	Note             *string
	Provider         *string
	Target           *string
	IncludeInContext *bool
	Meta             map[string]workitem.MetadataValue
}

type DeleteProjectAttachmentRequest struct {
	ID        string
	ProjectID string
}

type ProjectDetail struct {
	Project   workitem.Project
	WorkItems []workitem.WorkItem
	Sessions  []session.Session
	Runs      []workitem.WorkItemRun
}

type ProjectContext struct {
	ProjectID string
	Items     []ProjectContextItem
}

type ProjectContextItem struct {
	AttachmentID string
	Kind         string
	Provider     string
	Target       string
	Title        string
	Delivery     string
	ContentType  string
	Content      string
	SourceURL    string
	Error        string
}


type CreateWorkItemRequest struct {
	ProjectID    string
	WorkflowID   string
	Title        string
	BodyMarkdown string
	StageID      string
	Actor        string
}

type UpdateWorkItemRequest struct {
	ID           string
	Title        *string
	BodyMarkdown *string
	Actor        string
}

type MoveWorkItemRequest struct {
	ID      string
	StageID string
	Actor   string
}

type BindWorkItemWorktreeRequest struct {
	ID           string
	Branch       string
	Base         string
	WorktreePath string
	Actor        string
}

type AddWorkItemAttachmentRequest struct {
	WorkItemID string
	Kind       string
	Scope      string
	Path       string
	URL        string
	Note       string
	Actor      string
}

type DeleteWorkItemRequest struct {
	ID    string
	Actor string
}

type StartWorkItemRunRequest struct {
	WorkItemID           string
	Preset               string
	PromptTemplateID     string
	SessionID            string
	PTYID                string
	Launch               bool
	AgentProfileID       string
	SystemPrompt         string
	WorktreeOverridePath string
	Actor                string
}

type LaunchWorkItemRunRequest struct {
	ID                   string
	AgentProfileID       string
	SystemPrompt         string
	WorktreeOverridePath string
	Actor                string
}

type QueueExecutionRequest struct {
	WorkItemID string
	Actor      string
}

type LaunchExecutionRequest struct {
	WorkItemID           string
	AgentProfileID       string
	SystemPrompt         string
	WorktreeOverridePath string
	Actor                string
}

type StartPlanningRequest struct {
	WorkItemID     string
	SessionID      string
	PTYID          string
	Launch         bool
	AgentProfileID string
	SystemPrompt   string
	Actor          string
}

type SubmitDraftPlanRequest struct {
	WorkItemID string
	RunID      string
	Title      string
	Body       string
	Actor      string
}

type ApprovePlanRequest struct {
	ArtifactID string
	WorkItemID string
	Actor      string
}

type StartExecutionRequest struct {
	WorkItemID           string
	SessionID            string
	PTYID                string
	Launch               bool
	AgentProfileID       string
	SystemPrompt         string
	WorktreeOverridePath string
	Actor                string
}

type AskQuestionRequest struct {
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	Prompt     string
	Actor      string
}

type AnswerQuestionRequest struct {
	ID     string
	Answer string
	Actor  string
}

type CompleteExecutionRequest struct {
	RunID   string
	Message string
	Actor   string
}

type SubmitReviewFeedbackRequest struct {
	WorkItemID string
	RunID      string
	Body       string
	Actor      string
}

type ApproveDoneRequest struct {
	WorkItemID string
	Reason     string
	Actor      string
}

type CompleteGateRequest struct {
	ID             string
	Status         string
	OverrideReason string
	Actor          string
}

type CancelWorkItemRunRequest struct {
	ID    string
	Actor string
}

type ReportStatusRequest struct {
	Kind       string
	Message    string
	Actor      string
	ProjectID  string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
}

type ReportStatusResponse struct {
	Event    workitem.StatusEvent
	Run      *workitem.WorkItemRun
	WorkItem *workitem.WorkItem
}

type ListStatusEventsRequest struct {
	ProjectID  string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	UnreadOnly bool
}

type MarkStatusEventReadRequest struct {
	ID string
}

const daemonOwnedSessionMetadataKey = "whisk.daemon/owned_session"

type runCloseMode string

const (
	runCloseNone     runCloseMode = "none"
	runCloseComplete runCloseMode = "complete"
)

func (r *Runtime) ListProjects(context.Context) ([]workitem.Project, error) {
	return r.workItems.ListProjects(), nil
}

func (r *Runtime) GetProjectDetail(_ context.Context, projectID string) (ProjectDetail, error) {
	project, ok := r.workItems.GetProject(projectID)
	if !ok {
		return ProjectDetail{}, fmt.Errorf("project %s not found", projectID)
	}
	sessions := []session.Session{}
	for _, candidate := range r.state.List() {
		if candidate.ProjectID == projectID {
			sessions = append(sessions, candidate)
		}
	}
	runs := []workitem.WorkItemRun{}
	for _, run := range r.workItems.ListRuns("") {
		if run.ProjectID == projectID {
			runs = append(runs, run)
		}
	}
	return ProjectDetail{
		Project:   project,
		WorkItems: r.workItems.ListWorkItems(projectID),
		Sessions:  sessions,
		Runs:      runs,
	}, nil
}

func (r *Runtime) ListWorkflowTemplates(context.Context) ([]workitem.WorkflowTemplate, error) {
	return r.workItems.ListWorkflowTemplates(), nil
}

func (r *Runtime) ListPromptTemplates(context.Context) ([]workitem.PromptTemplate, error) {
	return r.workItems.ListPromptTemplates(), nil
}

func (r *Runtime) CreateProject(ctx context.Context, req CreateProjectRequest) (workitem.Project, error) {
	rootDir, err := validateExistingRootDir(req.RootDir)
	if err != nil {
		return workitem.Project{}, err
	}
	now := time.Now().UTC()
	project, err := r.workItems.CreateProject(workitem.CreateProject{
		ID:                r.ids(),
		ProjectWorkflowID: r.ids(),
		WorkflowID:        req.WorkflowID,
		Name:              req.Name,
		Description:       req.Description,
		Slug:              req.Slug,
		RootDir:           rootDir,
		Preferences:       req.Preferences,
		Now:               now,
	})
	if err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return project, nil
}

func (r *Runtime) UpdateProject(ctx context.Context, req UpdateProjectRequest) (workitem.Project, error) {
	project, err := r.workItems.UpdateProject(workitem.UpdateProject{
		ID:                       req.ID,
		Name:                     req.Name,
		Description:              req.Description,
		Slug:                     req.Slug,
		UseInteractiveAgentShell: req.UseInteractiveAgentShell,
		DefaultPhaseAgents:       req.DefaultPhaseAgents,
		Now:                      time.Now().UTC(),
	})
	if err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return project, nil
}

func (r *Runtime) DeleteProject(ctx context.Context, req DeleteProjectRequest) (workitem.Project, error) {
	project, err := r.workItems.DeleteProject(workitem.DeleteProject{
		ID:    req.ID,
		Actor: req.Actor,
		Now:   time.Now().UTC(),
	})
	if err != nil {
		return workitem.Project{}, err
	}
	for _, current := range r.state.List() {
		if current.ProjectID != req.ID {
			continue
		}
		if _, err := r.state.SetSessionProject(session.SetSessionProject{
			SessionID: current.ID,
			ProjectID: "",
		}); err != nil {
			return workitem.Project{}, err
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	return project, nil
}

func (r *Runtime) AddProjectAttachment(ctx context.Context, req AddProjectAttachmentRequest) (workitem.Project, error) {
	project, err := r.workItems.AddProjectAttachment(workitem.AddProjectAttachment{
		ID:               r.ids(),
		ProjectID:        req.ProjectID,
		Kind:             req.Kind,
		Scope:            req.Scope,
		Title:            req.Title,
		Path:             req.Path,
		URL:              req.URL,
		Note:             req.Note,
		Provider:         req.Provider,
		Target:           req.Target,
		IncludeInContext: req.IncludeInContext,
		Meta:             req.Meta,
		Now:              time.Now().UTC(),
	})
	if err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return project, nil
}

func (r *Runtime) UpdateProjectAttachment(ctx context.Context, req UpdateProjectAttachmentRequest) (workitem.Project, error) {
	project, err := r.workItems.UpdateProjectAttachment(workitem.UpdateProjectAttachment{
		ID:               req.ID,
		ProjectID:        req.ProjectID,
		Title:            req.Title,
		Path:             req.Path,
		URL:              req.URL,
		Note:             req.Note,
		Provider:         req.Provider,
		Target:           req.Target,
		IncludeInContext: req.IncludeInContext,
		Meta:             req.Meta,
		Now:              time.Now().UTC(),
	})
	if err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return project, nil
}

func (r *Runtime) DeleteProjectAttachment(ctx context.Context, req DeleteProjectAttachmentRequest) (workitem.Project, error) {
	project, err := r.workItems.DeleteProjectAttachment(workitem.DeleteProjectAttachment{
		ID:        req.ID,
		ProjectID: req.ProjectID,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return workitem.Project{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Project{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return project, nil
}

func (r *Runtime) ProjectContext(ctx context.Context, projectID string) (ProjectContext, error) {
	project, ok := r.workItems.GetProject(projectID)
	if !ok {
		return ProjectContext{}, fmt.Errorf("project %s not found", projectID)
	}
	out := ProjectContext{ProjectID: project.ID}
	for _, attachment := range project.Attachments {
		if !attachment.IncludeInContext {
			continue
		}
		item := ProjectContextItem{
			AttachmentID: attachment.ID,
			Kind:         attachment.Kind,
			Provider:     attachment.Provider,
			Target:       attachment.Target,
			Title:        attachment.Title,
			Delivery:     "reference",
		}
		switch attachment.Kind {
		case workitem.AttachmentKindFile:
			item.Target = attachment.Path
		case workitem.AttachmentKindURL:
			item.Target = attachment.URL
			item.SourceURL = attachment.URL
		case workitem.AttachmentKindNote:
			item.Delivery = "inline"
			item.ContentType = "text/markdown"
			item.Content = attachment.Note
		case workitem.AttachmentKindExternal:
			resolver := r.projectContextResolver(attachment.Provider)
			if resolver == nil {
				item.Delivery = "skipped"
				item.Error = "no trusted resolver configured"
				break
			}
			resolved, err := resolver.ResolveProjectAttachment(ctx, ResolveProjectAttachmentRequest{
				ProjectID:    project.ID,
				AttachmentID: attachment.ID,
				Provider:     attachment.Provider,
				Target:       attachment.Target,
				BudgetBytes:  64 * 1024,
			})
			if err != nil {
				item.Delivery = "skipped"
				item.Error = err.Error()
				break
			}
			if resolved.Title != "" {
				item.Title = resolved.Title
			}
			item.Delivery = resolved.Delivery
			item.ContentType = resolved.ContentType
			item.Content = resolved.Content
			item.SourceURL = resolved.SourceURL
			item.Error = resolved.Error
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func (r *Runtime) ListWorkItems(_ context.Context, projectID string) ([]workitem.WorkItem, error) {
	return r.workItems.ListWorkItems(projectID), nil
}

func (r *Runtime) CreateWorkItem(ctx context.Context, req CreateWorkItemRequest) (workitem.WorkItem, error) {
	now := time.Now().UTC()
	item, err := r.workItems.CreateWorkItem(workitem.CreateWorkItem{
		ID:           r.ids(),
		HistoryID:    r.ids(),
		ProjectID:    req.ProjectID,
		WorkflowID:   req.WorkflowID,
		Title:        req.Title,
		BodyMarkdown: req.BodyMarkdown,
		StageID:      req.StageID,
		Actor:        req.Actor,
		Now:          now,
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) UpdateWorkItem(ctx context.Context, req UpdateWorkItemRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.UpdateWorkItem(workitem.UpdateWorkItem{
		ID:           req.ID,
		HistoryID:    r.ids(),
		Title:        req.Title,
		BodyMarkdown: req.BodyMarkdown,
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) MoveWorkItem(ctx context.Context, req MoveWorkItemRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.MoveWorkItem(workitem.MoveWorkItem{
		ID:        req.ID,
		HistoryID: r.ids(),
		StageID:   req.StageID,
		Actor:     req.Actor,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) BindWorkItemWorktree(ctx context.Context, req BindWorkItemWorktreeRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.BindWorktree(workitem.BindWorktree{
		ID:           req.ID,
		HistoryID:    r.ids(),
		Branch:       req.Branch,
		Base:         req.Base,
		WorktreePath: req.WorktreePath,
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) AddWorkItemAttachment(ctx context.Context, req AddWorkItemAttachmentRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.AddAttachment(workitem.AddAttachment{
		ID:         r.ids(),
		HistoryID:  r.ids(),
		WorkItemID: req.WorkItemID,
		Kind:       req.Kind,
		Scope:      req.Scope,
		Path:       req.Path,
		URL:        req.URL,
		Note:       req.Note,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) DeleteWorkItem(ctx context.Context, req DeleteWorkItemRequest) (workitem.WorkItem, error) {
	runs := r.workItems.ListRuns(req.ID)
	closedSessions := map[string]struct{}{}
	for _, run := range runs {
		if run.SessionID != "" {
			if _, seen := closedSessions[run.SessionID]; seen {
				continue
			}
			closedSessions[run.SessionID] = struct{}{}
			if _, ok := r.state.Get(run.SessionID); ok {
				if _, err := r.CloseSession(ctx, CloseSessionRequest{SessionID: run.SessionID}); err != nil {
					return workitem.WorkItem{}, err
				}
			}
			continue
		}
		if run.PTYID != "" && r.ptys != nil {
			if _, err := r.KillPTY(ctx, KillPTYRequest{PTYID: run.PTYID}); err != nil {
				return workitem.WorkItem{}, err
			}
		}
	}
	item, err := r.workItems.DeleteWorkItem(workitem.DeleteWorkItem{
		ID:        req.ID,
		HistoryID: r.ids(),
		Actor:     req.Actor,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) ListWorkItemRuns(_ context.Context, workItemID string) ([]workitem.WorkItemRun, error) {
	return r.workItems.ListRuns(workItemID), nil
}

func (r *Runtime) StartWorkItemRun(ctx context.Context, req StartWorkItemRunRequest) (workitem.WorkItemRun, error) {
	now := time.Now().UTC()
	run, err := r.workItems.StartRun(workitem.StartRun{
		ID:               r.ids(),
		HistoryID:        r.ids(),
		RunHistoryID:     r.ids(),
		WorkItemID:       req.WorkItemID,
		Preset:           req.Preset,
		PromptTemplateID: req.PromptTemplateID,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		Actor:            req.Actor,
		Now:              now,
	})
	if err != nil {
		return workitem.WorkItemRun{}, err
	}
	if req.Launch {
		sessionID, ptyID, err := r.launchWorkItemRun(ctx, run, req)
		if err != nil {
			failed, failErr := r.workItems.FailRun(workitem.FailRun{
				ID:           run.ID,
				RunHistoryID: r.ids(),
				Actor:        req.Actor,
				Message:      err.Error(),
				Now:          time.Now().UTC(),
			})
			if failErr == nil {
				_ = r.persistWorkItems(ctx)
				r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
			}
			if failErr != nil {
				return workitem.WorkItemRun{}, failErr
			}
			return failed, err
		}
		run, err = r.workItems.MarkRunRunning(workitem.MarkRunRunning{
			ID:           run.ID,
			RunHistoryID: r.ids(),
			SessionID:    sessionID,
			PTYID:        ptyID,
			DaemonOwned:  true,
			Actor:        req.Actor,
			Now:          time.Now().UTC(),
		})
		if err != nil {
			return workitem.WorkItemRun{}, err
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItemRun{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return run, nil
}

func (r *Runtime) LaunchWorkItemRun(ctx context.Context, req LaunchWorkItemRunRequest) (workitem.WorkItemRun, error) {
	run, ok := r.workItems.GetRun(req.ID)
	if !ok {
		return workitem.WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status != workitem.RunStateQueued {
		return workitem.WorkItemRun{}, fmt.Errorf("work item run %s is %s, not queued", req.ID, run.Status)
	}
	launched, err := r.launchAndMarkWorkItemRun(ctx, run, StartWorkItemRunRequest{
		WorkItemID:           run.WorkItemID,
		Preset:               run.Preset,
		PromptTemplateID:     run.PromptTemplateID,
		AgentProfileID:       req.AgentProfileID,
		SystemPrompt:         req.SystemPrompt,
		WorktreeOverridePath: req.WorktreeOverridePath,
		Actor:                req.Actor,
	})
	if err != nil {
		failed, failErr := r.workItems.FailRun(workitem.FailRun{
			ID:           run.ID,
			RunHistoryID: r.ids(),
			Actor:        req.Actor,
			Message:      err.Error(),
			Now:          time.Now().UTC(),
		})
		if failErr == nil {
			_ = r.persistWorkItems(ctx)
			r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
		}
		if failErr != nil {
			return workitem.WorkItemRun{}, failErr
		}
		return failed, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItemRun{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return launched, nil
}

func (r *Runtime) StartPlanning(ctx context.Context, req StartPlanningRequest) (workitem.WorkItemRun, error) {
	now := time.Now().UTC()
	run, err := r.workItems.StartPlanning(workitem.StartPlanning{
		ID:           r.ids(),
		HistoryID:    r.ids(),
		RunHistoryID: r.ids(),
		WorkItemID:   req.WorkItemID,
		SessionID:    req.SessionID,
		PTYID:        req.PTYID,
		Actor:        req.Actor,
		Now:          now,
	})
	if err != nil {
		return workitem.WorkItemRun{}, err
	}
	if req.Launch {
		run, err = r.launchAndMarkWorkItemRun(ctx, run, StartWorkItemRunRequest{
			WorkItemID:     req.WorkItemID,
			Launch:         true,
			AgentProfileID: req.AgentProfileID,
			SystemPrompt:   req.SystemPrompt,
			Actor:          req.Actor,
		})
		if err != nil {
			return workitem.WorkItemRun{}, err
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItemRun{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return run, nil
}

func (r *Runtime) SubmitDraftPlan(ctx context.Context, req SubmitDraftPlanRequest) (workitem.Artifact, error) {
	artifact, err := r.workItems.SubmitDraftPlan(workitem.SubmitDraftPlan{
		ID:         r.ids(),
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		Title:      req.Title,
		Body:       req.Body,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.Artifact{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Artifact{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return artifact, nil
}

func (r *Runtime) ApprovePlan(ctx context.Context, req ApprovePlanRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.ApprovePlan(workitem.ApprovePlan{
		ArtifactID: req.ArtifactID,
		WorkItemID: req.WorkItemID,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.closeDaemonOwnedRunSessions(ctx, item.ID, req.Actor, runCloseComplete, func(run workitem.WorkItemRun) bool {
		return run.PromptTemplateID == workitem.PromptTemplatePlan
	}); err != nil {
		return workitem.WorkItem{}, err
	}
	if refreshed, ok := r.workItems.GetWorkItem(item.ID); ok {
		item = refreshed
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) StartExecution(ctx context.Context, req StartExecutionRequest) (workitem.WorkItemRun, error) {
	return r.startExecution(ctx, req)
}

func (r *Runtime) QueueExecution(ctx context.Context, req QueueExecutionRequest) (workitem.WorkItemRun, error) {
	return r.startExecution(ctx, StartExecutionRequest{
		WorkItemID: req.WorkItemID,
		Actor:      req.Actor,
	})
}

func (r *Runtime) LaunchExecution(ctx context.Context, req LaunchExecutionRequest) (workitem.WorkItemRun, error) {
	return r.startExecution(ctx, StartExecutionRequest{
		WorkItemID:           req.WorkItemID,
		Launch:               true,
		AgentProfileID:       req.AgentProfileID,
		SystemPrompt:         req.SystemPrompt,
		WorktreeOverridePath: req.WorktreeOverridePath,
		Actor:                req.Actor,
	})
}

func (r *Runtime) startExecution(ctx context.Context, req StartExecutionRequest) (workitem.WorkItemRun, error) {
	run, err := r.workItems.StartExecution(workitem.StartExecution{
		ID:           r.ids(),
		HistoryID:    r.ids(),
		RunHistoryID: r.ids(),
		WorkItemID:   req.WorkItemID,
		SessionID:    req.SessionID,
		PTYID:        req.PTYID,
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItemRun{}, err
	}
	if req.Launch {
		run, err = r.launchAndMarkWorkItemRun(ctx, run, StartWorkItemRunRequest{
			WorkItemID:           req.WorkItemID,
			Launch:               true,
			AgentProfileID:       req.AgentProfileID,
			SystemPrompt:         req.SystemPrompt,
			WorktreeOverridePath: req.WorktreeOverridePath,
			Actor:                req.Actor,
		})
		if err != nil {
			return workitem.WorkItemRun{}, err
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItemRun{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return run, nil
}

func (r *Runtime) AskQuestion(ctx context.Context, req AskQuestionRequest) (workitem.Question, error) {
	question, err := r.workItems.AskQuestion(workitem.AskQuestion{
		ID:         r.ids(),
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		Prompt:     req.Prompt,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.Question{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Question{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return question, nil
}

func (r *Runtime) AnswerQuestion(ctx context.Context, req AnswerQuestionRequest) (workitem.Question, error) {
	question, err := r.workItems.AnswerQuestion(workitem.AnswerQuestion{
		ID:     req.ID,
		Answer: req.Answer,
		Actor:  req.Actor,
		Now:    time.Now().UTC(),
	})
	if err != nil {
		return workitem.Question{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Question{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return question, nil
}

func (r *Runtime) CompleteExecution(ctx context.Context, req CompleteExecutionRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.CompleteExecution(workitem.CompleteExecution{
		RunID:   req.RunID,
		Actor:   req.Actor,
		Message: req.Message,
		Now:     time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.closeDaemonOwnedRunSessions(ctx, item.ID, req.Actor, runCloseNone, func(run workitem.WorkItemRun) bool {
		return run.ID == req.RunID
	}); err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) SubmitReviewFeedback(ctx context.Context, req SubmitReviewFeedbackRequest) (workitem.Artifact, error) {
	artifact, err := r.workItems.SubmitReviewFeedback(workitem.SubmitReviewFeedback{
		ID:         r.ids(),
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		Body:       req.Body,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.Artifact{}, err
	}
	if r.ptys != nil {
		for _, run := range r.workItems.ListRuns(req.WorkItemID) {
			if run.ID != req.RunID || run.PTYID == "" {
				continue
			}
			snapshot, err := r.PTYOutput(ctx, run.PTYID, 0)
			if err != nil || !snapshot.Record.Running {
				break
			}
			envelope := "<whisk-review-feedback>\n" + req.Body + "\n</whisk-review-feedback>"
			r.submitAgentMessage(run.PTYID, envelope)
			break
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Artifact{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return artifact, nil
}

// submitAgentMessage delivers a message into an already-running interactive agent
// TUI and submits it. Argv can only seed the first turn (see agents.BuildLaunch),
// so mid-session injection has to go through the terminal. It runs asynchronously:
// clear any stale draft, paste the text inside bracketed-paste markers so
// multi-line content stays editable data instead of submitting on each newline,
// wait for the TUI to settle the paste, then send Enter as its own keystroke. An
// Enter carried in the same burst as the paste gets folded into the draft and the
// message sits unsent, so it must arrive separately and after the paste lands.
func (r *Runtime) submitAgentMessage(ptyID, text string) {
	go func() {
		ctx := r.watchCtx
		// Clear a stale draft first: Ctrl-A (home) + Ctrl-K (kill to end).
		if err := r.WritePTY(ctx, ptyID, []byte{0x01, 0x0b}); err != nil {
			return
		}
		baseLen := 0
		if snapshot, err := r.PTYOutput(ctx, ptyID, 0); err == nil {
			baseLen = len(snapshot.OutputBytes)
		}
		if err := r.WritePTY(ctx, ptyID, bracketedPaste(text)); err != nil {
			return
		}
		r.waitPTYOutputSettled(ctx, ptyID, baseLen)
		_ = r.WritePTY(ctx, ptyID, []byte("\r"))
	}()
}

// bracketedPaste wraps text in bracketed-paste markers, mapping interior newlines
// to the carriage return a real paste carries between lines (so the TUI keeps them
// as data) and appending a trailing CR that absorbs any trailing backslash so it
// can't escape the submit Enter.
func bracketedPaste(text string) []byte {
	body := strings.ReplaceAll(text, "\r\n", "\r")
	body = strings.ReplaceAll(body, "\n", "\r")
	return []byte("\x1b[200~" + body + "\r\x1b[201~")
}

// waitPTYOutputSettled blocks until the PTY output has grown past baseLen (the
// paste echoed) and then stayed quiet for a short window, or a max timeout
// elapses. This commits the paste before the follow-up Enter — a fixed sleep
// races it under load or with large payloads.
func (r *Runtime) waitPTYOutputSettled(ctx context.Context, ptyID string, baseLen int) {
	const (
		quietWindow = 150 * time.Millisecond
		maxWait     = 3 * time.Second
		pollEvery   = 50 * time.Millisecond
	)
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	lastLen := -1
	var stableSince time.Time
	for {
		if snapshot, err := r.PTYOutput(ctx, ptyID, 0); err == nil {
			n := len(snapshot.OutputBytes)
			switch {
			case n != lastLen:
				lastLen = n
				stableSince = time.Now()
			case n > baseLen && !stableSince.IsZero() && time.Since(stableSince) >= quietWindow:
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
		}
	}
}

func (r *Runtime) ListArtifacts(_ context.Context, workItemID string) ([]workitem.Artifact, error) {
	return r.workItems.ListArtifacts(workItemID), nil
}

func (r *Runtime) ListQuestions(_ context.Context, workItemID string) ([]workitem.Question, error) {
	return r.workItems.ListQuestions(workItemID), nil
}

func (r *Runtime) ListGateReports(_ context.Context, workItemID string) ([]workitem.GateReport, error) {
	return r.workItems.ListGateReports(workItemID), nil
}

func (r *Runtime) ListWorkflowEvents(_ context.Context, workItemID string) ([]workitem.WorkflowEvent, error) {
	return r.workItems.ListWorkflowEvents(workItemID), nil
}

func (r *Runtime) CompleteGate(ctx context.Context, req CompleteGateRequest) (workitem.GateReport, error) {
	gate, err := r.workItems.CompleteGate(workitem.CompleteGate{
		ID:             req.ID,
		Status:         req.Status,
		OverrideReason: req.OverrideReason,
		Actor:          req.Actor,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		return workitem.GateReport{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.GateReport{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return gate, nil
}

func (r *Runtime) ApproveDone(ctx context.Context, req ApproveDoneRequest) (workitem.WorkItem, error) {
	item, err := r.workItems.ApproveDone(workitem.ApproveDone{
		WorkItemID: req.WorkItemID,
		Reason:     req.Reason,
		Actor:      req.Actor,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if err := r.closeDaemonOwnedRunSessions(ctx, item.ID, req.Actor, runCloseComplete, func(workitem.WorkItemRun) bool {
		return true
	}); err != nil {
		return workitem.WorkItem{}, err
	}
	if refreshed, ok := r.workItems.GetWorkItem(item.ID); ok {
		item = refreshed
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) launchAndMarkWorkItemRun(ctx context.Context, run workitem.WorkItemRun, req StartWorkItemRunRequest) (workitem.WorkItemRun, error) {
	sessionID, ptyID, err := r.launchWorkItemRun(ctx, run, req)
	if err != nil {
		return workitem.WorkItemRun{}, err
	}
	return r.workItems.MarkRunRunning(workitem.MarkRunRunning{
		ID:           run.ID,
		RunHistoryID: r.ids(),
		SessionID:    sessionID,
		PTYID:        ptyID,
		DaemonOwned:  true,
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
}

func (r *Runtime) closeDaemonOwnedRunSessions(ctx context.Context, workItemID string, actor string, mode runCloseMode, match func(workitem.WorkItemRun) bool) error {
	closedSessions := map[string]struct{}{}
	for _, run := range r.workItems.ListRuns(workItemID) {
		if !match(run) || !daemonOwnedRunSession(run) {
			continue
		}
		if !terminalRunStatus(run.Status) {
			switch mode {
			case runCloseComplete:
				if _, err := r.workItems.CompleteRun(workitem.CompleteRun{
					ID:           run.ID,
					RunHistoryID: r.ids(),
					Actor:        actor,
					Message:      "workflow phase completed",
					Now:          time.Now().UTC(),
				}); err != nil {
					return err
				}
			case runCloseNone:
			default:
				return fmt.Errorf("unknown run close mode %q", mode)
			}
		}
		if run.SessionID == "" {
			continue
		}
		if _, seen := closedSessions[run.SessionID]; seen {
			continue
		}
		closedSessions[run.SessionID] = struct{}{}
		if _, ok := r.state.Get(run.SessionID); !ok {
			continue
		}
		if _, err := r.CloseSession(ctx, CloseSessionRequest{SessionID: run.SessionID}); err != nil {
			return err
		}
	}
	return nil
}

func daemonOwnedRunSession(run workitem.WorkItemRun) bool {
	value, ok := run.Metadata[daemonOwnedSessionMetadataKey]
	return ok && value.Type == workitem.MetadataTypeBool && value.Bool
}

func terminalRunStatus(status string) bool {
	return status == workitem.RunStateCompleted || status == workitem.RunStateFailed || status == workitem.RunStateCancelled
}

func (r *Runtime) launchWorkItemRun(ctx context.Context, run workitem.WorkItemRun, req StartWorkItemRunRequest) (string, string, error) {
	if r.ptys == nil {
		return "", "", fmt.Errorf("pty backend required")
	}
	item, ok := r.workItems.GetWorkItem(run.WorkItemID)
	if !ok {
		return "", "", fmt.Errorf("work item %s not found", run.WorkItemID)
	}
	project, ok := r.workItems.GetProject(item.ProjectID)
	if !ok {
		return "", "", fmt.Errorf("project %s not found", item.ProjectID)
	}
	var err error
	item, err = r.ensureExecutionWorktree(ctx, project, item, run, req.WorktreeOverridePath, req.Actor)
	if err != nil {
		return "", "", err
	}
	workingDir := project.RootDir
	if item.Worktree != nil && item.Worktree.WorktreePath != "" {
		workingDir = item.Worktree.WorktreePath
	}
	profileID := resolveAgentProfileID(req.AgentProfileID, project.Preferences.DefaultPhaseAgents, run.Preset)
	launch, err := agents.BuildLaunch(agents.LaunchRequest{
		ProfileID:    profileID,
		WorkingDir:   workingDir,
		SystemPrompt: req.SystemPrompt,
		Prompt:       run.PromptSnapshot,
	})
	if err != nil {
		return "", "", err
	}
	bridgeLaunch, err := prepareAgentBridgeLaunch(r, launch.Provider, run.ID, launch.WorkingDir)
	if err != nil {
		return "", "", err
	}
	agentEnv := map[string]string{}
	for key, value := range launch.Env {
		agentEnv[key] = value
	}
	for key, value := range map[string]string{
		"WHISK_PROJECT_ID":   project.ID,
		"WHISK_WORK_ITEM_ID": item.ID,
		"WHISK_RUN_ID":       run.ID,
		"WHISK_ACTOR":        "agent",
	} {
		agentEnv[key] = value
	}
	if bridgeLaunch != nil {
		for key, value := range bridgeLaunch.Env {
			agentEnv[key] = value
		}
	}
	name := workItemRunSessionName(item, run)
	created, err := r.CreateSession(ctx, CreateSessionRequest{
		Name:       name,
		RootDir:    launch.WorkingDir,
		ProjectID:  project.ID,
		InitialPTY: agentStartPTYOptions(launch, agentEnv, project.Preferences.UseInteractiveAgentShell),
	})
	if err != nil {
		return "", "", err
	}
	if created.MainPtyID == "" {
		return "", "", fmt.Errorf("launched session without pty")
	}
	if bridgeLaunch != nil {
		bridge := bridgeLaunch.Bridge
		bridge.SessionID = created.Session.ID
		bridge.PTYID = created.MainPtyID
		if err := r.registerAgentBridge(bridge); err != nil {
			return "", "", err
		}
	}
	// Agent prompts (Claude, Codex) ride in argv so the agent auto-runs the first
	// turn — see agents.BuildLaunch. Only non-agent shell providers still carry a
	// prompt in Stdin to type in.
	if strings.TrimSpace(launch.Stdin) != "" {
		if err := r.WritePTY(ctx, created.MainPtyID, []byte(launch.Stdin+"\n")); err != nil {
			return "", "", err
		}
	}
	return created.Session.ID, created.MainPtyID, nil
}

func (r *Runtime) ensureExecutionWorktree(ctx context.Context, project workitem.Project, item workitem.WorkItem, run workitem.WorkItemRun, overridePath string, actor string) (workitem.WorkItem, error) {
	if item.Worktree != nil || run.PromptTemplateID != workitem.PromptTemplateImplement || r.worktrees == nil {
		return item, nil
	}
	branch := defaultWorktreeBranch(project, item)
	created, err := r.worktrees.CreateWorktree(ctx, CreateWorktreeRequest{
		RepoPath:     project.RootDir,
		Branch:       branch,
		OverridePath: overridePath,
	})
	if err != nil {
		return workitem.WorkItem{}, err
	}
	if strings.TrimSpace(created.Path) == "" {
		return workitem.WorkItem{}, fmt.Errorf("created worktree path required")
	}
	if strings.TrimSpace(actor) == "" {
		actor = "agent"
	}
	return r.workItems.BindWorktree(workitem.BindWorktree{
		ID:           item.ID,
		HistoryID:    r.ids(),
		Branch:       branch,
		WorktreePath: created.Path,
		Actor:        actor,
		Now:          time.Now().UTC(),
	})
}

func agentStartPTYOptions(launch agents.Launch, env map[string]string, useInteractiveShell bool) *StartPTYOptions {
	if useInteractiveShell {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "sh"
		}
		return &StartPTYOptions{Command: shell, Args: []string{"-lc", agents.CommandLine(launch.Command, launch.Args)}, Exec: true, Env: env}
	}
	return &StartPTYOptions{Command: launch.Command, Args: launch.Args, Exec: true, Env: env}
}

type agentBridgeLaunch struct {
	Env    map[string]string
	Bridge agentbridge.Bridge
}

func prepareAgentBridgeLaunch(r *Runtime, provider agents.Provider, runID string, workingDir string) (*agentBridgeLaunch, error) {
	bridgeProvider, ok := agentBridgeProvider(provider)
	if !ok {
		return nil, nil
	}
	return prepareAgentBridgeLaunchForProvider(r, bridgeProvider, runID, workingDir)
}

func prepareAgentBridgeLaunchForProvider(r *Runtime, bridgeProvider agentbridge.Provider, runID string, workingDir string) (*agentBridgeLaunch, error) {
	if strings.TrimSpace(r.daemonURL) == "" {
		return nil, nil
	}
	bridgeID := r.ids()
	token := r.ids()
	hookURL := agentBridgeHookURL(r.daemonURL, bridgeID)
	installed, err := bridgeinstaller.Install(bridgeinstaller.InstallRequest{
		RootDir:   workingDir,
		BridgeID:  bridgeID,
		RunID:     runID,
		Provider:  string(bridgeProvider),
		HookURL:   hookURL,
		Token:     token,
		WhiskCLI:  r.cliPath,
		WhiskdURL: r.daemonURL,
	})
	if err != nil {
		return nil, err
	}
	return &agentBridgeLaunch{
		Env: map[string]string{
			"WHISK_AGENT_BRIDGE_ID":          bridgeID,
			"WHISK_AGENT_BRIDGE_TOKEN":       token,
			"WHISK_AGENT_BRIDGE_PROVIDER":    string(bridgeProvider),
			"WHISK_AGENT_BRIDGE_HOOK_URL":    hookURL,
			"WHISK_AGENT_BRIDGE_CONFIG_DIR":  installed.Dir,
			"WHISK_AGENT_BRIDGE_HOOK_SCRIPT": installed.HookScript,
		},
		Bridge: agentbridge.Bridge{
			ID:        bridgeID,
			Provider:  bridgeProvider,
			TokenHash: agentbridge.HashHookToken(token),
		},
	}, nil
}

func agentBridgeProviderFromString(provider string) (agentbridge.Provider, bool) {
	switch strings.TrimSpace(provider) {
	case string(agentbridge.ProviderClaude):
		return agentbridge.ProviderClaude, true
	case string(agentbridge.ProviderCodex):
		return agentbridge.ProviderCodex, true
	default:
		return "", false
	}
}

func agentBridgeProvider(provider agents.Provider) (agentbridge.Provider, bool) {
	switch provider {
	case agents.ProviderClaude:
		return agentbridge.ProviderClaude, true
	case agents.ProviderCodex:
		return agentbridge.ProviderCodex, true
	default:
		return "", false
	}
}

func agentBridgeHookURL(baseURL string, bridgeID string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/v1/agent-bridges/" + bridgeID + "/hooks"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1/agent-bridges/" + bridgeID + "/hooks"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func workItemRunSessionName(item workitem.WorkItem, run workitem.WorkItemRun) string {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = item.ID
	}
	return fmt.Sprintf("#%d %s - %s", item.Number, runPhaseLabel(run), title)
}

func runPhaseLabel(run workitem.WorkItemRun) string {
	switch run.PromptTemplateID {
	case workitem.PromptTemplatePlan:
		return "Planning"
	case workitem.PromptTemplateImplement:
		return "Execution"
	case workitem.PromptTemplateReview:
		return "Review"
	default:
		return "Run"
	}
}

func defaultAgentProfileForPreset(preset string) string {
	switch preset {
	case workitem.RunPresetReader:
		return "claude-plan"
	case workitem.RunPresetManager, workitem.RunPresetReviewer, workitem.RunPresetWriter:
		return "claude"
	default:
		return ""
	}
}

func defaultWorktreeBranch(project workitem.Project, item workitem.WorkItem) string {
	projectSlug := strings.TrimSpace(project.Slug)
	if projectSlug == "" {
		projectSlug = "work"
	}
	itemSlug := worktreeBranchSlug(item.Title)
	if itemSlug == "" {
		itemSlug = "item"
	}
	return fmt.Sprintf("whisk/%s-%d-%s", projectSlug, item.Number, itemSlug)
}

func worktreeBranchSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

// resolveAgentProfileID chooses the agent profile to launch with. An explicit
// request wins; otherwise the project's per-phase default (keyed by run preset)
// is used; otherwise it falls back to the preset's builtin default.
func resolveAgentProfileID(explicit string, phaseAgents map[string]string, preset string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	if phaseAgents != nil {
		if id := strings.TrimSpace(phaseAgents[preset]); id != "" {
			return id
		}
	}
	return defaultAgentProfileForPreset(preset)
}

// ListAgentProfiles returns the selectable builtin agent profiles as a read model.
func (r *Runtime) ListAgentProfiles(context.Context) ([]agents.ProfileInfo, error) {
	return agents.ProfileList(), nil
}

func (r *Runtime) CancelWorkItemRun(ctx context.Context, req CancelWorkItemRunRequest) (workitem.WorkItemRun, error) {
	run, err := r.workItems.CancelRun(workitem.CancelRun{
		ID:           req.ID,
		RunHistoryID: r.ids(),
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return workitem.WorkItemRun{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItemRun{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return run, nil
}

func (r *Runtime) ReportStatus(ctx context.Context, req ReportStatusRequest) (ReportStatusResponse, error) {
	event, err := r.workItems.ReportStatus(workitem.ReportStatus{
		ID:           r.ids(),
		RunHistoryID: r.ids(),
		Kind:         req.Kind,
		Message:      req.Message,
		Actor:        req.Actor,
		ProjectID:    req.ProjectID,
		WorkItemID:   req.WorkItemID,
		RunID:        req.RunID,
		SessionID:    req.SessionID,
		PTYID:        req.PTYID,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return ReportStatusResponse{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return ReportStatusResponse{}, err
	}
	if event.RunID != "" {
		r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	}
	r.publish(ctx, RuntimeEvent{Type: EventStatusChanged})
	response := ReportStatusResponse{Event: event}
	if event.WorkItemID != "" {
		if item, ok := r.workItems.GetWorkItem(event.WorkItemID); ok {
			response.WorkItem = &item
		}
	}
	if event.RunID != "" {
		for _, run := range r.workItems.ListRuns(event.WorkItemID) {
			if run.ID == event.RunID {
				response.Run = &run
				break
			}
		}
	}
	return response, nil
}

func (r *Runtime) ListStatusEvents(_ context.Context, req ListStatusEventsRequest) ([]workitem.StatusEvent, error) {
	return r.workItems.ListStatusEvents(workitem.ListStatusEvents{
		ProjectID:  req.ProjectID,
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		UnreadOnly: req.UnreadOnly,
	}), nil
}

func (r *Runtime) MarkStatusEventRead(ctx context.Context, req MarkStatusEventReadRequest) (workitem.StatusEvent, error) {
	event, err := r.workItems.MarkStatusEventRead(workitem.MarkStatusEventRead{
		ID:  req.ID,
		Now: time.Now().UTC(),
	})
	if err != nil {
		return workitem.StatusEvent{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.StatusEvent{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventStatusChanged})
	return event, nil
}
