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
		Name:       req.Name,
		Slug:       req.Slug,
		RootDir:    req.RootDir,
		WorkflowID: req.WorkflowID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
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
