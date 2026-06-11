package session

import "fmt"

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
	ID    string `json:"id"`
	PtyID string `json:"ptyId"`
}

type Session struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	WorkingDir    string          `json:"workingDir"`
	Layout        LayoutNode      `json:"layout"`
	Panes         map[string]Pane `json:"panes"`
	FocusedPaneID string          `json:"focusedPaneId"`
}

type State struct {
	sessions map[string]Session
}

type CreateSession struct {
	SessionID  string
	PaneID     string
	PtyID      string
	Name       string
	WorkingDir string
}

type SplitPane struct {
	SessionID    string
	TargetPaneID string
	NewPaneID    string
	NewPtyID     string
	Direction    SplitDirection
}

func NewState() *State {
	return &State{sessions: map[string]Session{}}
}

func (s *State) CreateSession(req CreateSession) (Session, error) {
	if req.SessionID == "" {
		return Session{}, fmt.Errorf("session id required")
	}
	if req.PaneID == "" {
		return Session{}, fmt.Errorf("pane id required")
	}
	if req.PtyID == "" {
		return Session{}, fmt.Errorf("pty id required")
	}
	if _, exists := s.sessions[req.SessionID]; exists {
		return Session{}, fmt.Errorf("session %s already exists", req.SessionID)
	}
	name := req.Name
	if name == "" {
		name = "New Session"
	}
	session := Session{
		ID:         req.SessionID,
		Name:       name,
		WorkingDir: req.WorkingDir,
		Layout: LayoutNode{
			Kind:   LayoutLeaf,
			PaneID: req.PaneID,
		},
		Panes: map[string]Pane{
			req.PaneID: {ID: req.PaneID, PtyID: req.PtyID},
		},
		FocusedPaneID: req.PaneID,
	}
	s.sessions[req.SessionID] = session
	return cloneSession(session), nil
}

func (s *State) SplitPane(req SplitPane) (Session, error) {
	current, ok := s.sessions[req.SessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	if _, ok := current.Panes[req.TargetPaneID]; !ok {
		return Session{}, fmt.Errorf("pane %s not found", req.TargetPaneID)
	}
	if req.NewPaneID == "" {
		return Session{}, fmt.Errorf("new pane id required")
	}
	if req.NewPtyID == "" {
		return Session{}, fmt.Errorf("new pty id required")
	}
	if _, exists := current.Panes[req.NewPaneID]; exists {
		return Session{}, fmt.Errorf("pane %s already exists", req.NewPaneID)
	}
	if req.Direction == "" {
		req.Direction = SplitHorizontal
	}
	updatedLayout, changed := insertLeaf(current.Layout, req.TargetPaneID, req.Direction, req.NewPaneID)
	if !changed {
		return Session{}, fmt.Errorf("pane %s not found in layout", req.TargetPaneID)
	}
	current.Layout = updatedLayout
	current.Panes[req.NewPaneID] = Pane{ID: req.NewPaneID, PtyID: req.NewPtyID}
	current.FocusedPaneID = req.NewPaneID
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

func cloneSession(in Session) Session {
	out := in
	out.Layout = cloneLayout(in.Layout)
	out.Panes = make(map[string]Pane, len(in.Panes))
	for id, pane := range in.Panes {
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
