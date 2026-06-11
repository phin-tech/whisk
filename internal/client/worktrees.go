package client

import (
	"context"

	"github.com/phin-tech/whisk/internal/protocol"
)

func (c *HTTPClient) DetectWorktrunk(ctx context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error) {
	var status protocol.WorktrunkStatus
	err := c.post(ctx, "/v1/worktrunk/detect", req, &status)
	return status, err
}

func (c *HTTPClient) ListWorktrees(ctx context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error) {
	var worktrees []protocol.Worktree
	err := c.post(ctx, "/v1/worktrees/list", req, &worktrees)
	return worktrees, err
}

func (c *HTTPClient) CreateWorktree(ctx context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error) {
	var created protocol.CreatedWorktree
	err := c.post(ctx, "/v1/worktrees/create", req, &created)
	return created, err
}

func (c *HTTPClient) RemoveWorktree(ctx context.Context, req protocol.RemoveWorktreeRequest) error {
	return c.post(ctx, "/v1/worktrees/remove", req, nil)
}
