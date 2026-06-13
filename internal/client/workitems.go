package client

import (
	"context"
	"net/url"

	"github.com/phin-tech/whisk/internal/protocol"
)

func (c *HTTPClient) ListProjects(ctx context.Context) ([]protocol.Project, error) {
	var projects []protocol.Project
	err := c.get(ctx, "/v1/projects", nil, &projects)
	return projects, err
}

func (c *HTTPClient) CreateProject(ctx context.Context, req protocol.CreateProjectRequest) (protocol.Project, error) {
	var project protocol.Project
	err := c.post(ctx, "/v1/projects", req, &project)
	return project, err
}

func (c *HTTPClient) ListWorkflowTemplates(ctx context.Context) ([]protocol.WorkflowTemplate, error) {
	var templates []protocol.WorkflowTemplate
	err := c.get(ctx, "/v1/workflow-templates", nil, &templates)
	return templates, err
}

func (c *HTTPClient) ListPromptTemplates(ctx context.Context) ([]protocol.PromptTemplate, error) {
	var templates []protocol.PromptTemplate
	err := c.get(ctx, "/v1/prompt-templates", nil, &templates)
	return templates, err
}

func (c *HTTPClient) ListWorkItems(ctx context.Context, projectID string) ([]protocol.WorkItem, error) {
	query := url.Values{}
	if projectID != "" {
		query.Set("projectId", projectID)
	}
	var items []protocol.WorkItem
	err := c.get(ctx, "/v1/work-items", query, &items)
	return items, err
}

func (c *HTTPClient) CreateWorkItem(ctx context.Context, req protocol.CreateWorkItemRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	err := c.post(ctx, "/v1/work-items", req, &item)
	return item, err
}

func (c *HTTPClient) MoveWorkItem(ctx context.Context, req protocol.MoveWorkItemRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.ID) + "/move"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) StartPlanning(ctx context.Context, req protocol.StartPlanningRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/start-planning"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) SubmitDraftPlan(ctx context.Context, req protocol.SubmitDraftPlanRequest) (protocol.Artifact, error) {
	var artifact protocol.Artifact
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/plan-drafts"
	err := c.post(ctx, path, req, &artifact)
	return artifact, err
}

func (c *HTTPClient) ApprovePlan(ctx context.Context, req protocol.ApprovePlanRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/approve-plan"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) StartExecution(ctx context.Context, req protocol.StartExecutionRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/start-execution"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) QueueExecution(ctx context.Context, req protocol.QueueExecutionRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/queue-execution"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) LaunchExecution(ctx context.Context, req protocol.LaunchExecutionRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/launch-execution"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) AskQuestion(ctx context.Context, req protocol.AskQuestionRequest) (protocol.Question, error) {
	var question protocol.Question
	err := c.post(ctx, "/v1/questions", req, &question)
	return question, err
}

func (c *HTTPClient) AnswerQuestion(ctx context.Context, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	var question protocol.Question
	path := "/v1/questions/" + url.PathEscape(req.ID) + "/answer"
	err := c.post(ctx, path, req, &question)
	return question, err
}

func (c *HTTPClient) CompleteExecution(ctx context.Context, req protocol.CompleteExecutionRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-item-runs/" + url.PathEscape(req.RunID) + "/complete-execution"
	if req.WorkItemID != "" {
		path = "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/complete-execution"
	}
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) SubmitReviewFeedback(ctx context.Context, req protocol.SubmitReviewFeedbackRequest) (protocol.Artifact, error) {
	var artifact protocol.Artifact
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/review-feedback"
	err := c.post(ctx, path, req, &artifact)
	return artifact, err
}

func (c *HTTPClient) ApproveDone(ctx context.Context, req protocol.ApproveDoneRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/approve-done"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) ListArtifacts(ctx context.Context, workItemID string) ([]protocol.Artifact, error) {
	query := url.Values{}
	if workItemID != "" {
		query.Set("workItemId", workItemID)
	}
	var artifacts []protocol.Artifact
	err := c.get(ctx, "/v1/artifacts", query, &artifacts)
	return artifacts, err
}

func (c *HTTPClient) ListQuestions(ctx context.Context, workItemID string) ([]protocol.Question, error) {
	query := url.Values{}
	if workItemID != "" {
		query.Set("workItemId", workItemID)
	}
	var questions []protocol.Question
	err := c.get(ctx, "/v1/questions", query, &questions)
	return questions, err
}

