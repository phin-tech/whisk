package wailsapp

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

type Service struct {
	client    client.RuntimeClient
	forwarder *client.LocalForwarder
}

func NewService(runtimeClient client.RuntimeClient) *Service {
	service := &Service{client: runtimeClient}
	if httpClient, ok := runtimeClient.(*client.HTTPClient); ok {
		service.forwarder = client.NewLocalForwarder(httpClient, nil)
	}
	return service
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
