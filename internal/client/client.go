package client

import (
	"context"

	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

type RuntimeClient interface {
	ListSessions(ctx context.Context) ([]session.Session, error)
	CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error)
	SplitPane(ctx context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error)
	SetSessionRootDir(ctx context.Context, req protocol.SetSessionRootDirRequest) (session.Session, error)
	SetPaneWorkingDir(ctx context.Context, req protocol.SetPaneWorkingDirRequest) (session.Session, error)
	StartPanePTY(ctx context.Context, req protocol.StartPanePTYRequest) (protocol.StartedPanePTY, error)
	RestartPanePTY(ctx context.Context, req protocol.RestartPanePTYRequest) (protocol.RestartedPanePTY, error)
	DetachPanePTY(ctx context.Context, req protocol.DetachPanePTYRequest) (protocol.DetachedPanePTY, error)
	CloseSession(ctx context.Context, req protocol.CloseSessionRequest) ([]session.Session, error)
	ClosePane(ctx context.Context, req protocol.ClosePaneRequest) (session.Session, error)
	KillPTY(ctx context.Context, req protocol.KillPTYRequest) (protocol.PTYInfo, error)
	AddPTYBookmark(ctx context.Context, req protocol.AddPTYBookmarkRequest) (protocol.PTYBookmark, error)
	ListPTYBookmarks(ctx context.Context, ptyID string) ([]protocol.PTYBookmark, error)
	RemovePTYBookmark(ctx context.Context, req protocol.RemovePTYBookmarkRequest) error
	WritePTY(ctx context.Context, req protocol.WritePTYRequest) error
	ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error
	Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error)
	ListPTYs(ctx context.Context) ([]protocol.PTYInfo, error)
	NextEvent(ctx context.Context, req protocol.NextEventRequest) (protocol.RuntimeEvent, error)
	DetectWorktrunk(ctx context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error)
	ListWorktrees(ctx context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error)
	CreateWorktree(ctx context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error)
	RemoveWorktree(ctx context.Context, req protocol.RemoveWorktreeRequest) error
	CreateHTTPForward(ctx context.Context, req protocol.CreateHTTPForwardRequest) (protocol.HTTPForward, error)
	ListHTTPForwards(ctx context.Context) ([]protocol.HTTPForward, error)
	DeleteHTTPForward(ctx context.Context, id string) error
}
