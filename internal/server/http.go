package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/buildinfo"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

func NewHTTP(runtime *app.Runtime) http.Handler {
	server := &HTTPServer{runtime: runtime}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", server.health)
	mux.HandleFunc("GET /v1/compat", server.compatibility)
	mux.HandleFunc("GET /v1/sessions", server.listSessions)
	mux.HandleFunc("POST /v1/sessions", server.createSession)
	mux.HandleFunc("DELETE /v1/sessions/{sessionID}", server.closeSession)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/split", server.splitPane)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/set-root-dir", server.setSessionRootDir)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/set-working-dir", server.setPaneWorkingDir)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/start-pty", server.startPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/restart-pty", server.restartPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/panes/{paneID}/detach-pty", server.detachPanePTY)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/windows/{windowID}/panes/{paneID}/close", server.closePane)
	mux.HandleFunc("GET /v1/ptys", server.listPTYs)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/write", server.writePTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/resize", server.resizePTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/kill", server.killPTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/bookmarks", server.addPTYBookmark)
	mux.HandleFunc("GET /v1/ptys/{ptyID}/bookmarks", server.listPTYBookmarks)
	mux.HandleFunc("DELETE /v1/pty-bookmarks/{bookmarkID}", server.removePTYBookmark)
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
	mux.HandleFunc("GET /v1/projects", server.listProjects)
	mux.HandleFunc("POST /v1/projects", server.createProject)
	mux.HandleFunc("GET /v1/workflow-templates", server.listWorkflowTemplates)
	mux.HandleFunc("GET /v1/prompt-templates", server.listPromptTemplates)
	mux.HandleFunc("GET /v1/work-items", server.listWorkItems)
	mux.HandleFunc("POST /v1/work-items", server.createWorkItem)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/move", server.moveWorkItem)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/bind-worktree", server.bindWorkItemWorktree)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/attachments", server.addWorkItemAttachment)
	mux.HandleFunc("POST /v1/work-items/{workItemID}/delete", server.deleteWorkItem)
	mux.HandleFunc("GET /v1/work-item-runs", server.listWorkItemRuns)
	mux.HandleFunc("POST /v1/work-item-runs", server.startWorkItemRun)
	mux.HandleFunc("POST /v1/work-item-runs/{runID}/cancel", server.cancelWorkItemRun)
	mux.HandleFunc("POST /v1/status", server.reportStatus)
	mux.HandleFunc("GET /v1/status-events", server.listStatusEvents)
	mux.HandleFunc("POST /v1/status-events/{statusEventID}/read", server.markStatusEventRead)
	return mux
}

func (s *HTTPServer) addPTYBookmark(w http.ResponseWriter, r *http.Request) {
	var req protocol.AddPTYBookmarkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PTYID = pathValue(r, "ptyID", req.PTYID)
	bookmark, err := s.runtime.AddPTYBookmark(r.Context(), app.AddPTYBookmarkRequest{
		PTYID:  req.PTYID,
		Offset: req.Offset,
		Kind:   req.Kind,
		Label:  req.Label,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, bookmark)
}

func (s *HTTPServer) listPTYBookmarks(w http.ResponseWriter, r *http.Request) {
	bookmarks, err := s.runtime.ListPTYBookmarks(r.Context(), pathValue(r, "ptyID", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, bookmarks)
}

func (s *HTTPServer) removePTYBookmark(w http.ResponseWriter, r *http.Request) {
	req := app.RemovePTYBookmarkRequest{BookmarkID: pathValue(r, "bookmarkID", "")}
	if err := s.runtime.RemovePTYBookmark(r.Context(), req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	runtime *app.Runtime
}

func (s *HTTPServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *HTTPServer) compatibility(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, protocol.CompatibilityResponse{
		APIVersion: protocol.DaemonAPIVersion,
		GitSHA:     buildinfo.GitSHA(),
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

func toAppStartPTYOptions(options *protocol.StartPTYOptions) *app.StartPTYOptions {
	if options == nil {
		return nil
	}
	return &app.StartPTYOptions{Cols: options.Cols, Rows: options.Rows, Command: options.Command}
}

func (s *HTTPServer) writePTY(w http.ResponseWriter, r *http.Request) {
	var req protocol.WritePTYRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PtyID = pathValue(r, "ptyID", req.PtyID)
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
	ctx := r.Context()
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}
	event, err := s.runtime.NextEvent(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusOK, protocol.RuntimeEvent{Type: protocol.RuntimeEventNone})
			return
		}
		writeError(w, http.StatusRequestTimeout, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.RuntimeEvent{
		Type:   string(event.Type),
		PtyID:  event.PtyID,
		Offset: event.Offset,
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
