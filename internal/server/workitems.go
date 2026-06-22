package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.runtime.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *HTTPServer) createProject(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.CreateProject(r.Context(), app.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		Slug:        req.Slug,
		RootDir:     req.RootDir,
		WorkflowID:  req.WorkflowID,
		Preferences: req.Preferences,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *HTTPServer) updateProject(w http.ResponseWriter, r *http.Request) {
	var req protocol.UpdateProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.UpdateProject(r.Context(), app.UpdateProjectRequest{
		ID:                       pathValue(r, "projectID", ""),
		Name:                     req.Name,
		Description:              req.Description,
		Slug:                     req.Slug,
		UseInteractiveAgentShell: req.UseInteractiveAgentShell,
		DefaultPhaseAgents:       req.DefaultPhaseAgents,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *HTTPServer) deleteProject(w http.ResponseWriter, r *http.Request) {
	var req protocol.DeleteProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.DeleteProject(r.Context(), app.DeleteProjectRequest{
		ID:    pathValue(r, "projectID", ""),
		Actor: req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *HTTPServer) getProjectDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := s.runtime.GetProjectDetail(r.Context(), pathValue(r, "projectID", ""))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.ProjectDetail{
		Project:   detail.Project,
		WorkItems: detail.WorkItems,
		Sessions:  detail.Sessions,
		Runs:      detail.Runs,
	})
}

func (s *HTTPServer) addProjectAttachment(w http.ResponseWriter, r *http.Request) {
	var req protocol.AddProjectAttachmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ProjectID = pathValue(r, "projectID", req.ProjectID)
	project, err := s.runtime.AddProjectAttachment(r.Context(), app.AddProjectAttachmentRequest{
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
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *HTTPServer) updateProjectAttachment(w http.ResponseWriter, r *http.Request) {
	var req protocol.UpdateProjectAttachmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.UpdateProjectAttachment(r.Context(), app.UpdateProjectAttachmentRequest{
		ID:               pathValue(r, "attachmentID", ""),
		ProjectID:        req.ProjectID,
		Title:            req.Title,
		Path:             req.Path,
		URL:              req.URL,
		Note:             req.Note,
		Provider:         req.Provider,
		Target:           req.Target,
		IncludeInContext: req.IncludeInContext,
		Meta:             req.Meta,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *HTTPServer) deleteProjectAttachment(w http.ResponseWriter, r *http.Request) {
	var req protocol.DeleteProjectAttachmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.DeleteProjectAttachment(r.Context(), app.DeleteProjectAttachmentRequest{
		ID:        pathValue(r, "attachmentID", ""),
		ProjectID: req.ProjectID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *HTTPServer) getProjectContext(w http.ResponseWriter, r *http.Request) {
	context, err := s.runtime.ProjectContext(r.Context(), pathValue(r, "projectID", ""))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.ProjectContext{
		ProjectID: context.ProjectID,
		Items:     toProtocolProjectContextItems(context.Items),
	})
}

func toProtocolProjectContextItems(items []app.ProjectContextItem) []protocol.ProjectContextItem {
	out := make([]protocol.ProjectContextItem, 0, len(items))
	for _, item := range items {
		out = append(out, protocol.ProjectContextItem{
			AttachmentID: item.AttachmentID,
			Kind:         item.Kind,
			Provider:     item.Provider,
			Target:       item.Target,
			Title:        item.Title,
			Delivery:     item.Delivery,
			ContentType:  item.ContentType,
			Content:      item.Content,
			SourceURL:    item.SourceURL,
			Error:        item.Error,
		})
	}
	return out
}

func (s *HTTPServer) startPlanning(w http.ResponseWriter, r *http.Request) {
	var req protocol.StartPlanningRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	run, err := s.runtime.StartPlanning(r.Context(), app.StartPlanningRequest{
		WorkItemID:     req.WorkItemID,
		SessionID:      req.SessionID,
		PTYID:          req.PTYID,
		Launch:         req.Launch,
		AgentProfileID: req.AgentProfileID,
		SystemPrompt:   req.SystemPrompt,
		Actor:          req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *HTTPServer) submitDraftPlan(w http.ResponseWriter, r *http.Request) {
	var req protocol.SubmitDraftPlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	artifact, err := s.runtime.SubmitDraftPlan(r.Context(), app.SubmitDraftPlanRequest{
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		Title:      req.Title,
		Body:       req.Body,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (s *HTTPServer) approvePlan(w http.ResponseWriter, r *http.Request) {
	var req protocol.ApprovePlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	item, err := s.runtime.ApprovePlan(r.Context(), app.ApprovePlanRequest{
		ArtifactID: req.ArtifactID,
		WorkItemID: req.WorkItemID,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) startExecution(w http.ResponseWriter, r *http.Request) {
	var req protocol.StartExecutionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	run, err := s.runtime.StartExecution(r.Context(), app.StartExecutionRequest{
		WorkItemID:     req.WorkItemID,
		SessionID:      req.SessionID,
		PTYID:          req.PTYID,
		Launch:         req.Launch,
		AgentProfileID: req.AgentProfileID,
		SystemPrompt:   req.SystemPrompt,
		Actor:          req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *HTTPServer) queueExecution(w http.ResponseWriter, r *http.Request) {
	var req protocol.QueueExecutionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	run, err := s.runtime.QueueExecution(r.Context(), app.QueueExecutionRequest{
		WorkItemID: req.WorkItemID,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *HTTPServer) launchExecution(w http.ResponseWriter, r *http.Request) {
	var req protocol.LaunchExecutionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	run, err := s.runtime.LaunchExecution(r.Context(), app.LaunchExecutionRequest{
		WorkItemID:     req.WorkItemID,
		AgentProfileID: req.AgentProfileID,
		SystemPrompt:   req.SystemPrompt,
		Actor:          req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *HTTPServer) askQuestion(w http.ResponseWriter, r *http.Request) {
	var req protocol.AskQuestionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	question, err := s.runtime.AskQuestion(r.Context(), app.AskQuestionRequest{
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		Prompt:     req.Prompt,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, question)
}

func (s *HTTPServer) answerQuestion(w http.ResponseWriter, r *http.Request) {
	var req protocol.AnswerQuestionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "questionID", req.ID)
	question, err := s.runtime.AnswerQuestion(r.Context(), app.AnswerQuestionRequest{
		ID:     req.ID,
		Answer: req.Answer,
		Actor:  req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, question)
}

func (s *HTTPServer) completeExecution(w http.ResponseWriter, r *http.Request) {
	var req protocol.CompleteExecutionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.RunID = pathValue(r, "runID", req.RunID)
	item, err := s.runtime.CompleteExecution(r.Context(), app.CompleteExecutionRequest{
		RunID:   req.RunID,
		Message: req.Message,
		Actor:   req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) submitReviewFeedback(w http.ResponseWriter, r *http.Request) {
	var req protocol.SubmitReviewFeedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	artifact, err := s.runtime.SubmitReviewFeedback(r.Context(), app.SubmitReviewFeedbackRequest{
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		Body:       req.Body,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (s *HTTPServer) approveDone(w http.ResponseWriter, r *http.Request) {
	var req protocol.ApproveDoneRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	item, err := s.runtime.ApproveDone(r.Context(), app.ApproveDoneRequest{
		WorkItemID: req.WorkItemID,
		Reason:     req.Reason,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) listArtifacts(w http.ResponseWriter, r *http.Request) {
	artifacts, err := s.runtime.ListArtifacts(r.Context(), r.URL.Query().Get("workItemId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, artifacts)
}

func (s *HTTPServer) listQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := s.runtime.ListQuestions(r.Context(), r.URL.Query().Get("workItemId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, questions)
}

func (s *HTTPServer) listGateReports(w http.ResponseWriter, r *http.Request) {
	gates, err := s.runtime.ListGateReports(r.Context(), r.URL.Query().Get("workItemId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, gates)
}

func (s *HTTPServer) completeGate(w http.ResponseWriter, r *http.Request) {
	var req protocol.CompleteGateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "gateReportID", req.ID)
	gate, err := s.runtime.CompleteGate(r.Context(), app.CompleteGateRequest{
		ID:             req.ID,
		Status:         req.Status,
		OverrideReason: req.OverrideReason,
		Actor:          req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, gate)
}

func (s *HTTPServer) listWorkflowEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.runtime.ListWorkflowEvents(r.Context(), r.URL.Query().Get("workItemId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *HTTPServer) listWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.runtime.ListWorkflowTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *HTTPServer) listPromptTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.runtime.ListPromptTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *HTTPServer) listAgentProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.runtime.ListAgentProfiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.AgentProfile, len(profiles))
	for i, profile := range profiles {
		out[i] = protocol.AgentProfile{
			ID:          profile.ID,
			Provider:    string(profile.Provider),
			Label:       profile.Label,
			Description: profile.Description,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) listWorkItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.runtime.ListWorkItems(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *HTTPServer) createWorkItem(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateWorkItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.runtime.CreateWorkItem(r.Context(), app.CreateWorkItemRequest{
		ProjectID:    req.ProjectID,
		WorkflowID:   req.WorkflowID,
		Title:        req.Title,
		BodyMarkdown: req.BodyMarkdown,
		StageID:      req.StageID,
		Actor:        req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *HTTPServer) moveWorkItem(w http.ResponseWriter, r *http.Request) {
	var req protocol.MoveWorkItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "workItemID", req.ID)
	item, err := s.runtime.MoveWorkItem(r.Context(), app.MoveWorkItemRequest{
		ID:      req.ID,
		StageID: req.StageID,
		Actor:   req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) bindWorkItemWorktree(w http.ResponseWriter, r *http.Request) {
	var req protocol.BindWorkItemWorktreeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "workItemID", req.ID)
	item, err := s.runtime.BindWorkItemWorktree(r.Context(), app.BindWorkItemWorktreeRequest{
		ID:           req.ID,
		Branch:       req.Branch,
		Base:         req.Base,
		WorktreePath: req.WorktreePath,
		Actor:        req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) addWorkItemAttachment(w http.ResponseWriter, r *http.Request) {
	var req protocol.AddWorkItemAttachmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.WorkItemID = pathValue(r, "workItemID", req.WorkItemID)
	item, err := s.runtime.AddWorkItemAttachment(r.Context(), app.AddWorkItemAttachmentRequest{
		WorkItemID: req.WorkItemID,
		Kind:       req.Kind,
		Scope:      req.Scope,
		Path:       req.Path,
		URL:        req.URL,
		Note:       req.Note,
		Actor:      req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *HTTPServer) deleteWorkItem(w http.ResponseWriter, r *http.Request) {
	var req protocol.DeleteWorkItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "workItemID", req.ID)
	item, err := s.runtime.DeleteWorkItem(r.Context(), app.DeleteWorkItemRequest{
		ID:    req.ID,
		Actor: req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *HTTPServer) listWorkItemRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.runtime.ListWorkItemRuns(r.Context(), r.URL.Query().Get("workItemId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *HTTPServer) startWorkItemRun(w http.ResponseWriter, r *http.Request) {
	var req protocol.StartWorkItemRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	run, err := s.runtime.StartWorkItemRun(r.Context(), app.StartWorkItemRunRequest{
		WorkItemID:       req.WorkItemID,
		Preset:           req.Preset,
		PromptTemplateID: req.PromptTemplateID,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		Launch:           req.Launch,
		AgentProfileID:   req.AgentProfileID,
		SystemPrompt:     req.SystemPrompt,
		Actor:            req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *HTTPServer) launchWorkItemRun(w http.ResponseWriter, r *http.Request) {
	var req protocol.LaunchWorkItemRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "runID", req.ID)
	run, err := s.runtime.LaunchWorkItemRun(r.Context(), app.LaunchWorkItemRunRequest{
		ID:             req.ID,
		AgentProfileID: req.AgentProfileID,
		SystemPrompt:   req.SystemPrompt,
		Actor:          req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *HTTPServer) cancelWorkItemRun(w http.ResponseWriter, r *http.Request) {
	var req protocol.CancelWorkItemRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "runID", req.ID)
	run, err := s.runtime.CancelWorkItemRun(r.Context(), app.CancelWorkItemRunRequest{
		ID:    req.ID,
		Actor: req.Actor,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *HTTPServer) reportStatus(w http.ResponseWriter, r *http.Request) {
	var req protocol.ReportStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	report, err := s.runtime.ReportStatus(r.Context(), app.ReportStatusRequest{
		Kind:       req.Kind,
		Message:    req.Message,
		Actor:      req.Actor,
		ProjectID:  req.ProjectID,
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.ReportStatusResponse{
		Event:    report.Event,
		Run:      report.Run,
		WorkItem: report.WorkItem,
	})
}

func (s *HTTPServer) listStatusEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.runtime.ListStatusEvents(r.Context(), app.ListStatusEventsRequest{
		ProjectID:  r.URL.Query().Get("projectId"),
		WorkItemID: r.URL.Query().Get("workItemId"),
		RunID:      r.URL.Query().Get("runId"),
		SessionID:  r.URL.Query().Get("sessionId"),
		PTYID:      r.URL.Query().Get("ptyId"),
		UnreadOnly: r.URL.Query().Get("unreadOnly") == "true",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *HTTPServer) markStatusEventRead(w http.ResponseWriter, r *http.Request) {
	var req protocol.MarkStatusEventReadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "statusEventID", req.ID)
	event, err := s.runtime.MarkStatusEventRead(r.Context(), app.MarkStatusEventReadRequest{ID: req.ID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}
