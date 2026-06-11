package wailsapp

import (
	"context"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

type Service struct {
	client client.RuntimeClient
}

func NewService(runtimeClient client.RuntimeClient) *Service {
	return &Service{client: runtimeClient}
}

func (s *Service) ListSessions(ctx context.Context) ([]session.Session, error) {
	return s.client.ListSessions(ctx)
}

func (s *Service) CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	return s.client.CreateSession(ctx, req)
}

func (s *Service) SplitPane(ctx context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	return s.client.SplitPane(ctx, req)
}

func (s *Service) WritePTY(ctx context.Context, req protocol.WritePTYRequest) error {
	return s.client.WritePTY(ctx, req)
}

func (s *Service) ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error {
	return s.client.ResizePTY(ctx, req)
}

func (s *Service) Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	return s.client.Output(ctx, req)
}

func (s *Service) DetectWorktrunk(ctx context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error) {
	return s.client.DetectWorktrunk(ctx, req)
}

func (s *Service) ListWorktrees(ctx context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error) {
	return s.client.ListWorktrees(ctx, req)
}

func (s *Service) CreateWorktree(ctx context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error) {
	return s.client.CreateWorktree(ctx, req)
}

func (s *Service) RemoveWorktree(ctx context.Context, req protocol.RemoveWorktreeRequest) error {
	return s.client.RemoveWorktree(ctx, req)
}
