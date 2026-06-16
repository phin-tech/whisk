package wailsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

// daemonControlTimeout bounds start/stop/restart operations so a frontend action can't hang the
// UI waiting on a wedged daemon.
const daemonControlTimeout = 12 * time.Second

type AppSettingsStore interface {
	Load(context.Context) (appsettings.Settings, error)
	Save(context.Context, appsettings.Settings) (appsettings.Settings, error)
}

type Service struct {
	client    client.RuntimeClient
	forwarder *client.LocalForwarder
	settings  AppSettingsStore
}

func NewService(runtimeClient client.RuntimeClient) *Service {
	return NewServiceWithSettings(runtimeClient, nil)
}

func NewServiceWithSettings(runtimeClient client.RuntimeClient, settings AppSettingsStore) *Service {
	service := &Service{client: runtimeClient, settings: settings}
	if httpClient, ok := runtimeClient.(*client.HTTPClient); ok {
		service.forwarder = client.NewLocalForwarder(httpClient, nil)
	}
	return service
}

func (s *Service) LoadAppSettings(ctx context.Context) (appsettings.Settings, error) {
	if s.settings == nil {
		return appsettings.Default(), nil
	}
	return s.settings.Load(ctx)
}

func (s *Service) SaveAppSettings(ctx context.Context, settings appsettings.Settings) (appsettings.Settings, error) {
	if s.settings == nil {
		return appsettings.Normalize(settings)
	}
	return s.settings.Save(ctx, settings)
}

// DaemonStatus describes the daemon the app talks to, for the daemon preferences panel.
type DaemonStatus struct {
	// Running is true when the daemon answers health checks.
	Running bool `json:"running"`
	// Address is the daemon URL (e.g. http://127.0.0.1:8787).
	Address string `json:"address"`
	// Managed is true when this app started the daemon (a live PID file names it), as opposed to
	// one started independently (e.g. `whisk daemon run`).
	Managed bool `json:"managed"`
	// APIVersion and GitSHA come from the daemon's compatibility endpoint when it is reachable.
	APIVersion int    `json:"apiVersion"`
	GitSHA     string `json:"gitSha"`
	// Error holds a human-readable reason when the daemon is unreachable or incompatible.
	Error string `json:"error"`
}

func (s *Service) httpClient() (*client.HTTPClient, error) {
	httpClient, ok := s.client.(*client.HTTPClient)
	if !ok {
		return nil, fmt.Errorf("daemon control requires an HTTP daemon client")
	}
	return httpClient, nil
}

func (s *Service) daemonStatus(ctx context.Context, httpClient *client.HTTPClient) DaemonStatus {
	baseURL := httpClient.BaseURL()
	status := DaemonStatus{Address: baseURL, Managed: daemon.IsManaged(baseURL)}
	if err := httpClient.Health(ctx); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Running = true
	compat, err := httpClient.Compatibility(ctx)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.APIVersion = compat.APIVersion
	status.GitSHA = compat.GitSHA
	return status
}

// DaemonStatus reports the current state of the daemon for display in the preferences panel.
func (s *Service) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	httpClient, err := s.httpClient()
	if err != nil {
		return DaemonStatus{}, err
	}
	return s.daemonStatus(ctx, httpClient), nil
}

// StartDaemon starts a daemon if one is not already running, then returns the resulting status.
func (s *Service) StartDaemon(ctx context.Context) (DaemonStatus, error) {
	httpClient, err := s.httpClient()
	if err != nil {
		return DaemonStatus{}, err
	}
	opCtx, cancel := context.WithTimeout(ctx, daemonControlTimeout)
	defer cancel()
	if _, err := daemon.Ensure(opCtx, httpClient.BaseURL()); err != nil {
		return s.daemonStatus(ctx, httpClient), err
	}
	return s.daemonStatus(ctx, httpClient), nil
}

