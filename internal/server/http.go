package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/buildinfo"
	"github.com/phin-tech/whisk/internal/controlauth"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/ptytrace"
)

type HTTPOption func(*httpOptions)

type httpOptions struct {
	controlToken             string
	agentHookRequestMaxBytes int64
	agentHookReadTimeout     time.Duration
	ptyOutputBatchInterval   time.Duration
}

func WithControlToken(token string) HTTPOption {
	return func(opts *httpOptions) {
		opts.controlToken = token
	}
}

func WithAgentHookReceiverLimits(maxBytes int64, readTimeout time.Duration) HTTPOption {
	return func(opts *httpOptions) {
		opts.agentHookRequestMaxBytes = maxBytes
		opts.agentHookReadTimeout = readTimeout
	}
}

func WithPTYOutputBatchInterval(interval time.Duration) HTTPOption {
	return func(opts *httpOptions) {
		opts.ptyOutputBatchInterval = interval
	}
}

const (
	defaultAgentHookRequestMaxBytes = 1 * 1024 * 1024
	defaultAgentHookReadTimeout     = 5 * time.Second
	defaultPTYOutputBatchInterval   = 8 * time.Millisecond
	ptyRecentInputWindow            = 100 * time.Millisecond
	ptyRecentInputFlushMaxBytes     = 1024
)

func RequireBearerAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !controlauth.TokensEqual(token, requestControlToken(r)) {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestControlToken(r *http.Request) string {
	if token, ok := controlauth.BearerToken(r.Header.Get(controlauth.AuthorizationHeader)); ok {
		return token
	}
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/ptys/") && strings.HasSuffix(r.URL.Path, "/attach") {
		return r.URL.Query().Get(controlauth.AccessTokenQueryName)
	}
	return ""
}

