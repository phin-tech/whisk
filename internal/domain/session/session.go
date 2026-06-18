package session

import (
	"fmt"
	"path/filepath"
)

type LayoutKind string

const (
	LayoutLeaf  LayoutKind = "leaf"
	LayoutSplit LayoutKind = "split"
)

type SplitDirection string

const (
	SplitHorizontal SplitDirection = "horizontal"
	SplitVertical   SplitDirection = "vertical"
)

type LayoutNode struct {
	Kind      LayoutKind     `json:"kind"`
	PaneID    string         `json:"paneId,omitempty"`
	Direction SplitDirection `json:"direction,omitempty"`
	Children  []LayoutNode   `json:"children,omitempty"`
	Sizes     []float64      `json:"sizes,omitempty"`
}

type Pane struct {
	ID           string  `json:"id"`
	WindowID     string  `json:"windowId"`
	CurrentPTYID *string `json:"currentPtyId,omitempty"`
	WorkingDir   string  `json:"workingDir"`
}

type SessionWindow struct {
	ID        string     `json:"id"`
	SessionID string     `json:"sessionId"`
	Name      string     `json:"name"`
	Layout    LayoutNode `json:"layout"`
}

type PTYOwner struct {
	SessionID string
	WindowID  string
	PaneID    string
}

type Session struct {
	ID        string                   `json:"id"`
	ProjectID string                   `json:"projectId,omitempty"`
	Name      string                   `json:"name"`
	RootDir   string                   `json:"rootDir"`
	Windows   map[string]SessionWindow `json:"windows"`
	Panes     map[string]Pane          `json:"panes"`
}

type State struct {
	sessions map[string]Session
}

type CreateSession struct {
	SessionID    string
	WindowID     string
	PaneID       string
	InitialPTYID *string
	ProjectID    string
	Name         string
	RootDir      string
	WorkingDir   string
}

type SplitPane struct {
	SessionID    string
	WindowID     string
	TargetPaneID string
	NewPaneID    string
	NewPTYID     *string
	Direction    SplitDirection
}

type SetSessionRootDir struct {
	SessionID string
	RootDir   string
}

type SetSessionProject struct {
	SessionID string
	ProjectID string
}

type SetPaneWorkingDir struct {
	SessionID  string
	PaneID     string
	WorkingDir string
}

type StartPanePTY struct {
	SessionID string
	PaneID    string
	PTYID     string
}

type ClosePane struct {
	SessionID string
	WindowID  string
	PaneID    string
}

type DetachPanePTY struct {
	SessionID string
	PaneID    string
}

type RestartPanePTY struct {
	SessionID string
	PaneID    string
	NewPTYID  string
}

type RemoveSession struct {
	SessionID string
}

func NewState() *State {
	return &State{sessions: map[string]Session{}}
}

func NewStateFromSessions(sessions []Session) (*State, error) {
	state := NewState()
	if err := state.ReplaceSessions(sessions); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) ReplaceSessions(sessions []Session) error {
	next := make(map[string]Session, len(sessions))
	for _, candidate := range sessions {
		normalized, err := validateSession(candidate)
		if err != nil {
			return err
		}
		if _, exists := next[normalized.ID]; exists {
			return fmt.Errorf("session %s already exists", normalized.ID)
		}
		next[normalized.ID] = normalized
	}
	s.sessions = next
	return nil
}