// StopDaemon shuts the daemon down and returns the resulting status.
func (s *Service) StopDaemon(ctx context.Context) (DaemonStatus, error) {
	httpClient, err := s.httpClient()
	if err != nil {
		return DaemonStatus{}, err
	}
	opCtx, cancel := context.WithTimeout(ctx, daemonControlTimeout)
	defer cancel()
	if err := daemon.Stop(opCtx, httpClient.BaseURL()); err != nil {
		return s.daemonStatus(ctx, httpClient), err
	}
	return s.daemonStatus(ctx, httpClient), nil
}

// RestartDaemon stops the daemon and starts a fresh one, returning the resulting status.
func (s *Service) RestartDaemon(ctx context.Context) (DaemonStatus, error) {
	httpClient, err := s.httpClient()
	if err != nil {
		return DaemonStatus{}, err
	}
	opCtx, cancel := context.WithTimeout(ctx, daemonControlTimeout)
	defer cancel()
	baseURL := httpClient.BaseURL()
	if err := daemon.Stop(opCtx, baseURL); err != nil {
		return s.daemonStatus(ctx, httpClient), err
	}
	if _, err := daemon.Ensure(opCtx, baseURL); err != nil {
		return s.daemonStatus(ctx, httpClient), err
	}
	return s.daemonStatus(ctx, httpClient), nil
}

func (s *Service) ClearDaemon(ctx context.Context, req protocol.ClearDaemonRequest) (protocol.ClearDaemonResponse, error) {
	return s.client.ClearDaemon(ctx, req)
}

func (s *Service) ListSessions(ctx context.Context) ([]session.Session, error) {
	return s.client.ListSessions(ctx)
}

func (s *Service) CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	return s.client.CreateSession(ctx, req)
}

func (s *Service) SplitPane(ctx context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	return s.client.SplitPane(ctx, req)
}

func (s *Service) SetSessionRootDir(ctx context.Context, req protocol.SetSessionRootDirRequest) (session.Session, error) {
	return s.client.SetSessionRootDir(ctx, req)
}

func (s *Service) SetPaneWorkingDir(ctx context.Context, req protocol.SetPaneWorkingDirRequest) (session.Session, error) {
	return s.client.SetPaneWorkingDir(ctx, req)
}

func (s *Service) StartPanePTY(ctx context.Context, req protocol.StartPanePTYRequest) (protocol.StartedPanePTY, error) {
	return s.client.StartPanePTY(ctx, req)
}

func (s *Service) RestartPanePTY(ctx context.Context, req protocol.RestartPanePTYRequest) (protocol.RestartedPanePTY, error) {
	return s.client.RestartPanePTY(ctx, req)
}

func (s *Service) DetachPanePTY(ctx context.Context, req protocol.DetachPanePTYRequest) (protocol.DetachedPanePTY, error) {
	return s.client.DetachPanePTY(ctx, req)
}

func (s *Service) CloseSession(ctx context.Context, req protocol.CloseSessionRequest) ([]session.Session, error) {
	return s.client.CloseSession(ctx, req)
}

func (s *Service) ClosePane(ctx context.Context, req protocol.ClosePaneRequest) (session.Session, error) {
	return s.client.ClosePane(ctx, req)
}

func (s *Service) WritePTY(ctx context.Context, req protocol.WritePTYRequest) error {
	return s.client.WritePTY(ctx, req)
}

func (s *Service) ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error {
	return s.client.ResizePTY(ctx, req)
}

func (s *Service) KillPTY(ctx context.Context, req protocol.KillPTYRequest) (protocol.PTYInfo, error) {
	return s.client.KillPTY(ctx, req)
}

func (s *Service) AddPTYBookmark(ctx context.Context, req protocol.AddPTYBookmarkRequest) (protocol.PTYBookmark, error) {
	return s.client.AddPTYBookmark(ctx, req)
}

func (s *Service) ListPTYBookmarks(ctx context.Context, ptyID string) ([]protocol.PTYBookmark, error) {
	return s.client.ListPTYBookmarks(ctx, ptyID)
}

func (s *Service) RemovePTYBookmark(ctx context.Context, req protocol.RemovePTYBookmarkRequest) error {
	return s.client.RemovePTYBookmark(ctx, req)
}

