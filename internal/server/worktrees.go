package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) detectWorktrunk(w http.ResponseWriter, r *http.Request) {
	var req protocol.DetectWorktrunkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := s.runtime.DetectWorktrunk(r.Context(), app.DetectWorktrunkRequest{
		RepoPath:     req.RepoPath,
		OverridePath: req.OverridePath,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolWorktrunkStatus(status))
}

func (s *HTTPServer) listWorktrees(w http.ResponseWriter, r *http.Request) {
	var req protocol.ListWorktreesRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	worktrees, err := s.runtime.ListWorktrees(r.Context(), app.ListWorktreesRequest{
		RepoPath:     req.RepoPath,
		OverridePath: req.OverridePath,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := make([]protocol.Worktree, 0, len(worktrees))
	for _, worktree := range worktrees {
		out = append(out, toProtocolWorktree(worktree))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) createWorktree(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateWorktreeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	created, err := s.runtime.CreateWorktree(r.Context(), app.CreateWorktreeRequest{
		RepoPath:     req.RepoPath,
		Branch:       req.Branch,
		Base:         req.Base,
		OverridePath: req.OverridePath,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocol.CreatedWorktree{Path: created.Path})
}

func (s *HTTPServer) removeWorktree(w http.ResponseWriter, r *http.Request) {
	var req protocol.RemoveWorktreeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.runtime.RemoveWorktree(r.Context(), app.RemoveWorktreeRequest{
		RepoPath:     req.RepoPath,
		WorktreePath: req.WorktreePath,
		AlsoBranch:   req.AlsoBranch,
		Force:        req.Force,
		OverridePath: req.OverridePath,
	}); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toProtocolWorktrunkStatus(status app.WorktrunkStatus) protocol.WorktrunkStatus {
	return protocol.WorktrunkStatus{
		Available:   status.Available,
		ConfigFound: status.ConfigFound,
		Binary: protocol.WorktrunkBinary{
			Path:    status.Binary.Path,
			Version: status.Binary.Version,
		},
	}
}

func toProtocolWorktree(worktree app.Worktree) protocol.Worktree {
	return protocol.Worktree{
		Branch:    worktree.Branch,
		Path:      worktree.Path,
		Kind:      worktree.Kind,
		IsMain:    worktree.IsMain,
		IsCurrent: worktree.IsCurrent,
		Dirty:     worktree.Dirty,
		Locked:    worktree.Locked,
	}
}