func (s *State) CreateSession(req CreateSession) (Session, error) {
	if req.SessionID == "" {
		return Session{}, fmt.Errorf("session id required")
	}
	if req.WindowID == "" {
		return Session{}, fmt.Errorf("window id required")
	}
	if req.PaneID == "" {
		return Session{}, fmt.Errorf("pane id required")
	}
	rootDir, err := cleanAbsolutePath(req.RootDir, "root dir")
	if err != nil {
		return Session{}, err
	}
	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = rootDir
	}
	workingDir, err = cleanAbsolutePath(workingDir, "working dir")
	if err != nil {
		return Session{}, err
	}
	if _, exists := s.sessions[req.SessionID]; exists {
		return Session{}, fmt.Errorf("session %s already exists", req.SessionID)
	}
	name := req.Name
	if name == "" {
		name = filepath.Base(rootDir)
	}
	if name == "." || name == string(filepath.Separator) {
		name = "New Session"
	}
	session := Session{
		ID:        req.SessionID,
		ProjectID: req.ProjectID,
		Name:      name,
		RootDir:   rootDir,
		Windows: map[string]SessionWindow{
			req.WindowID: {
				ID:        req.WindowID,
				SessionID: req.SessionID,
				Name:      "Main",
				Layout: LayoutNode{
					Kind:   LayoutLeaf,
					PaneID: req.PaneID,
				},
			},
		},
		Panes: map[string]Pane{
			req.PaneID: {
				ID:           req.PaneID,
				WindowID:     req.WindowID,
				CurrentPTYID: cloneStringPtr(req.InitialPTYID),
				WorkingDir:   workingDir,
			},
		},
	}
	s.sessions[req.SessionID] = session
	return cloneSession(session), nil
}