func (s *Service) Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	return s.client.Output(ctx, req)
}

func (s *Service) ListPTYs(ctx context.Context) ([]protocol.PTYInfo, error) {
	return s.client.ListPTYs(ctx)
}

func (s *Service) NextEvent(ctx context.Context, req protocol.NextEventRequest) (protocol.RuntimeEvent, error) {
	return s.client.NextEvent(ctx, req)
}

func (s *Service) DetectWorktrunk(ctx context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error) {
	return s.client.DetectWorktrunk(ctx, req)
}

func (s *Service) ListWorktrees(ctx context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error) {
	return s.client.ListWorktrees(ctx, req)
}

func (s *Service) CreateWorktree(ctx context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error) {
	return s.client.CreateWorktree(ctx, req)
}

func (s *Service) RemoveWorktree(ctx context.Context, req protocol.RemoveWorktreeRequest) error {
	return s.client.RemoveWorktree(ctx, req)
}

func (s *Service) ListProjects(ctx context.Context) ([]protocol.Project, error) {
	return s.client.ListProjects(ctx)
}

func (s *Service) CreateProject(ctx context.Context, req protocol.CreateProjectRequest) (protocol.Project, error) {
	return s.client.CreateProject(ctx, req)
}

func (s *Service) ListWorkflowTemplates(ctx context.Context) ([]protocol.WorkflowTemplate, error) {
	return s.client.ListWorkflowTemplates(ctx)
}

func (s *Service) ListPromptTemplates(ctx context.Context) ([]protocol.PromptTemplate, error) {
	return s.client.ListPromptTemplates(ctx)
}

func (s *Service) ListWorkItems(ctx context.Context, projectID string) ([]protocol.WorkItem, error) {
	return s.client.ListWorkItems(ctx, projectID)
}

func (s *Service) CreateWorkItem(ctx context.Context, req protocol.CreateWorkItemRequest) (protocol.WorkItem, error) {
	return s.client.CreateWorkItem(ctx, req)
}

func (s *Service) MoveWorkItem(ctx context.Context, req protocol.MoveWorkItemRequest) (protocol.WorkItem, error) {
	return s.client.MoveWorkItem(ctx, req)
}

func (s *Service) StartPlanning(ctx context.Context, req protocol.StartPlanningRequest) (protocol.WorkItemRun, error) {
	return s.client.StartPlanning(ctx, req)
}

func (s *Service) SubmitDraftPlan(ctx context.Context, req protocol.SubmitDraftPlanRequest) (protocol.Artifact, error) {
	return s.client.SubmitDraftPlan(ctx, req)
}

func (s *Service) ApprovePlan(ctx context.Context, req protocol.ApprovePlanRequest) (protocol.WorkItem, error) {
	return s.client.ApprovePlan(ctx, req)
}

func (s *Service) StartExecution(ctx context.Context, req protocol.StartExecutionRequest) (protocol.WorkItemRun, error) {
	return s.client.StartExecution(ctx, req)
}

func (s *Service) QueueExecution(ctx context.Context, req protocol.QueueExecutionRequest) (protocol.WorkItemRun, error) {
	return s.client.QueueExecution(ctx, req)
}

func (s *Service) LaunchExecution(ctx context.Context, req protocol.LaunchExecutionRequest) (protocol.WorkItemRun, error) {
	return s.client.LaunchExecution(ctx, req)
}

func (s *Service) AskQuestion(ctx context.Context, req protocol.AskQuestionRequest) (protocol.Question, error) {
	return s.client.AskQuestion(ctx, req)
}

func (s *Service) AnswerQuestion(ctx context.Context, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	return s.client.AnswerQuestion(ctx, req)
}

func (s *Service) CompleteExecution(ctx context.Context, req protocol.CompleteExecutionRequest) (protocol.WorkItem, error) {
	return s.client.CompleteExecution(ctx, req)
}

func (s *Service) SubmitReviewFeedback(ctx context.Context, req protocol.SubmitReviewFeedbackRequest) (protocol.Artifact, error) {
	return s.client.SubmitReviewFeedback(ctx, req)
}