func NewHTTP(runtime *app.Runtime, optionFns ...HTTPOption) http.Handler {
	opts := httpOptions{}
	for _, option := range optionFns {
		option(&opts)
	}
	if opts.agentHookRequestMaxBytes <= 0 {
		opts.agentHookRequestMaxBytes = defaultAgentHookRequestMaxBytes
	}
	if opts.agentHookReadTimeout <= 0 {
		opts.agentHookReadTimeout = defaultAgentHookReadTimeout
	}
	if opts.ptyOutputBatchInterval <= 0 {
		opts.ptyOutputBatchInterval = defaultPTYOutputBatchInterval
	}
	server := &HTTPServer{
		runtime:                  runtime,
		agentHookRequestMaxBytes: opts.agentHookRequestMaxBytes,
		agentHookReadTimeout:     opts.agentHookReadTimeout,
		ptyOutputBatchInterval:   opts.ptyOutputBatchInterval,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", server.health)
	mux.HandleFunc("GET /v1/compat", server.compatibility)
	mux.HandleFunc("POST /v1/daemon/clear", server.clearDaemon)
	mux.HandleFunc("GET /v1/onboarding", server.onboardingStatus)
	mux.HandleFunc("POST /v1/onboarding/apply", server.applyOnboarding)
	mux.HandleFunc("POST /v1/agent-bridges/{bridgeID}/hooks", server.agentBridgeHook)
	mux.HandleFunc("POST /v1/agent-hook-events", server.recordAgentHookEvent)
	mux.HandleFunc("GET /v1/agent-bridge-approvals", server.listAgentBridgeApprovals)
	mux.HandleFunc("POST /v1/agent-bridge-approvals/{approvalID}/resolve", server.resolveAgentBridgeApproval)
	mux.HandleFunc("GET /v1/agent-prompts", server.listAgentPrompts)
	mux.HandleFunc("POST /v1/agent-prompts/{promptID}/resolve", server.resolveAgentPrompt)
	mux.HandleFunc("GET /v1/agent-bridge-events", server.listAgentBridgeEvents)
	mux.HandleFunc("POST /v1/agent-bridge-events/{eventID}/read", server.markAgentBridgeEventRead)
	mux.HandleFunc("GET /v1/agent-hook-integrations", server.listAgentHookIntegrations)
	mux.HandleFunc("POST /v1/agent-hook-integrations/check", server.checkAgentHookIntegration)
	mux.HandleFunc("POST /v1/agent-hook-integrations/install", server.installAgentHookIntegration)
	mux.HandleFunc("POST /v1/agent-hook-integrations/remove", server.removeAgentHookIntegration)
	mux.HandleFunc("GET /v1/agent-hook-log", server.agentHookLogStatus)
	mux.HandleFunc("POST /v1/agent-hook-log/settings", server.setAgentHookLogSettings)
	mux.HandleFunc("POST /v1/agent-hook-log/clear", server.clearAgentHookLog)
	mux.HandleFunc("POST /v1/agent-hook-log/open", server.openAgentHookLog)
	mux.HandleFunc("GET /v1/plugins", server.listPlugins)
	mux.HandleFunc("GET /v1/plugin-registry", server.listRegistryPlugins)
	mux.HandleFunc("POST /v1/plugin-registry/install", server.installRegistryPlugin)
	mux.HandleFunc("POST /v1/plugins/rescan", server.rescanPlugins)
	mux.HandleFunc("POST /v1/plugins/{pluginID}/trust", server.trustPlugin)
	mux.HandleFunc("POST /v1/plugins/{pluginID}/untrust", server.untrustPlugin)
	mux.HandleFunc("POST /v1/plugins/{pluginID}/project-attachment-templates/{templateID}", server.runPluginProjectAttachmentTemplate)
	mux.HandleFunc("GET /v1/ui-contributions", server.listUIContributions)
	mux.HandleFunc("GET /v1/sessions", server.listSessions)
	mux.HandleFunc("POST /v1/sessions", server.createSession)
	mux.HandleFunc("DELETE /v1/sessions/{sessionID}", server.closeSession)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/split", server.splitPane)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/set-root-dir", server.setSessionRootDir)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/set-project", server.setSessionProject)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/set-working-dir", server.setPaneWorkingDir)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/start-pty", server.startPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/restart-pty", server.restartPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/detach-pty", server.detachPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/windows/{windowID}/panes/{paneID}/close", server.closePane)
	mux.HandleFunc("GET /v1/ptys", server.listPTYs)
	mux.HandleFunc("GET /v1/pty-history", server.listPTYHistory)
	mux.HandleFunc("GET /v1/pty-history/{ptyID}", server.readPTYHistory)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/write", server.writePTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/resize", server.resizePTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/kill", server.killPTY)
	mux.HandleFunc("DELETE /v1/ptys/{ptyID}", server.deletePTY)
	mux.HandleFunc("GET /v1/ptys/{ptyID}/attach", server.attachPTY)
	mux.HandleFunc("GET /v1/ptys/{ptyID}/output", server.output)
	mux.HandleFunc("POST /v1/worktrunk/detect", server.detectWorktrunk)
	mux.HandleFunc("POST /v1/worktrees/list", server.listWorktrees)
	mux.HandleFunc("POST /v1/worktrees/create", server.createWorktree)
	mux.HandleFunc("POST /v1/worktrees/remove", server.removeWorktree)
	mux.HandleFunc("POST /v1/http-forwards", server.createHTTPForward)
	mux.HandleFunc("GET /v1/http-forwards", server.listHTTPForwards)
	mux.HandleFunc("DELETE /v1/http-forwards/{forwardID}", server.deleteHTTPForward)
	mux.HandleFunc("/v1/http-forwards/{forwardID}/proxy", server.proxyHTTPForward)
	mux.HandleFunc("/v1/http-forwards/{forwardID}/proxy/", server.proxyHTTPForward)
	mux.HandleFunc("GET /v1/events/next", server.nextEvent)
	mux.HandleFunc("POST /v1/mail", server.sendMail)
	mux.HandleFunc("GET /v1/mail", server.listMail)
	mux.HandleFunc("GET /v1/mail/next", server.nextMail)
	mux.HandleFunc("POST /v1/mail/{mailID}/read", server.markMailRead)
	mux.HandleFunc("POST /v1/mail/{mailID}/reply", server.replyMail)
	mux.HandleFunc("GET /v1/projects", server.listProjects)
	mux.HandleFunc("POST /v1/projects", server.createProject)
	mux.HandleFunc("POST /v1/projects/{projectID}/update", server.updateProject)
	mux.HandleFunc("POST /v1/projects/{projectID}/delete", server.deleteProject)
	mux.HandleFunc("GET /v1/projects/{projectID}/detail", server.getProjectDetail)
	mux.HandleFunc("POST /v1/projects/{projectID}/workflow-definition", server.setProjectWorkflowDefinition)
	mux.HandleFunc("POST /v1/projects/{projectID}/workflow-migration-plan", server.planProjectWorkflowMigration)
	mux.HandleFunc("POST /v1/projects/{projectID}/attachments", server.addProjectAttachment)
	mux.HandleFunc("POST /v1/project-attachments/{attachmentID}/update", server.updateProjectAttachment)
	mux.HandleFunc("POST /v1/project-attachments/{attachmentID}/delete", server.deleteProjectAttachment)
	mux.HandleFunc("GET /v1/projects/{projectID}/context", server.getProjectContext)
	mux.HandleFunc("GET /v1/workflow-definitions", server.listWorkflowDefinitions)
	mux.HandleFunc("POST /v1/workflow-definitions/validate", server.validateWorkflowDefinition)
	mux.HandleFunc("POST /v1/workflow-definitions/validate-file", server.validateWorkflowDefinitionFile)
	mux.HandleFunc("POST /v1/workflow-definitions/import", server.importWorkflowDefinition)
	mux.HandleFunc("POST /v1/workflow-definitions/import-file", server.importWorkflowDefinitionFile)
	mux.HandleFunc("POST /v1/workflow-definitions/export-file", server.exportWorkflowDefinitionFile)
	mux.HandleFunc("POST /v1/workflow-definitions/{workflowID}/{version}/delete", server.deleteWorkflowDefinition)
	mux.HandleFunc("GET /v1/workflow-templates", server.listWorkflowTemplates)
	mux.HandleFunc("GET /v1/prompt-templates", server.listPromptTemplates)
	mux.HandleFunc("GET /v1/agent-profiles", server.listAgentProfiles)
	mux.HandleFunc("GET /v1/agents/detected", server.listDetectedAgents)
	mux.HandleFunc("GET /v1/work-items", server.listWorkItems)
	mux.HandleFunc("POST /v1/work-items", server.createWorkItem)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/update", server.updateWorkItem)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/move", server.moveWorkItem)
	mux.HandleFunc("GET /v1/work-items/{workItemID}/workflow-actions", server.listWorkItemWorkflowActions)
	mux.HandleFunc("GET /v1/work-item-links", server.listWorkItemLinks)
	mux.HandleFunc("POST /v1/work-item-links", server.addWorkItemLink)
	mux.HandleFunc("GET /v1/ready-work", server.readyWork)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/start-planning", server.startPlanning)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/plan-drafts", server.submitDraftPlan)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/approve-plan", server.approvePlan)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/start-execution", server.startExecution)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/queue-execution", server.queueExecution)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/launch-execution", server.launchExecution)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/complete-execution", server.completeExecution)
	mux.HandleFunc("POST /v1/work-item-runs/{runID}/complete-execution", server.completeExecution)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/review-feedback", server.submitReviewFeedback)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/approve-done", server.approveDone)
	mux.HandleFunc("GET /v1/artifacts", server.listArtifacts)
	mux.HandleFunc("GET /v1/questions", server.listQuestions)
	mux.HandleFunc("POST /v1/questions", server.askQuestion)
	mux.HandleFunc("POST /v1/questions/{questionID}/answer", server.answerQuestion)
	mux.HandleFunc("GET /v1/gate-reports", server.listGateReports)
	mux.HandleFunc("POST /v1/gate-reports/{gateReportID}/complete", server.completeGate)
	mux.HandleFunc("GET /v1/workflow-events", server.listWorkflowEvents)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/bind-worktree", server.bindWorkItemWorktree)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/attachments", server.addWorkItemAttachment)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/delete", server.deleteWorkItem)
	mux.HandleFunc("GET /v1/work-item-runs", server.listWorkItemRuns)
	mux.HandleFunc("POST /v1/work-item-runs", server.startWorkItemRun)
	mux.HandleFunc("POST /v1/work-item-runs/{runID}/launch", server.launchWorkItemRun)
	mux.HandleFunc("POST /v1/work-item-runs/{runID}/cancel", server.cancelWorkItemRun)
	mux.HandleFunc("POST /v1/status", server.reportStatus)
	mux.HandleFunc("GET /v1/status-events", server.listStatusEvents)
	mux.HandleFunc("POST /v1/status-events/{statusEventID}/read", server.markStatusEventRead)
	if opts.controlToken != "" {
		return RequireBearerAuth(opts.controlToken, mux)
	}
	return mux
}

