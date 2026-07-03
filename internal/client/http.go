package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/phin-tech/whisk/internal/controlauth"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

type HTTPClient struct {
	baseURL      string
	client       *http.Client
	controlToken string
}

type HTTPOption func(*HTTPClient)

func WithControlToken(token string) HTTPOption {
	return func(c *HTTPClient) {
		c.controlToken = token
	}
}

func NewHTTP(baseURL string, httpClient *http.Client, options ...HTTPOption) *HTTPClient {
	c := &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  httpClient,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// BaseURL returns the daemon URL this client targets, e.g. http://127.0.0.1:8787.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

func (c *HTTPClient) ControlToken() (string, error) {
	if c.controlToken != "" {
		return c.controlToken, nil
	}
	return controlauth.ReadToken()
}

func (c *HTTPClient) AuthorizeRequest(req *http.Request) error {
	if req.URL != nil && req.URL.Path == "/v1/health" {
		return nil
	}
	token, err := c.ControlToken()
	if err != nil {
		if errors.Is(err, controlauth.ErrNoToken) {
			return nil
		}
		return fmt.Errorf("read daemon auth token: %w", err)
	}
	req.Header.Set(controlauth.AuthorizationHeader, controlauth.BearerHeader(token))
	return nil
}

func (c *HTTPClient) Health(ctx context.Context) error {
	var response struct {
		OK bool `json:"ok"`
	}
	if err := c.get(ctx, "/v1/health", nil, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("daemon health check failed")
	}
	return nil
}

func (c *HTTPClient) Compatibility(ctx context.Context) (protocol.CompatibilityResponse, error) {
	var response protocol.CompatibilityResponse
	err := c.get(ctx, "/v1/compat", nil, &response)
	return response, err
}

func (c *HTTPClient) ClearDaemon(ctx context.Context, req protocol.ClearDaemonRequest) (protocol.ClearDaemonResponse, error) {
	var response protocol.ClearDaemonResponse
	err := c.post(ctx, "/v1/daemon/clear", req, &response)
	return response, err
}

func (c *HTTPClient) OnboardingStatus(ctx context.Context) (protocol.OnboardingStatus, error) {
	var status protocol.OnboardingStatus
	err := c.get(ctx, "/v1/onboarding", nil, &status)
	return status, err
}

func (c *HTTPClient) ApplyOnboarding(ctx context.Context, req protocol.OnboardingApplyRequest) (protocol.OnboardingStatus, error) {
	var status protocol.OnboardingStatus
	err := c.post(ctx, "/v1/onboarding/apply", req, &status)
	return status, err
}

func (c *HTTPClient) AgentBridgeHook(ctx context.Context, bridgeID string, req protocol.AgentBridgeHookRequest) (protocol.AgentBridgeHookResponse, error) {
	var response protocol.AgentBridgeHookResponse
	path := "/v1/agent-bridges/" + url.PathEscape(bridgeID) + "/hooks"
	err := c.post(ctx, path, req, &response)
	return response, err
}

func (c *HTTPClient) RecordAgentHookEvent(ctx context.Context, req protocol.AgentBridgeHookRequest) (protocol.AgentBridgeEvent, error) {
	var event protocol.AgentBridgeEvent
	err := c.post(ctx, "/v1/agent-hook-events", req, &event)
	return event, err
}

func (c *HTTPClient) ListAgentBridgeApprovals(ctx context.Context, req protocol.ListAgentBridgeApprovalsRequest) ([]protocol.AgentBridgeApproval, error) {
	query := url.Values{}
	if req.Status != "" {
		query.Set("status", req.Status)
	}
	var approvals []protocol.AgentBridgeApproval
	err := c.get(ctx, "/v1/agent-bridge-approvals", query, &approvals)
	return approvals, err
}

func (c *HTTPClient) ResolveAgentBridgeApproval(ctx context.Context, id string, req protocol.ResolveAgentBridgeApprovalRequest) (protocol.AgentBridgeApproval, error) {
	var approval protocol.AgentBridgeApproval
	path := "/v1/agent-bridge-approvals/" + url.PathEscape(id) + "/resolve"
	err := c.post(ctx, path, req, &approval)
	return approval, err
}

func (c *HTTPClient) ListAgentPrompts(ctx context.Context, req protocol.ListAgentPromptsRequest) ([]protocol.AgentPrompt, error) {
	query := url.Values{}
	if req.Status != "" {
		query.Set("status", req.Status)
	}
	var prompts []protocol.AgentPrompt
	err := c.get(ctx, "/v1/agent-prompts", query, &prompts)
	return prompts, err
}

func (c *HTTPClient) ResolveAgentPrompt(ctx context.Context, id string, req protocol.ResolveAgentPromptRequest) (protocol.AgentPrompt, error) {
	var prompt protocol.AgentPrompt
	path := "/v1/agent-prompts/" + url.PathEscape(id) + "/resolve"
	err := c.post(ctx, path, req, &prompt)
	return prompt, err
}

func (c *HTTPClient) ListAgentBridgeEvents(ctx context.Context, req protocol.ListAgentBridgeEventsRequest) ([]protocol.AgentBridgeEvent, error) {
	query := url.Values{}
	if req.Status != "" {
		query.Set("status", req.Status)
	}
	var events []protocol.AgentBridgeEvent
	err := c.get(ctx, "/v1/agent-bridge-events", query, &events)
	return events, err
}

func (c *HTTPClient) MarkAgentBridgeEventRead(ctx context.Context, req protocol.MarkAgentBridgeEventReadRequest) (protocol.AgentBridgeEvent, error) {
	var event protocol.AgentBridgeEvent
	path := "/v1/agent-bridge-events/" + url.PathEscape(req.ID) + "/read"
	err := c.post(ctx, path, req, &event)
	return event, err
}

func (c *HTTPClient) ListAgentHookIntegrations(ctx context.Context) ([]protocol.AgentHookIntegration, error) {
	var integrations []protocol.AgentHookIntegration
	err := c.get(ctx, "/v1/agent-hook-integrations", nil, &integrations)
	return integrations, err
}

func (c *HTTPClient) CheckAgentHookIntegration(ctx context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	var integration protocol.AgentHookIntegration
	err := c.post(ctx, "/v1/agent-hook-integrations/check", req, &integration)
	return integration, err
}

func (c *HTTPClient) InstallAgentHookIntegration(ctx context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	var integration protocol.AgentHookIntegration
	err := c.post(ctx, "/v1/agent-hook-integrations/install", req, &integration)
	return integration, err
}

func (c *HTTPClient) RemoveAgentHookIntegration(ctx context.Context, req protocol.AgentHookIntegrationRequest) (protocol.AgentHookIntegration, error) {
	var integration protocol.AgentHookIntegration
	err := c.post(ctx, "/v1/agent-hook-integrations/remove", req, &integration)
	return integration, err
}

func (c *HTTPClient) AgentHookLogStatus(ctx context.Context) (protocol.AgentHookLogStatus, error) {
	var status protocol.AgentHookLogStatus
	err := c.get(ctx, "/v1/agent-hook-log", nil, &status)
	return status, err
}

func (c *HTTPClient) SetAgentHookLogSettings(ctx context.Context, req protocol.SetAgentHookLogSettingsRequest) (protocol.AgentHookLogStatus, error) {
	var status protocol.AgentHookLogStatus
	err := c.post(ctx, "/v1/agent-hook-log/settings", req, &status)
	return status, err
}

func (c *HTTPClient) ClearAgentHookLog(ctx context.Context) (protocol.AgentHookLogStatus, error) {
	var status protocol.AgentHookLogStatus
	err := c.post(ctx, "/v1/agent-hook-log/clear", struct{}{}, &status)
	return status, err
}

func (c *HTTPClient) OpenAgentHookLog(ctx context.Context) (protocol.AgentHookLogStatus, error) {
	var status protocol.AgentHookLogStatus
	err := c.post(ctx, "/v1/agent-hook-log/open", struct{}{}, &status)
	return status, err
}

func (c *HTTPClient) ListPlugins(ctx context.Context) ([]protocol.PluginStatus, error) {
	var plugins []protocol.PluginStatus
	err := c.get(ctx, "/v1/plugins", nil, &plugins)
	return plugins, err
}

func (c *HTTPClient) RescanPlugins(ctx context.Context) ([]protocol.PluginStatus, error) {
	var plugins []protocol.PluginStatus
	err := c.post(ctx, "/v1/plugins/rescan", struct{}{}, &plugins)
	return plugins, err
}

func (c *HTTPClient) TrustPlugin(ctx context.Context, id string) (protocol.PluginStatus, error) {
	var status protocol.PluginStatus
	path := "/v1/plugins/" + url.PathEscape(id) + "/trust"
	err := c.post(ctx, path, struct{}{}, &status)
	return status, err
}

func (c *HTTPClient) UntrustPlugin(ctx context.Context, id string) (protocol.PluginStatus, error) {
	var status protocol.PluginStatus
	path := "/v1/plugins/" + url.PathEscape(id) + "/untrust"
	err := c.post(ctx, path, struct{}{}, &status)
	return status, err
}

func (c *HTTPClient) ListRegistryPlugins(ctx context.Context) ([]protocol.RegistryPlugin, error) {
	var plugins []protocol.RegistryPlugin
	err := c.get(ctx, "/v1/plugin-registry", nil, &plugins)
	return plugins, err
}

func (c *HTTPClient) InstallPlugin(ctx context.Context, registry, id string) (protocol.PluginStatus, error) {
	var status protocol.PluginStatus
	err := c.post(ctx, "/v1/plugin-registry/install", protocol.InstallRegistryPluginRequest{Registry: registry, ID: id}, &status)
	return status, err
}

func (c *HTTPClient) RunPluginProjectAttachmentTemplate(ctx context.Context, pluginID string, templateID string, req protocol.RunPluginProjectAttachmentTemplateRequest) (protocol.Project, error) {
	var project protocol.Project
	path := "/v1/plugins/" + url.PathEscape(pluginID) + "/project-attachment-templates/" + url.PathEscape(templateID)
	err := c.post(ctx, path, req, &project)
	return project, err
}

func (c *HTTPClient) ListSessions(ctx context.Context) ([]session.Session, error) {
	var sessions []session.Session
	err := c.get(ctx, "/v1/sessions", nil, &sessions)
	return sessions, err
}

func (c *HTTPClient) CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	var created protocol.CreatedSession
	err := c.post(ctx, "/v1/sessions", req, &created)
	return created, err
}