func (c *HTTPClient) ListGateReports(ctx context.Context, workItemID string) ([]protocol.GateReport, error) {
	query := url.Values{}
	if workItemID != "" {
		query.Set("workItemId", workItemID)
	}
	var gates []protocol.GateReport
	err := c.get(ctx, "/v1/gate-reports", query, &gates)
	return gates, err
}

func (c *HTTPClient) CompleteGate(ctx context.Context, req protocol.CompleteGateRequest) (protocol.GateReport, error) {
	var gate protocol.GateReport
	path := "/v1/gate-reports/" + url.PathEscape(req.ID) + "/complete"
	err := c.post(ctx, path, req, &gate)
	return gate, err
}

func (c *HTTPClient) ListWorkflowEvents(ctx context.Context, workItemID string) ([]protocol.WorkflowEvent, error) {
	query := url.Values{}
	if workItemID != "" {
		query.Set("workItemId", workItemID)
	}
	var events []protocol.WorkflowEvent
	err := c.get(ctx, "/v1/workflow-events", query, &events)
	return events, err
}

func (c *HTTPClient) BindWorkItemWorktree(ctx context.Context, req protocol.BindWorkItemWorktreeRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.ID) + "/bind-worktree"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) AddWorkItemAttachment(ctx context.Context, req protocol.AddWorkItemAttachmentRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.WorkItemID) + "/attachments"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) DeleteWorkItem(ctx context.Context, req protocol.DeleteWorkItemRequest) (protocol.WorkItem, error) {
	var item protocol.WorkItem
	path := "/v1/work-items/" + url.PathEscape(req.ID) + "/delete"
	err := c.post(ctx, path, req, &item)
	return item, err
}

func (c *HTTPClient) ListWorkItemRuns(ctx context.Context, workItemID string) ([]protocol.WorkItemRun, error) {
	query := url.Values{}
	if workItemID != "" {
		query.Set("workItemId", workItemID)
	}
	var runs []protocol.WorkItemRun
	err := c.get(ctx, "/v1/work-item-runs", query, &runs)
	return runs, err
}

func (c *HTTPClient) StartWorkItemRun(ctx context.Context, req protocol.StartWorkItemRunRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	err := c.post(ctx, "/v1/work-item-runs", req, &run)
	return run, err
}

func (c *HTTPClient) LaunchWorkItemRun(ctx context.Context, req protocol.LaunchWorkItemRunRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-item-runs/" + url.PathEscape(req.ID) + "/launch"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) CancelWorkItemRun(ctx context.Context, req protocol.CancelWorkItemRunRequest) (protocol.WorkItemRun, error) {
	var run protocol.WorkItemRun
	path := "/v1/work-item-runs/" + url.PathEscape(req.ID) + "/cancel"
	err := c.post(ctx, path, req, &run)
	return run, err
}

func (c *HTTPClient) ReportStatus(ctx context.Context, req protocol.ReportStatusRequest) (protocol.ReportStatusResponse, error) {
	var report protocol.ReportStatusResponse
	err := c.post(ctx, "/v1/status", req, &report)
	return report, err
}

func (c *HTTPClient) ListStatusEvents(ctx context.Context, req protocol.ListStatusEventsRequest) ([]protocol.StatusEvent, error) {
	query := url.Values{}
	if req.ProjectID != "" {
		query.Set("projectId", req.ProjectID)
	}
	if req.WorkItemID != "" {
		query.Set("workItemId", req.WorkItemID)
	}
	if req.RunID != "" {
		query.Set("runId", req.RunID)
	}
	if req.SessionID != "" {
		query.Set("sessionId", req.SessionID)
	}
	if req.PTYID != "" {
		query.Set("ptyId", req.PTYID)
	}
	if req.UnreadOnly {
		query.Set("unreadOnly", "true")
	}
	var events []protocol.StatusEvent
	err := c.get(ctx, "/v1/status-events", query, &events)
	return events, err
}

func (c *HTTPClient) MarkStatusEventRead(ctx context.Context, req protocol.MarkStatusEventReadRequest) (protocol.StatusEvent, error) {
	var event protocol.StatusEvent
	path := "/v1/status-events/" + url.PathEscape(req.ID) + "/read"
	err := c.post(ctx, path, req, &event)
	return event, err
}
