package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

func NewHTTP(runtime *app.Runtime) http.Handler {
	server := &HTTPServer{runtime: runtime}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", server.health)
	mux.HandleFunc("GET /v1/sessions", server.listSessions)
	mux.HandleFunc("POST /v1/sessions", server.createSession)
	mux.HandleFunc("POST /v1/sessions/{sessionID}/split", server.splitPane)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/write", server.writePTY)
	mux.HandleFunc("POST /v1/ptys/{ptyID}/resize", server.resizePTY)
	mux.HandleFunc("GET /v1/ptys/{ptyID}/output", server.output)
	return mux
}

type HTTPServer struct {
	runtime *app.Runtime
}

func (s *HTTPServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.CreatedSession{
		Session:   created.Session,
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
		TargetPaneID: req.TargetPaneID,
		Direction:    direction,
		Cols:         req.Cols,
		Rows:         req.Rows,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.SplitPaneResult{
		Session: result.Session,
		PaneID:  result.PaneID,
		PtyID:   result.PtyID,
	})
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
		PtyID:  snapshot.Record.ID,
		Offset: snapshot.Offset + uint64(len(snapshot.OutputBytes)),
		Output: string(snapshot.OutputBytes),
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