func (s *State) SplitPane(req SplitPane) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	window, ok := current.Windows[req.WindowID]
	if !ok {
		return Session{}, fmt.Errorf("window %s not found", req.WindowID)
	}
	target, ok := current.Panes[req.TargetPaneID]
	if !ok || target.WindowID != req.WindowID {
		return Session{}, fmt.Errorf("pane %s not found", req.TargetPaneID)
	}
	if req.NewPaneID == "" {
		return Session{}, fmt.Errorf("new pane id required")
	}
	if _, exists := current.Panes[req.NewPaneID]; exists {
		return Session{}, fmt.Errorf("pane %s already exists", req.NewPaneID)
	}
	if req.Direction == "" {
		req.Direction = SplitHorizontal
	}
	updatedLayout, changed := insertLeaf(window.Layout, req.TargetPaneID, req.Direction, req.NewPaneID)
	if !changed {
		return Session{}, fmt.Errorf("pane %s not found in layout", req.TargetPaneID)
	}
	window.Layout = updatedLayout
	current.Windows[req.WindowID] = window
	current.Panes[req.NewPaneID] = Pane{
		ID:           req.NewPaneID,
		WindowID:     req.WindowID,
		CurrentPTYID: cloneStringPtr(req.NewPTYID),
		WorkingDir:   target.WorkingDir,
	}
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) SetSessionRootDir(req SetSessionRootDir) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	rootDir, err := cleanAbsolutePath(req.RootDir, "root dir")
	if err != nil {
		return Session{}, err
	}
	oldRoot := current.RootDir
	current.RootDir = rootDir
	for id, pane := range current.Panes {
		if pane.WorkingDir == "" || pane.WorkingDir == oldRoot {
			pane.WorkingDir = rootDir
			current.Panes[id] = pane
		}
	}
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) SetSessionProject(req SetSessionProject) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	current.ProjectID = req.ProjectID
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) SetPaneWorkingDir(req SetPaneWorkingDir) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return Session{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	workingDir, err := cleanAbsolutePath(req.WorkingDir, "working dir")
	if err != nil {
		return Session{}, err
	}
	pane.WorkingDir = workingDir
	current.Panes[req.PaneID] = pane
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) StartPanePTY(req StartPanePTY) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	if req.PTYID == "" {
		return Session{}, fmt.Errorf("pty id required")
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return Session{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID != nil {
		return Session{}, fmt.Errorf("pane %s already has current pty", req.PaneID)
	}
	pane.CurrentPTYID = &req.PTYID
	current.Panes[req.PaneID] = pane
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) DetachPanePTY(req DetachPanePTY) (Session, string, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, "", fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return Session{}, "", fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID == nil {
		return Session{}, "", fmt.Errorf("pane %s has no current pty", req.PaneID)
	}
	detached := *pane.CurrentPTYID
	pane.CurrentPTYID = nil
	current.Panes[req.PaneID] = pane
	s.sessions[req.SessionID] = current
	return cloneSession(current), detached, nil
}

func (s *State) RestartPanePTY(req RestartPanePTY) (Session, string, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, "", fmt.Errorf("session %s not found", req.SessionID)
	}
	if req.NewPTYID == "" {
		return Session{}, "", fmt.Errorf("new pty id required")
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return Session{}, "", fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID == nil {
		return Session{}, "", fmt.Errorf("pane %s has no current pty", req.PaneID)
	}
	oldPTY := *pane.CurrentPTYID
	pane.CurrentPTYID = &req.NewPTYID
	current.Panes[req.PaneID] = pane
	s.sessions[req.SessionID] = current
	return cloneSession(current), oldPTY, nil
}

func (s *State) RemoveSession(req RemoveSession) (Session, error) {
	if req.SessionID == "" {
		return Session{}, fmt.Errorf("session id required")
	}
	removed, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	delete(s.sessions, req.SessionID)
	return cloneSession(removed), nil
}

func (s *State) ClosePane(req ClosePane) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	window, ok := current.Windows[req.WindowID]
	if !ok {
		return Session{}, fmt.Errorf("window %s not found", req.WindowID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok || pane.WindowID != req.WindowID {
		return Session{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID != nil {
		return Session{}, fmt.Errorf("pane %s has current pty", req.PaneID)
	}
	if countWindowPanes(current, req.WindowID) <= 1 {
		return Session{}, fmt.Errorf("cannot close last pane in window")
	}
	nextLayout, removed := removeLeaf(window.Layout, req.PaneID)
	if !removed {
		return Session{}, fmt.Errorf("pane %s not found in layout", req.PaneID)
	}
	window.Layout = nextLayout
	current.Windows[req.WindowID] = window
	delete(current.Panes, req.PaneID)
	s.sessions[req.SessionID] = current
	return cloneSession(current), nil
}

func (s *State) Get(id string) (Session, bool) {
	session, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	return cloneSession(session), true
}

func (s *State) List() []Session {
	out := make([]Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		out = append(out, cloneSession(session))
	}
	return out
}

func (s *State) PTYOwners() map[string]PTYOwner {
	owners := make(map[string]PTYOwner)
	for _, session := range s.sessions {
		for paneID, pane := range session.Panes {
			if pane.CurrentPTYID == nil {
				continue
			}
			owners[*pane.CurrentPTYID] = PTYOwner{
				SessionID: session.ID,
				WindowID:  pane.WindowID,
				PaneID:    paneID,
			}
		}
	}
	return owners
}

func cleanAbsolutePath(value string, label string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s required", label)
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	return filepath.Clean(value), nil
}

func validateSession(candidate Session) (Session, error) {
	if candidate.ID == "" {
		return Session{}, fmt.Errorf("session id required")
	}
	rootDir, err := cleanAbsolutePath(candidate.RootDir, "root dir")
	if err != nil {
		return Session{}, err
	}
	if len(candidate.Windows) == 0 {
		return Session{}, fmt.Errorf("session %s must have a window", candidate.ID)
	}
	if len(candidate.Panes) == 0 {
		return Session{}, fmt.Errorf("session %s must have a pane", candidate.ID)
	}
	candidate.RootDir = rootDir
	for windowID, window := range candidate.Windows {
		if windowID == "" || window.ID == "" || window.ID != windowID {
			return Session{}, fmt.Errorf("window id mismatch")
		}
		if window.SessionID != candidate.ID {
			return Session{}, fmt.Errorf("window %s session mismatch", windowID)
		}
		seen := map[string]struct{}{}
		if err := validateLayout(window.Layout, candidate.Panes, windowID, seen); err != nil {
			return Session{}, err
		}
		candidate.Windows[windowID] = window
	}
	for paneID, pane := range candidate.Panes {
		if paneID == "" || pane.ID == "" || pane.ID != paneID {
			return Session{}, fmt.Errorf("pane id mismatch")
		}
		if _, ok := candidate.Windows[pane.WindowID]; !ok {
			return Session{}, fmt.Errorf("pane %s window %s not found", paneID, pane.WindowID)
		}
		workingDir, err := cleanAbsolutePath(pane.WorkingDir, "working dir")
		if err != nil {
			return Session{}, err
		}
		pane.WorkingDir = workingDir
		pane.CurrentPTYID = cloneStringPtr(pane.CurrentPTYID)
		candidate.Panes[paneID] = pane
	}
	return cloneSession(candidate), nil
}

func validateLayout(node LayoutNode, panes map[string]Pane, windowID string, seen map[string]struct{}) error {
	switch node.Kind {
	case LayoutLeaf:
		if node.PaneID == "" {
			return fmt.Errorf("layout leaf pane id required")
		}
		pane, ok := panes[node.PaneID]
		if !ok {
			return fmt.Errorf("layout pane %s not found", node.PaneID)
		}
		if pane.WindowID != windowID {
			return fmt.Errorf("layout pane %s belongs to window %s", node.PaneID, pane.WindowID)
		}
		if _, exists := seen[node.PaneID]; exists {
			return fmt.Errorf("layout pane %s duplicated", node.PaneID)
		}
		seen[node.PaneID] = struct{}{}
		return nil
	case LayoutSplit:
		if node.Direction != SplitHorizontal && node.Direction != SplitVertical {
			return fmt.Errorf("layout split direction required")
		}
		if len(node.Children) < 2 {
			return fmt.Errorf("layout split requires at least two children")
		}
		if len(node.Sizes) > 0 && len(node.Sizes) != len(node.Children) {
			return fmt.Errorf("layout sizes must match children")
		}
		for _, child := range node.Children {
			if err := validateLayout(child, panes, windowID, seen); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("layout kind required")
	}
}

func insertLeaf(node LayoutNode, targetID string, direction SplitDirection, newPaneID string) (LayoutNode, bool) {
	if node.Kind == LayoutLeaf {
		if node.PaneID != targetID {
			return node, false
		}
		return LayoutNode{
			Kind:      LayoutSplit,
			Direction: direction,
			Children: []LayoutNode{
				{Kind: LayoutLeaf, PaneID: targetID},
				{Kind: LayoutLeaf, PaneID: newPaneID},
			},
		}, true
	}
	next := node
	next.Children = make([]LayoutNode, len(node.Children))
	changed := false
	for i, child := range node.Children {
		updated, childChanged := insertLeaf(child, targetID, direction, newPaneID)
		next.Children[i] = updated
		changed = changed || childChanged
	}
	if !changed {
		return node, false
	}
	if node.Direction != direction {
		next.Sizes = nil
		return next, true
	}
	flat := make([]LayoutNode, 0, len(next.Children)+1)
	for _, child := range next.Children {
		if child.Kind == LayoutSplit && child.Direction == direction {
			flat = append(flat, child.Children...)
			continue
		}
		flat = append(flat, child)
	}
	next.Children = flat
	next.Sizes = nil
	return next, true
}

func removeLeaf(node LayoutNode, paneID string) (LayoutNode, bool) {
	if node.Kind == LayoutLeaf {
		return node, node.PaneID == paneID
	}
	next := node
	children := make([]LayoutNode, 0, len(node.Children))
	removed := false
	for _, child := range node.Children {
		updated, childRemoved := removeLeaf(child, paneID)
		if childRemoved {
			removed = true
			if child.Kind == LayoutLeaf && child.PaneID == paneID {
				continue
			}
			if updated.Kind == "" {
				continue
			}
		}
		children = append(children, updated)
	}
	if !removed {
		return node, false
	}
	if len(children) == 1 {
		return children[0], true
	}
	next.Children = children
	next.Sizes = nil
	return next, true
}

func countWindowPanes(current Session, windowID string) int {
	count := 0
	for _, pane := range current.Panes {
		if pane.WindowID == windowID {
			count++
		}
	}
	return count
}

func cloneSession(in Session) Session {
	out := in
	out.Windows = make(map[string]SessionWindow, len(in.Windows))
	for id, window := range in.Windows {
		window.Layout = cloneLayout(window.Layout)
		out.Windows[id] = window
	}
	out.Panes = make(map[string]Pane, len(in.Panes))
	for id, pane := range in.Panes {
		pane.CurrentPTYID = cloneStringPtr(pane.CurrentPTYID)
		out.Panes[id] = pane
	}
	return out
}

func cloneLayout(in LayoutNode) LayoutNode {
	out := in
	if in.Children != nil {
		out.Children = make([]LayoutNode, len(in.Children))
		for i, child := range in.Children {
			out.Children[i] = cloneLayout(child)
		}
	}
	if in.Sizes != nil {
		out.Sizes = append([]float64(nil), in.Sizes...)
	}
	return out
}

func cloneStringPtr(in *string) *string {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