func (s *Service) BindWorkItemWorktree(ctx context.Context, req protocol.BindWorkItemWorktreeRequest) (protocol.WorkItem, error) {
	return s.client.BindWorkItemWorktree(ctx, req)
}

func (s *Service) AddWorkItemAttachment(ctx context.Context, req protocol.AddWorkItemAttachmentRequest) (protocol.WorkItem, error) {
	return s.client.AddWorkItemAttachment(ctx, req)
}

func (s *Service) DeleteWorkItem(ctx context.Context, req protocol.DeleteWorkItemRequest) (protocol.WorkItem, error) {
	return s.client.DeleteWorkItem(ctx, req)
}

func (s *Service) ListWorkItemRuns(ctx context.Context, workItemID string) ([]protocol.WorkItemRun, error) {
	return s.client.ListWorkItemRuns(ctx, workItemID)
}

func (s *Service) StartWorkItemRun(ctx context.Context, req protocol.StartWorkItemRunRequest) (protocol.WorkItemRun, error) {
	return s.client.StartWorkItemRun(ctx, req)
}

func (s *Service) LaunchWorkItemRun(ctx context.Context, req protocol.LaunchWorkItemRunRequest) (protocol.WorkItemRun, error) {
	return s.client.LaunchWorkItemRun(ctx, req)
}

func (s *Service) CancelWorkItemRun(ctx context.Context, req protocol.CancelWorkItemRunRequest) (protocol.WorkItemRun, error) {
	return s.client.CancelWorkItemRun(ctx, req)
}

func (s *Service) ApproveDone(ctx context.Context, req protocol.ApproveDoneRequest) (protocol.WorkItem, error) {
	return s.client.ApproveDone(ctx, req)
}

func (s *Service) ListArtifacts(ctx context.Context, workItemID string) ([]protocol.Artifact, error) {
	return s.client.ListArtifacts(ctx, workItemID)
}

func (s *Service) ListQuestions(ctx context.Context, workItemID string) ([]protocol.Question, error) {
	return s.client.ListQuestions(ctx, workItemID)
}

func (s *Service) ListGateReports(ctx context.Context, workItemID string) ([]protocol.GateReport, error) {
	return s.client.ListGateReports(ctx, workItemID)
}

func (s *Service) CompleteGate(ctx context.Context, req protocol.CompleteGateRequest) (protocol.GateReport, error) {
	return s.client.CompleteGate(ctx, req)
}

func (s *Service) ListWorkflowEvents(ctx context.Context, workItemID string) ([]protocol.WorkflowEvent, error) {
	return s.client.ListWorkflowEvents(ctx, workItemID)
}

func (s *Service) ReportStatus(ctx context.Context, req protocol.ReportStatusRequest) (protocol.ReportStatusResponse, error) {
	return s.client.ReportStatus(ctx, req)
}

func (s *Service) ListStatusEvents(ctx context.Context, req protocol.ListStatusEventsRequest) ([]protocol.StatusEvent, error) {
	return s.client.ListStatusEvents(ctx, req)
}

func (s *Service) MarkStatusEventRead(ctx context.Context, req protocol.MarkStatusEventReadRequest) (protocol.StatusEvent, error) {
	return s.client.MarkStatusEventRead(ctx, req)
}

func (s *Service) ListHTTPForwards(ctx context.Context) ([]protocol.HTTPForward, error) {
	return s.client.ListHTTPForwards(ctx)
}

func (s *Service) StartHTTPForward(ctx context.Context, req protocol.StartHTTPForwardRequest) (protocol.StartedHTTPForward, error) {
	if s.forwarder == nil {
		return protocol.StartedHTTPForward{}, fmt.Errorf("local HTTP forwarding requires an HTTP daemon client")
	}
	return s.forwarder.Start(ctx, req)
}

func (s *Service) StopHTTPForward(ctx context.Context, id string) error {
	if s.forwarder == nil {
		return fmt.Errorf("local HTTP forwarding requires an HTTP daemon client")
	}
	return s.forwarder.Stop(ctx, id)
}