func (s *HTTPServer) onboardingStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.OnboardingStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.OnboardingStatus(status))
}

func (s *HTTPServer) applyOnboarding(w http.ResponseWriter, r *http.Request) {
	var req protocol.OnboardingApplyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := s.runtime.ApplyOnboarding(r.Context(), app.OnboardingApplyRequest{ItemIDs: req.ItemIDs})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.OnboardingStatus(status))
}

func (s *HTTPServer) clearDaemon(w http.ResponseWriter, r *http.Request) {
	var req protocol.ClearDaemonRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	cleared, err := s.runtime.ClearDaemon(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.ClearDaemonResponse{
		SessionsCleared:  cleared.SessionsCleared,
		PTYsCleared:      cleared.PTYsCleared,
		ProjectsCleared:  cleared.ProjectsCleared,
		WorkItemsCleared: cleared.WorkItemsCleared,
		ForwardsCleared:  cleared.ForwardsCleared,
	})
}

func (s *HTTPServer) agentBridgeHook(w http.ResponseWriter, r *http.Request) {
	var req protocol.AgentBridgeHookRequest
	if !s.decodeAgentHookJSON(w, r, &req) {
		return
	}
	bridgeID := pathValue(r, "bridgeID", "")
	resp, err := s.runtime.HandleAgentBridgeHook(r.Context(), app.AgentBridgeHookRequest{
		BridgeID:         bridgeID,
		Token:            req.Token,
		Provider:         req.Provider,
		EventName:        req.EventName,
		HookProtocol:     req.HookProtocol,
		ToolName:         req.ToolName,
		ToolInput:        req.ToolInput,
		ToolOutput:       req.ToolOutput,
		Message:          req.Message,
		NotificationType: req.NotificationType,
		ElicitationID:    req.ElicitationID,
		Action:           req.Action,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		RawPayload:       req.RawPayload,
		Decision: app.AgentBridgeHookDecision{
			Action: req.Decision.Action,
			Reason: req.Decision.Reason,
		},
	})
	if err != nil {
		if errors.Is(err, app.ErrUnauthorizedAgentBridgeHook) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.AgentBridgeHookResponse{Output: resp.Output})
}

