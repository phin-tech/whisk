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
	WritePTY(ctx context.Context, req protocol.WritePTYRequest) error
	ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error
	Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error)
}