func (c *HTTPClient) SplitPane(ctx context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	var result protocol.SplitPaneResult
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/split"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) SetSessionRootDir(ctx context.Context, req protocol.SetSessionRootDirRequest) (session.Session, error) {
	var result session.Session
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/set-root-dir"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) SetSessionProject(ctx context.Context, req protocol.SetSessionProjectRequest) (session.Session, error) {
	var result session.Session
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/set-project"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) SetPaneWorkingDir(ctx context.Context, req protocol.SetPaneWorkingDirRequest) (session.Session, error) {
	var result session.Session
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/panes/" + url.PathEscape(req.PaneID) + "/set-working-dir"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) StartPanePTY(ctx context.Context, req protocol.StartPanePTYRequest) (protocol.StartedPanePTY, error) {
	var result protocol.StartedPanePTY
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/panes/" + url.PathEscape(req.PaneID) + "/start-pty"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) RestartPanePTY(ctx context.Context, req protocol.RestartPanePTYRequest) (protocol.RestartedPanePTY, error) {
	var result protocol.RestartedPanePTY
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/panes/" + url.PathEscape(req.PaneID) + "/restart-pty"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) DetachPanePTY(ctx context.Context, req protocol.DetachPanePTYRequest) (protocol.DetachedPanePTY, error) {
	var result protocol.DetachedPanePTY
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/panes/" + url.PathEscape(req.PaneID) + "/detach-pty"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) KillPTY(ctx context.Context, req protocol.KillPTYRequest) (protocol.PTYInfo, error) {
	var result protocol.PTYInfo
	path := "/v1/ptys/" + url.PathEscape(req.PTYID) + "/kill"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) DeletePTY(ctx context.Context, req protocol.DeletePTYRequest) error {
	path := "/v1/ptys/" + url.PathEscape(req.PTYID)
	return c.delete(ctx, path)
}