func (s *HTTPServer) recordAgentHookEvent(w http.ResponseWriter, r *http.Request) {
	var req protocol.AgentBridgeHookRequest
	if !s.decodeAgentHookJSON(w, r, &req) {
		return
	}
	event, err := s.runtime.RecordAgentHookEvent(r.Context(), app.AgentBridgeHookRequest{
		Provider:         req.Provider,
		EventName:        req.EventName,
		HookProtocol:     req.HookProtocol,
		ToolName:         req.ToolName,
		ToolInput:        req.ToolInput,
		ToolOutput:       req.ToolOutput,
		Message:          req.Message,
		NotificationType: req.NotificationType,
		ElicitationID:    req.ElicitationID,
		Action:           req.Action,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		RawPayload:       req.RawPayload,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProtocolAgentBridgeEvent(event))
}

func (s *HTTPServer) listAgentBridgeApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := s.runtime.ListAgentBridgeApprovals(r.Context(), app.ListAgentBridgeApprovalsRequest{
		Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := make([]protocol.AgentBridgeApproval, 0, len(approvals))
	for _, approval := range approvals {
		out = append(out, toProtocolAgentBridgeApproval(approval))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) resolveAgentBridgeApproval(w http.ResponseWriter, r *http.Request) {
	var req protocol.ResolveAgentBridgeApprovalRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	approval, err := s.runtime.ResolveAgentBridgeApproval(r.Context(), app.ResolveAgentBridgeApprovalRequest{
		ID:     pathValue(r, "approvalID", ""),
		Action: req.Action,
		Reason: req.Reason,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentBridgeApproval(approval))
}

func (s *HTTPServer) listAgentPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := s.runtime.ListAgentPrompts(r.Context(), app.ListAgentPromptsRequest{
		Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := make([]protocol.AgentPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		out = append(out, toProtocolAgentPrompt(prompt))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) resolveAgentPrompt(w http.ResponseWriter, r *http.Request) {
	var req protocol.ResolveAgentPromptRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	prompt, err := s.runtime.ResolveAgentPrompt(r.Context(), app.ResolveAgentPromptRequest{
		ID:     pathValue(r, "promptID", ""),
		Answer: req.Answer,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentPrompt(prompt))
}

func (s *HTTPServer) listAgentBridgeEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.runtime.ListAgentBridgeEvents(r.Context(), app.ListAgentBridgeEventsRequest{
		Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := make([]protocol.AgentBridgeEvent, 0, len(events))
	for _, event := range events {
		out = append(out, toProtocolAgentBridgeEvent(event))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) markAgentBridgeEventRead(w http.ResponseWriter, r *http.Request) {
	var req protocol.MarkAgentBridgeEventReadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = pathValue(r, "eventID", req.ID)
	event, err := s.runtime.MarkAgentBridgeEventRead(r.Context(), app.MarkAgentBridgeEventReadRequest{ID: req.ID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentBridgeEvent(event))
}

func (s *HTTPServer) listAgentHookIntegrations(w http.ResponseWriter, r *http.Request) {
	integrations, err := s.runtime.ListAgentHookIntegrations(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := make([]protocol.AgentHookIntegration, 0, len(integrations))
	for _, integration := range integrations {
		out = append(out, toProtocolAgentHookIntegration(integration))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) checkAgentHookIntegration(w http.ResponseWriter, r *http.Request) {
	var req protocol.AgentHookIntegrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	integration, err := s.runtime.CheckAgentHookIntegration(r.Context(), app.AgentHookIntegrationRequest{Provider: req.Provider})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookIntegration(integration))
}

func (s *HTTPServer) installAgentHookIntegration(w http.ResponseWriter, r *http.Request) {
	var req protocol.AgentHookIntegrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	integration, err := s.runtime.InstallAgentHookIntegration(r.Context(), app.AgentHookIntegrationRequest{Provider: req.Provider})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookIntegration(integration))
}

func (s *HTTPServer) removeAgentHookIntegration(w http.ResponseWriter, r *http.Request) {
	var req protocol.AgentHookIntegrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	integration, err := s.runtime.RemoveAgentHookIntegration(r.Context(), app.AgentHookIntegrationRequest{Provider: req.Provider})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookIntegration(integration))
}

func (s *HTTPServer) agentHookLogStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.AgentHookLogStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookLogStatus(status))
}

func (s *HTTPServer) setAgentHookLogSettings(w http.ResponseWriter, r *http.Request) {
	var req protocol.SetAgentHookLogSettingsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := s.runtime.SetAgentHookLogSettings(r.Context(), app.SetAgentHookLogSettingsRequest{
		Enabled:           req.Enabled,
		ClearAfterSession: req.ClearAfterSession,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookLogStatus(status))
}

func (s *HTTPServer) clearAgentHookLog(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.ClearAgentHookLog(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookLogStatus(status))
}

func (s *HTTPServer) openAgentHookLog(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.OpenAgentHookLog(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolAgentHookLogStatus(status))
}

func toProtocolAgentBridgeApproval(approval agentbridge.Approval) protocol.AgentBridgeApproval {
	return protocol.AgentBridgeApproval{
		ID:        approval.ID,
		BridgeID:  approval.BridgeID,
		SessionID: approval.SessionID,
		PTYID:     approval.PTYID,
		RunID:     approval.RunID,
		Provider:  string(approval.Provider),
		EventName: approval.EventName,
		ToolName:  approval.ToolName,
		ToolInput: approval.ToolInput,
		Status:    string(approval.Status),
		Decision: protocol.AgentBridgeHookDecision{
			Action: string(approval.Decision.Action),
			Reason: approval.Decision.Reason,
		},
		CreatedAt:  approval.CreatedAt,
		ResolvedAt: approval.ResolvedAt,
	}
}

func toProtocolAgentBridgeEvent(event agentbridge.Event) protocol.AgentBridgeEvent {
	normalized := agentbridge.NormalizeEvent(event)
	options := make([]protocol.AgentBridgeEventOption, 0, len(normalized.Options))
	for _, option := range normalized.Options {
		options = append(options, protocol.AgentBridgeEventOption{Label: option.Label, Value: option.Value})
	}
	return protocol.AgentBridgeEvent{
		ID:                event.ID,
		BridgeID:          event.BridgeID,
		Kind:              string(normalized.Kind),
		Title:             normalized.Title,
		SessionID:         normalized.SessionID,
		ProviderSessionID: normalized.ProviderSessionID,
		PTYID:             normalized.PTYID,
		CWD:               normalized.CWD,
		Agent:             normalized.Agent,
		Provider:          string(event.Provider),
		EventName:         event.EventName,
		ToolName:          event.ToolName,
		Message:           event.Message,
		NotificationType:  event.NotificationType,
		ElicitationID:     event.ElicitationID,
		Action:            event.Action,
		Result:            event.Result,
		Options:           options,
		Answerable:        normalized.Answerable,
		Status:            string(event.Status),
		CreatedAt:         event.CreatedAt,
		Raw:               event.Raw,
	}
}

func toProtocolAgentPrompt(prompt agentbridge.Prompt) protocol.AgentPrompt {
	options := make([]protocol.AgentBridgeEventOption, 0, len(prompt.Options))
	for _, option := range prompt.Options {
		options = append(options, protocol.AgentBridgeEventOption{Label: option.Label, Value: option.Value})
	}
	return protocol.AgentPrompt{
		ID:            prompt.ID,
		BridgeID:      prompt.BridgeID,
		SessionID:     prompt.SessionID,
		PTYID:         prompt.PTYID,
		RunID:         prompt.RunID,
		Provider:      string(prompt.Provider),
		Kind:          string(prompt.Kind),
		EventName:     prompt.EventName,
		ToolName:      prompt.ToolName,
		ToolInput:     prompt.ToolInput,
		Message:       prompt.Message,
		CWD:           prompt.CWD,
		ElicitationID: prompt.ElicitationID,
		Options:       options,
		Status:        string(prompt.Status),
		Answer:        prompt.Answer,
		CreatedAt:     prompt.CreatedAt,
		ResolvedAt:    prompt.ResolvedAt,
	}
}

func toProtocolAgentHookLogStatus(status app.AgentHookLogStatus) protocol.AgentHookLogStatus {
	return protocol.AgentHookLogStatus{
		Enabled:           status.Enabled,
		ClearAfterSession: status.ClearAfterSession,
		Path:              status.Path,
		SizeBytes:         status.SizeBytes,
	}
}

func toProtocolAgentHookIntegration(integration app.AgentHookIntegration) protocol.AgentHookIntegration {
	return protocol.AgentHookIntegration{
		Provider:         integration.Provider,
		State:            integration.State,
		Status:           integration.Status,
		InstalledVersion: integration.InstalledVersion,
		LatestVersion:    integration.LatestVersion,
		HelperPath:       integration.HelperPath,
		ConfigPath:       integration.ConfigPath,
		ManifestPath:     integration.ManifestPath,
		Detail:           integration.Detail,
	}
}

func (s *HTTPServer) closeSession(w http.ResponseWriter, r *http.Request) {
	req := protocol.CloseSessionRequest{SessionID: pathValue(r, "sessionID", "")}
	sessions, err := s.runtime.CloseSession(r.Context(), app.CloseSessionRequest{SessionID: req.SessionID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *HTTPServer) restartPanePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.RestartPanePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	req.PaneID = pathValue(r, "paneID", req.PaneID)
	restarted, err := s.runtime.RestartPanePTY(r.Context(), app.RestartPanePTYRequest{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
		Options:   *toAppStartPTYOptions(&req.Options),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.RestartedPanePTY{
		Session:  restarted.Session,
		PTYID:    restarted.PTYID,
		OldPTYID: restarted.OldPTYID,
	})
}

func (s *HTTPServer) killPTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.KillPTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PTYID = pathValue(r, "ptyID", req.PTYID)
	killed, err := s.runtime.KillPTY(r.Context(), app.KillPTYRequest{PTYID: req.PTYID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolPTYInfo(killed))
}

func (s *HTTPServer) deletePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.DeletePTYRequest
	req.PTYID = pathValue(r, "ptyID", req.PTYID)
	if err := s.runtime.DeletePTY(r.Context(), app.DeletePTYRequest{PTYID: req.PTYID}); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toProtocolPTYInfo(pty app.PTYInfo) protocol.PTYInfo {
	return protocol.PTYInfo{
		ID:             pty.ID,
		WorkingDir:     pty.WorkingDir,
		Cols:           pty.Cols,
		Rows:           pty.Rows,
		Running:        pty.Running,
		Status:         string(pty.Status),
		ExitCode:       pty.ExitCode,
		SessionID:      pty.SessionID,
		WindowID:       pty.WindowID,
		PaneID:         pty.PaneID,
		OriginWindowID: pty.OriginWindowID,
		OriginPaneID:   pty.OriginPaneID,
	}
}

func toProtocolPTYHistorySummary(history app.PTYHistorySummary) protocol.PTYHistorySummary {
	return protocol.PTYHistorySummary{
		PTYID:      history.PTYID,
		SessionID:  history.SessionID,
		WindowID:   history.WindowID,
		PaneID:     history.PaneID,
		WorkingDir: history.WorkingDir,
		CreatedAt:  history.CreatedAt,
		ExitCode:   history.ExitCode,
	}
}

func toProtocolPTYHistory(history app.PTYHistory) protocol.PTYHistory {
	return protocol.PTYHistory{
		PTYID:      history.PTYID,
		SessionID:  history.SessionID,
		WindowID:   history.WindowID,
		PaneID:     history.PaneID,
		WorkingDir: history.WorkingDir,
		CreatedAt:  history.CreatedAt,
		ExitCode:   history.ExitCode,
		Output:     history.Output,
	}
}

func (s *HTTPServer) setSessionRootDir(w http.ResponseWriter, r *http.Request) {
	var req protocol.SetSessionRootDirRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	updated, err := s.runtime.SetSessionRootDir(r.Context(), app.SetSessionRootDirRequest{
		SessionID: req.SessionID,
		RootDir:   req.RootDir,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *HTTPServer) setSessionProject(w http.ResponseWriter, r *http.Request) {
	var req protocol.SetSessionProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	updated, err := s.runtime.SetSessionProject(r.Context(), app.SetSessionProjectRequest{
		SessionID: req.SessionID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *HTTPServer) setPaneWorkingDir(w http.ResponseWriter, r *http.Request) {
	var req protocol.SetPaneWorkingDirRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	req.PaneID = pathValue(r, "paneID", req.PaneID)
	updated, err := s.runtime.SetPaneWorkingDir(r.Context(), app.SetPaneWorkingDirRequest{
		SessionID:  req.SessionID,
		PaneID:     req.PaneID,
		WorkingDir: req.WorkingDir,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *HTTPServer) startPanePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.StartPanePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	req.PaneID = pathValue(r, "paneID", req.PaneID)
	started, err := s.runtime.StartPanePTY(r.Context(), app.StartPanePTYRequest{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
		Options:   *toAppStartPTYOptions(&req.Options),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.StartedPanePTY{
		Session: started.Session,
		PTYID:   started.PTYID,
	})
}

func (s *HTTPServer) detachPanePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.DetachPanePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	req.PaneID = pathValue(r, "paneID", req.PaneID)
	detached, err := s.runtime.DetachPanePTY(r.Context(), app.DetachPanePTYRequest{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.DetachedPanePTY{
		Session: detached.Session,
		PTYID:   detached.PTYID,
	})
}

func (s *HTTPServer) closePane(w http.ResponseWriter, r *http.Request) {
	var req protocol.ClosePaneRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	req.WindowID = pathValue(r, "windowID", req.WindowID)
	req.PaneID = pathValue(r, "paneID", req.PaneID)
	updated, err := s.runtime.ClosePane(r.Context(), app.ClosePaneRequest{
		SessionID: req.SessionID,
		WindowID:  req.WindowID,
		PaneID:    req.PaneID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type HTTPServer struct {
	runtime                  *app.Runtime
	agentHookRequestMaxBytes int64
	agentHookReadTimeout     time.Duration
	ptyOutputBatchInterval   time.Duration
}

func (s *HTTPServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *HTTPServer) compatibility(w http.ResponseWriter, _ *http.Request) {
	build := buildinfo.Current()
	writeJSON(w, http.StatusOK, protocol.CompatibilityResponse{
		APIVersion:                        protocol.DaemonAPIVersion,
		ProtocolVersion:                   protocol.ProtocolVersion,
		SupportedPreviousProtocolVersions: append([]int(nil), protocol.SupportedPreviousProtocolVersions...),
		GitSHA:                            build.GitSHA,
		Version:                           build.Version,
		Dirty:                             build.Dirty,
	})
}

func (s *HTTPServer) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.runtime.ListSessions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *HTTPServer) createSession(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	created, err := s.runtime.CreateSession(r.Context(), app.CreateSessionRequest{
		Name:       req.Name,
		RootDir:    req.RootDir,
		WorkingDir: req.WorkingDir,
		ProjectID:  req.ProjectID,
		InitialPTY: toAppStartPTYOptions(req.InitialPTY),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.CreatedSession{
		Session:   created.Session,
		WindowID:  created.WindowID,
		PaneID:    created.PaneID,
		PTYID:     created.PTYID,
		MainPtyID: created.MainPtyID,
	})
}

func (s *HTTPServer) splitPane(w http.ResponseWriter, r *http.Request) {
	var req protocol.SplitPaneRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.SessionID = pathValue(r, "sessionID", req.SessionID)
	direction, err := parseDirection(req.Direction)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.runtime.SplitPane(r.Context(), app.SplitPaneRequest{
		SessionID:    req.SessionID,
		WindowID:     req.WindowID,
		TargetPaneID: req.TargetPaneID,
		Direction:    direction,
		InitialPTY:   toAppStartPTYOptions(req.InitialPTY),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.SplitPaneResult{
		Session: result.Session,
		PaneID:  result.PaneID,
		PTYID:   result.PTYID,
		PtyID:   result.PtyID,
	})
}

func (s *HTTPServer) listPTYs(w http.ResponseWriter, r *http.Request) {
	ptys, err := s.runtime.ListPTYs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.PTYInfo, 0, len(ptys))
	for _, pty := range ptys {
		out = append(out, toProtocolPTYInfo(pty))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) listPTYHistory(w http.ResponseWriter, r *http.Request) {
	history, err := s.runtime.ListPTYHistory(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.PTYHistorySummary, 0, len(history))
	for _, item := range history {
		out = append(out, toProtocolPTYHistorySummary(item))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) readPTYHistory(w http.ResponseWriter, r *http.Request) {
	history, err := s.runtime.ReadPTYHistory(r.Context(), r.PathValue("ptyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolPTYHistory(history))
}

func toAppStartPTYOptions(options *protocol.StartPTYOptions) *app.StartPTYOptions {
	if options == nil {
		return nil
	}
	return &app.StartPTYOptions{
		Cols:        options.Cols,
		Rows:        options.Rows,
		Command:     options.Command,
		Env:         options.Env,
		Args:        options.Args,
		Exec:        options.Exec,
		AgentBridge: toAppStartPTYAgentBridgeOptions(options.AgentBridge),
	}
}

func toAppStartPTYAgentBridgeOptions(options *protocol.StartPTYAgentBridgeOptions) *app.StartPTYAgentBridgeOptions {
	if options == nil {
		return nil
	}
	return &app.StartPTYAgentBridgeOptions{
		Enabled:  options.Enabled,
		Provider: options.Provider,
	}
}

func (s *HTTPServer) writePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.WritePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PtyID = pathValue(r, "ptyID", req.PtyID)
	writePTYInputTrace("daemon.http", req.PtyID, len(req.Data))
	if err := s.runtime.WritePTY(r.Context(), req.PtyID, []byte(req.Data)); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) resizePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.ResizePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PtyID = pathValue(r, "ptyID", req.PtyID)
	if err := s.runtime.ResizePTY(r.Context(), req.PtyID, app.PTYSize{Cols: req.Cols, Rows: req.Rows}); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) output(w http.ResponseWriter, r *http.Request) {
	ptyID := r.PathValue("ptyID")
	fromOffset, err := strconv.ParseUint(r.URL.Query().Get("from"), 10, 64)
	if err != nil && r.URL.Query().Get("from") != "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid from offset"))
		return
	}
	snapshot, err := s.runtime.PTYOutput(r.Context(), ptyID, fromOffset)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.OutputSnapshot{
		PtyID:        snapshot.Record.ID,
		Offset:       snapshot.Offset + uint64(len(snapshot.OutputBytes)),
		Output:       string(snapshot.OutputBytes),
		OutputBase64: base64.StdEncoding.EncodeToString(snapshot.OutputBytes),
	})
}

func (s *HTTPServer) attachPTY(w http.ResponseWriter, r *http.Request) {
	ptyID := r.PathValue("ptyID")
	fromOffset, err := strconv.ParseUint(r.URL.Query().Get("from"), 10, 64)
	if err != nil && r.URL.Query().Get("from") != "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid from offset"))
		return
	}
	binaryOutput := r.URL.Query().Get("binary") == "1"
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"wails://*",
			"http://localhost:*",
			"http://127.0.0.1:*",
			"http://[::1]:*",
		},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	attach, err := s.runtime.AttachPTY(ctx, app.AttachPTYRequest{
		PtyID:            ptyID,
		ReplayFromOffset: fromOffset,
	})
	if err != nil {
		_ = writePTYStreamFrame(ctx, conn, protocol.PTYStreamFrame{Type: "error", PtyID: ptyID, Message: err.Error()})
		return
	}
	defer attach.Close()
	go s.readPTYStreamInput(ctx, cancel, conn, ptyID)

	if len(attach.ReplayBytes) > 0 {
		if err := writePTYOutputFrame(ctx, conn, ptyID, attach.ReplayOffset, attach.ReplayBytes, binaryOutput); err != nil {
			return
		}
	}
	nextOffset := attach.ReplayOffset + uint64(len(attach.ReplayBytes))
	batcher := newPTYOutputBatcher(func(ctx context.Context, segment ptyOutputSegment) error {
		return writePTYOutputFrame(ctx, conn, ptyID, segment.offset, segment.bytes, binaryOutput)
	})
	flushTicker := time.NewTicker(s.ptyOutputBatchInterval)
	defer flushTicker.Stop()
	for {
		select {
		case event, ok := <-attach.Events:
			if !ok {
				return
			}
			switch event.Kind {
			case app.PTYOutput:
				segments, updatedOffset, err := s.ptyOutputSegmentsForEvent(ctx, ptyID, nextOffset, event)
				if err != nil {
					return
				}
				nextOffset = updatedOffset
				options := ptyOutputBatchOptions{}
				if s.runtime.PTYInputRecent(ptyID, ptyRecentInputWindow) {
					options.FlushImmediately = true
					options.FlushMaxBytes = ptyRecentInputFlushMaxBytes
				}
				for _, segment := range segments {
					if err := batcher.Enqueue(ctx, segment, options); err != nil {
						return
					}
				}
			case app.PTYExit:
				if err := batcher.Flush(ctx); err != nil {
					return
				}
				_ = writePTYStreamFrame(ctx, conn, protocol.PTYStreamFrame{Type: "exit", PtyID: ptyID, Code: event.Code})
				return
			}
		case <-flushTicker.C:
			if err := batcher.Flush(ctx); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *HTTPServer) ptyOutputSegmentsForEvent(ctx context.Context, ptyID string, nextOffset uint64, event app.PTYEvent) ([]ptyOutputSegment, uint64, error) {
	segments := make([]ptyOutputSegment, 0, 2)
	if event.Offset != nextOffset {
		snapshot, err := s.runtime.PTYOutput(ctx, ptyID, nextOffset)
		if err != nil {
			return nil, nextOffset, err
		}
		if len(snapshot.OutputBytes) > 0 {
			segments = append(segments, ptyOutputSegment{
				offset: snapshot.Offset,
				bytes:  snapshot.OutputBytes,
			})
			nextOffset = snapshot.Offset + uint64(len(snapshot.OutputBytes))
		} else {
			nextOffset = snapshot.Offset
		}
	}
	if event.Offset < nextOffset {
		overlap := nextOffset - event.Offset
		if overlap >= uint64(len(event.Bytes)) {
			return segments, nextOffset, nil
		}
		event.Bytes = event.Bytes[overlap:]
		event.Offset = nextOffset
	}
	if len(event.Bytes) == 0 {
		return segments, nextOffset, nil
	}
	segments = append(segments, ptyOutputSegment{
		offset: event.Offset,
		bytes:  event.Bytes,
	})
	return segments, event.Offset + uint64(len(event.Bytes)), nil
}

func (s *HTTPServer) readPTYStreamInput(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, ptyID string) {
	defer cancel()
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			return
		}
		var frame protocol.PTYStreamFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			return
		}
		if frame.Type != "input" {
			continue
		}
		if frame.PtyID != "" && frame.PtyID != ptyID {
			return
		}
		writePTYInputTrace("daemon.websocket", ptyID, len(frame.Data))
		if err := s.runtime.WritePTY(ctx, ptyID, []byte(frame.Data)); err != nil {
			return
		}
	}
}

func ptyInputTraceLine(channel string, ptyID string, bytes int, at time.Time) string {
	return ptytrace.Line(channel, ptyID, bytes, at)
}

func writePTYInputTrace(channel string, ptyID string, bytes int) {
	ptytrace.Write(ptyInputTraceLine(channel, ptyID, bytes, time.Now()))
}

func writePTYOutputFrame(ctx context.Context, conn *websocket.Conn, ptyID string, offset uint64, output []byte, binaryOutput bool) error {
	if binaryOutput {
		return writePTYBinaryOutputFrame(ctx, conn, offset, output)
	}
	return writePTYTextOutputFrame(ctx, conn, ptyID, offset, output)
}

func writePTYTextOutputFrame(ctx context.Context, conn *websocket.Conn, ptyID string, offset uint64, output []byte) error {
	return writePTYStreamFrame(ctx, conn, protocol.PTYStreamFrame{
		Type:         "output",
		PtyID:        ptyID,
		Offset:       offset,
		OutputBase64: base64.StdEncoding.EncodeToString(output),
	})
}

func writePTYBinaryOutputFrame(ctx context.Context, conn *websocket.Conn, offset uint64, output []byte) error {
	return conn.Write(ctx, websocket.MessageBinary, protocol.EncodePTYBinaryOutputFrame(offset, output))
}

func writePTYStreamFrame(ctx context.Context, conn *websocket.Conn, frame protocol.PTYStreamFrame) error {
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func (s *HTTPServer) nextEvent(w http.ResponseWriter, r *http.Request) {
	timeoutMs := 30_000
	if raw := r.URL.Query().Get("timeoutMs"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid timeoutMs"))
			return
		}
		timeoutMs = parsed
	}
	var afterSeq uint64
	if raw := r.URL.Query().Get("afterSeq"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid afterSeq"))
			return
		}
		afterSeq = parsed
	}
	ctx := r.Context()
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}
	event, err := s.runtime.NextEvent(ctx, afterSeq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusOK, protocol.NextEventResponse{Event: protocol.RuntimeEvent{Type: protocol.RuntimeEventNone}})
			return
		}
		writeError(w, http.StatusRequestTimeout, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.NextEventResponse{
		Event: protocol.RuntimeEvent{
			Type:   string(event.Event.Type),
			Seq:    event.Event.Seq,
			PtyID:  event.Event.PtyID,
			Offset: event.Event.Offset,
		},
		Missed: event.Missed,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func (s *HTTPServer) decodeAgentHookJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	return decodeJSONWithLimits(w, r, out, s.agentHookRequestMaxBytes, s.agentHookReadTimeout)
}

func decodeJSONWithLimits(w http.ResponseWriter, r *http.Request, out any, maxBytes int64, readTimeout time.Duration) bool {
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	}
	if readTimeout <= 0 {
		return decodeJSON(w, r, out)
	}
	ctx, cancel := context.WithTimeout(r.Context(), readTimeout)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(out); err != nil {
			errCh <- err
			return
		}
		var extra any
		err := decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			errCh <- nil
			return
		}
		if err == nil {
			errCh <- fmt.Errorf("request body must contain only one JSON value")
			return
		}
		errCh <- err
	}()
	select {
	case err := <-errCh:
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("request body too large"))
				return false
			}
			writeError(w, http.StatusBadRequest, err)
			return false
		}
		return true
	case <-ctx.Done():
		_ = r.Body.Close()
		writeError(w, http.StatusRequestTimeout, ctx.Err())
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, protocol.ErrorResponse{Error: err.Error()})
}

func pathValue(r *http.Request, name string, fallback string) string {
	value := r.PathValue(name)
	if value == "" {
		return fallback
	}
	return value
}

func parseDirection(value string) (session.SplitDirection, error) {
	switch strings.ToLower(value) {
	case "", "horizontal":
		return session.SplitHorizontal, nil
	case "vertical":
		return session.SplitVertical, nil
	default:
		return "", fmt.Errorf("unknown split direction %q", value)
	}
}
