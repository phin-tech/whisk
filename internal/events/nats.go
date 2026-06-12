package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/phin-tech/whisk/internal/app"
)

const (
	subjectSessionChanged   = "whisk.session.changed"
	subjectPTYChanged       = "whisk.pty.changed"
	subjectPTYOutput        = "whisk.pty.output"
	subjectWorkItemsChanged = "whisk.workitems.changed"
	subjectStatusChanged    = "whisk.status.changed"
)

type NATSBus struct {
	server *natsserver.Server
	conn   *nats.Conn
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
	return &NATSBus{server: server, conn: conn}, nil
}

func (b *NATSBus) Publish(_ context.Context, event app.RuntimeEvent) error {
	if b == nil || b.conn == nil {
		return nil
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.conn.Publish(subjectFor(event.Type), data)
}

func (b *NATSBus) Next(ctx context.Context) (app.RuntimeEvent, error) {
	if b == nil || b.conn == nil {
		return app.RuntimeEvent{}, fmt.Errorf("nats event bus unavailable")
	}
	sub, err := b.conn.SubscribeSync("whisk.>")
	if err != nil {
		return app.RuntimeEvent{}, err
	}
	defer func() { _ = sub.Unsubscribe() }()
	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return app.RuntimeEvent{}, err
	}
	var event app.RuntimeEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return app.RuntimeEvent{}, err
	}
	return event, nil
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
	default:
		return "whisk.unknown"
	}
}
