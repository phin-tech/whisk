package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/phin-tech/whisk/internal/app"
)

const (
	subjectSessionChanged              = "whisk.session.changed"
	subjectPTYChanged                  = "whisk.pty.changed"
	subjectPTYOutput                   = "whisk.pty.output"
	subjectWorkItemsChanged            = "whisk.workitems.changed"
	subjectStatusChanged               = "whisk.status.changed"
	subjectPluginsChanged              = "whisk.plugins.changed"
	subjectMailboxChanged              = "whisk.mailbox.changed"
	subjectAgentBridgeApprovalsChanged = "whisk.agent_bridge_approvals.changed"
	subjectAgentPromptsChanged         = "whisk.agent_prompts.changed"
	subjectAgentHookEventsChanged      = "whisk.agent_hook_events.changed"

	retainedRuntimeEventLimit = 256
)

type NATSBus struct {
	server   *natsserver.Server
	conn     *nats.Conn
	mu       sync.Mutex
	retained []app.RuntimeEvent
	notify   chan struct{}
}

func NewNATSBus() (*NATSBus, error) {
	server, err := natsserver.NewServer(&natsserver.Options{
		Host:   "127.0.0.1",
		Port:   -1,
		NoLog:  true,
		NoSigs: true,
	})
	if err != nil {
		return nil, err
	}
	go server.Start()
	if !server.ReadyForConnections(2 * time.Second) {
		server.Shutdown()
		return nil, fmt.Errorf("embedded nats server did not become ready")
	}
	conn, err := nats.Connect(server.ClientURL(), nats.Name("whiskd-runtime-events"))
	if err != nil {
		server.Shutdown()
		return nil, err
	}
	return &NATSBus{server: server, conn: conn, notify: make(chan struct{})}, nil
}

func (b *NATSBus) Publish(_ context.Context, event app.RuntimeEvent) error {
	if b == nil || b.conn == nil {
		return nil
	}
	b.mu.Lock()
	b.retained = append(b.retained, event)
	if len(b.retained) > retainedRuntimeEventLimit {
		b.retained = append([]app.RuntimeEvent(nil), b.retained[len(b.retained)-retainedRuntimeEventLimit:]...)
	}
	notify := b.notify
	b.notify = make(chan struct{})
	close(notify)
	b.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.conn.Publish(subjectFor(event.Type), data)
}

func (b *NATSBus) Next(ctx context.Context, afterSeq uint64) (app.NextRuntimeEventResult, error) {
	if b == nil || b.conn == nil {
		return app.NextRuntimeEventResult{}, fmt.Errorf("nats event bus unavailable")
	}
	for {
		b.mu.Lock()
		if result, ok := b.nextRetainedLocked(afterSeq); ok {
			b.mu.Unlock()
			return result, nil
		}
		notify := b.notify
		b.mu.Unlock()

		select {
		case <-notify:
		case <-ctx.Done():
			return app.NextRuntimeEventResult{}, ctx.Err()
		}
	}
}

func (b *NATSBus) nextRetainedLocked(afterSeq uint64) (app.NextRuntimeEventResult, bool) {
	if len(b.retained) == 0 {
		return app.NextRuntimeEventResult{}, false
	}
	latest := b.retained[len(b.retained)-1].Seq
	if afterSeq == latest {
		return app.NextRuntimeEventResult{}, false
	}
	if afterSeq > latest {
		return app.NextRuntimeEventResult{Event: b.retained[0], Missed: afterSeq > 0}, true
	}
	for _, event := range b.retained {
		if event.Seq > afterSeq {
			missed := afterSeq > 0 && event.Seq > afterSeq+1
			return app.NextRuntimeEventResult{Event: event, Missed: missed}, true
		}
	}
	return app.NextRuntimeEventResult{}, false
}

func (b *NATSBus) Close() {
	if b == nil {
		return
	}
	if b.conn != nil {
		b.conn.Close()
	}
	if b.server != nil {
		b.server.Shutdown()
	}
}

func subjectFor(eventType app.RuntimeEventType) string {
	switch eventType {
	case app.EventSessionChanged:
		return subjectSessionChanged
	case app.EventPTYChanged:
		return subjectPTYChanged
	case app.EventPTYOutput:
		return subjectPTYOutput
	case app.EventWorkItemsChanged:
		return subjectWorkItemsChanged
	case app.EventStatusChanged:
		return subjectStatusChanged
	case app.EventPluginsChanged:
		return subjectPluginsChanged
	case app.EventMailboxChanged:
		return subjectMailboxChanged
	case app.EventAgentBridgeApprovalsChanged:
		return subjectAgentBridgeApprovalsChanged
	case app.EventAgentPromptsChanged:
		return subjectAgentPromptsChanged
	case app.EventAgentHookEventsChanged:
		return subjectAgentHookEventsChanged
	default:
		return "whisk.unknown"
	}
}