func (c *HTTPClient) CloseSession(ctx context.Context, req protocol.CloseSessionRequest) ([]session.Session, error) {
	var result []session.Session
	path := "/v1/sessions/" + url.PathEscape(req.SessionID)
	err := c.deleteJSON(ctx, path, &result)
	return result, err
}

func (c *HTTPClient) ClosePane(ctx context.Context, req protocol.ClosePaneRequest) (session.Session, error) {
	var result session.Session
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/windows/" + url.PathEscape(req.WindowID) + "/panes/" + url.PathEscape(req.PaneID) + "/close"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) WritePTY(ctx context.Context, req protocol.WritePTYRequest) error {
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/write"
	return c.post(ctx, path, req, nil)
}

func (c *HTTPClient) ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error {
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/resize"
	return c.post(ctx, path, req, nil)
}

func (c *HTTPClient) Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	query := url.Values{"from": {strconv.FormatUint(req.FromOffset, 10)}}
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/output"
	var snapshot protocol.OutputSnapshot
	err := c.get(ctx, path, query, &snapshot)
	return snapshot, err
}

func (c *HTTPClient) ListPTYs(ctx context.Context) ([]protocol.PTYInfo, error) {
	var ptys []protocol.PTYInfo
	err := c.get(ctx, "/v1/ptys", nil, &ptys)
	return ptys, err
}

func (c *HTTPClient) ListPTYHistory(ctx context.Context) ([]protocol.PTYHistorySummary, error) {
	var history []protocol.PTYHistorySummary
	err := c.get(ctx, "/v1/pty-history", nil, &history)
	return history, err
}

func (c *HTTPClient) ReadPTYHistory(ctx context.Context, ptyID string) (protocol.PTYHistory, error) {
	var history protocol.PTYHistory
	path := "/v1/pty-history/" + url.PathEscape(ptyID)
	err := c.get(ctx, path, nil, &history)
	return history, err
}

func (c *HTTPClient) NextEvent(ctx context.Context, req protocol.NextEventRequest) (protocol.RuntimeEvent, error) {
	query := url.Values{}
	if req.TimeoutMs > 0 {
		query.Set("timeoutMs", strconv.Itoa(req.TimeoutMs))
	}
	var event protocol.RuntimeEvent
	err := c.get(ctx, "/v1/events/next", query, &event)
	return event, err
}

func (c *HTTPClient) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *HTTPClient) post(ctx context.Context, path string, in any, out any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *HTTPClient) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *HTTPClient) deleteJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *HTTPClient) do(req *http.Request, out any) error {
	if err := c.AuthorizeRequest(req); err != nil {
		return err
	}
	httpClient := c.client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		var errorResponse protocol.ErrorResponse
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return errors.New(errorResponse.Error)
		}
		return fmt.Errorf("daemon request failed: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}
	return json.NewDecoder(response.Body).Decode(out)
}
