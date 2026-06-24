package worktrunk

import (
	"context"
	"strings"

	"github.com/phin-tech/whisk/internal/app"
)

type BackendOptions struct {
	OverridePath string
}

type Backend struct {
	runner       Runner
	overridePath string
}

func NewBackend(runner Runner) *Backend {
	return NewBackendWithOptions(runner, BackendOptions{})
}

func NewBackendWithOptions(runner Runner, opts BackendOptions) *Backend {
	if runner == nil {
		runner = OSRunner{}
	}
	return &Backend{runner: runner, overridePath: strings.TrimSpace(opts.OverridePath)}
}

func (b *Backend) DetectWorktrunk(ctx context.Context, req app.DetectWorktrunkRequest) (app.WorktrunkStatus, error) {
	binary, available, err := Detect(ctx, b.runner, DetectOptions{OverridePath: firstNonEmpty(req.OverridePath, b.overridePath)})
	if err != nil {
		return app.WorktrunkStatus{}, err
	}
	return app.WorktrunkStatus{
		Available:   available,
		ConfigFound: DetectWTConfig(req.RepoPath),
		Binary:      app.WorktrunkBinary{Path: binary.Path, Version: binary.Version},
	}, nil
}

func (b *Backend) ListWorktrees(ctx context.Context, req app.ListWorktreesRequest) ([]app.Worktree, error) {
	client, err := b.client(ctx, req.OverridePath)
	if err != nil {
		return nil, err
	}
	items, err := client.List(ctx, req.RepoPath)
	if err != nil {
		return nil, err
	}
	worktrees := make([]app.Worktree, 0, len(items))
	for _, item := range items {
		worktrees = append(worktrees, toAppWorktree(item))
	}
	return worktrees, nil
}

func (b *Backend) CreateWorktree(ctx context.Context, req app.CreateWorktreeRequest) (app.CreatedWorktree, error) {
	client, err := b.client(ctx, req.OverridePath)
	if err != nil {
		return app.CreatedWorktree{}, err
	}
	path, err := client.Create(ctx, CreateRequest{
		RepoPath: req.RepoPath,
		Branch:   req.Branch,
		Base:     req.Base,
	})
	if err != nil {
		return app.CreatedWorktree{}, err
	}
	return app.CreatedWorktree{Path: path}, nil
}

func (b *Backend) RemoveWorktree(ctx context.Context, req app.RemoveWorktreeRequest) error {
	client, err := b.client(ctx, req.OverridePath)
	if err != nil {
		return err
	}
	return client.Remove(ctx, RemoveRequest{
		RepoPath:     req.RepoPath,
		WorktreePath: req.WorktreePath,
		AlsoBranch:   req.AlsoBranch,
		Force:        req.Force,
	})
}

func (b *Backend) client(ctx context.Context, overridePath string) (*Client, error) {
	binary, available, err := Detect(ctx, b.runner, DetectOptions{OverridePath: firstNonEmpty(overridePath, b.overridePath)})
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, &NotFoundError{Path: "wt"}
	}
	return NewClient(binary, b.runner), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func toAppWorktree(item Item) app.Worktree {
	return app.Worktree{
		Branch:    item.Branch,
		Path:      item.Path,
		Kind:      item.Kind,
		IsMain:    item.IsMain,
		IsCurrent: item.IsCurrent,
		Dirty:     item.WorkingTree.Dirty,
		Locked:    item.Worktree.Locked,
	}
}
