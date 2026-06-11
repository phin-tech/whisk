package app

import (
	"context"
	"fmt"
)

type WorktreeBackend interface {
	DetectWorktrunk(ctx context.Context, req DetectWorktrunkRequest) (WorktrunkStatus, error)
	ListWorktrees(ctx context.Context, req ListWorktreesRequest) ([]Worktree, error)
	CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (CreatedWorktree, error)
	RemoveWorktree(ctx context.Context, req RemoveWorktreeRequest) error
}

type DetectWorktrunkRequest struct {
	RepoPath     string
	OverridePath string
}

type WorktrunkBinary struct {
	Path    string
	Version string
}

type WorktrunkStatus struct {
	Available   bool
	ConfigFound bool
	Binary      WorktrunkBinary
}

type ListWorktreesRequest struct {
	RepoPath string
}

type Worktree struct {
	Branch    string
	Path      string
	Kind      string
	IsMain    bool
	IsCurrent bool
	Dirty     bool
	Locked    bool
}

type CreateWorktreeRequest struct {
	RepoPath string
	Branch   string
	Base     string
}

type CreatedWorktree struct {
	Path string
}

type RemoveWorktreeRequest struct {
	RepoPath     string
	WorktreePath string
	AlsoBranch   bool
	Force        bool
}

func (r *Runtime) DetectWorktrunk(ctx context.Context, req DetectWorktrunkRequest) (WorktrunkStatus, error) {
	if r.worktrees == nil {
		return WorktrunkStatus{}, fmt.Errorf("worktree backend required")
	}
	return r.worktrees.DetectWorktrunk(ctx, req)
}

func (r *Runtime) ListWorktrees(ctx context.Context, req ListWorktreesRequest) ([]Worktree, error) {
	if r.worktrees == nil {
		return nil, fmt.Errorf("worktree backend required")
	}
	return r.worktrees.ListWorktrees(ctx, req)
}

func (r *Runtime) CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (CreatedWorktree, error) {
	if r.worktrees == nil {
		return CreatedWorktree{}, fmt.Errorf("worktree backend required")
	}
	return r.worktrees.CreateWorktree(ctx, req)
}

func (r *Runtime) RemoveWorktree(ctx context.Context, req RemoveWorktreeRequest) error {
	if r.worktrees == nil {
		return fmt.Errorf("worktree backend required")
	}
	return r.worktrees.RemoveWorktree(ctx, req)
}
