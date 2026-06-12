package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type CreateProjectRequest struct {
	Name        string
	Slug        string
	RootDir     string
	WorkflowID  string
	Preferences workitem.ProjectPreferences
}

type CreateWorkItemRequest struct {
	ProjectID    string
	WorkflowID   string
	Title        string
	BodyMarkdown string
	StageID      string
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
	WorkItemID       string
	Preset           string
	PromptTemplateID string
	SessionID        string
	PTYID            string
	Launch           bool
	AgentProfileID   string
	SystemPrompt     string
	Actor            string
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
	WorkItemID     string
	SessionID      string
	PTYID          string
	Launch         bool
	AgentProfileID string
	SystemPrompt   string
	Actor          string
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

func (r *Runtime) ListProjects(context.Context) ([]workitem.Project, error) {
	return r.workItems.ListProjects(), nil
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
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.WorkItem{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return item, nil
}

func (r *Runtime) StartExecution(ctx context.Context, req StartExecutionRequest) (workitem.WorkItemRun, error) {
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
	for _, run := range r.workItems.ListRuns(req.WorkItemID) {
		if run.ID == req.RunID && run.PTYID != "" {
			envelope := "\n<whisk-review-feedback>\n" + req.Body + "\n</whisk-review-feedback>\n"
			if err := r.WritePTY(ctx, run.PTYID, []byte(envelope)); err != nil {
				return workitem.Artifact{}, err
			}
			break
		}
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return workitem.Artifact{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return artifact, nil
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
		Actor:        req.Actor,
		Now:          time.Now().UTC(),
	})
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
	workingDir := project.RootDir
	if item.Worktree != nil && item.Worktree.WorktreePath != "" {
		workingDir = item.Worktree.WorktreePath
	}
	profileID := strings.TrimSpace(req.AgentProfileID)
	if profileID == "" {
		profileID = defaultAgentProfileForPreset(run.Preset)
	}
	launch, err := agents.BuildLaunch(agents.LaunchRequest{
		ProfileID:    profileID,
		WorkingDir:   workingDir,
		SystemPrompt: req.SystemPrompt,
		Prompt:       run.PromptSnapshot,
	})
	if err != nil {
		return "", "", err
	}
	command := shellCommand(launch.Command, launch.Args, launch.Env)
	name := fmt.Sprintf("Run #%d %s", item.Number, item.Title)
	created, err := r.CreateSession(ctx, CreateSessionRequest{
		Name:    name,
		RootDir: launch.WorkingDir,
		InitialPTY: &StartPTYOptions{
			Command: command,
			Env: map[string]string{
				"WHISK_PROJECT_ID":   project.ID,
				"WHISK_WORK_ITEM_ID": item.ID,
				"WHISK_RUN_ID":       run.ID,
				"WHISK_ACTOR":        "agent",
			},
		},
	})
	if err != nil {
		return "", "", err
	}
	if created.MainPtyID == "" {
		return "", "", fmt.Errorf("launched session without pty")
	}
	if strings.TrimSpace(launch.Stdin) != "" {
		if err := r.WritePTY(ctx, created.MainPtyID, []byte(launch.Stdin+"\n")); err != nil {
			return "", "", err
		}
	}
	return created.Session.ID, created.MainPtyID, nil
}

func defaultAgentProfileForPreset(preset string) string {
	switch preset {
	case workitem.RunPresetReader, workitem.RunPresetManager, workitem.RunPresetReviewer, workitem.RunPresetWriter:
		return "codex"
	default:
		return "codex"
	}
}

func shellCommand(command string, args []string, env map[string]string) string {
	parts := make([]string, 0, len(env)+1+len(args))
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := env[key]
		parts = append(parts, key+"="+shellQuote(value))
	}
	parts = append(parts, shellQuote(command))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
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
