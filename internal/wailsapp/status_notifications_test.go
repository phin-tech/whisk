package wailsapp

import (
	"context"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestStatusNotificationsSuppressFocusedPane(t *testing.T) {
	presenter := &statusNotificationPresenterFake{}
	service := &Service{
		notificationPresenter: presenter,
		notificationShown:     map[string]struct{}{},
		notificationEvents:    map[string]protocol.StatusEvent{},
	}
	if err := service.SetNotificationFocusContext(context.Background(), NotificationFocusContext{
		ActiveMain:    "session",
		SessionID:     "sess_01",
		PaneID:        "pane_01",
		WindowFocused: true,
	}); err != nil {
		t.Fatalf("set focus: %v", err)
	}

	event := protocol.StatusEvent{
		ID:                   "status_01",
		Kind:                 workitem.StatusKindQuestion,
		Message:              "Need API key.",
		SessionID:            "sess_01",
		PaneID:               "pane_01",
		RequiresAttention:    true,
		NotificationKey:      "status|session:sess_01|pane:pane_01|actor:codex|kind:question",
		NotificationSeverity: workitem.StatusNotificationSeverityAttention,
		CreatedAt:            time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	}
	service.presentStatusNotifications(context.Background(), []protocol.StatusEvent{event})
	if len(presenter.presented) != 0 {
		t.Fatalf("focused pane notification presented = %#v", presenter.presented)
	}

	if err := service.SetNotificationFocusContext(context.Background(), NotificationFocusContext{
		ActiveMain:    "session",
		SessionID:     "sess_01",
		PaneID:        "pane_01",
		WindowFocused: false,
	}); err != nil {
		t.Fatalf("set unfocused: %v", err)
	}
	service.presentStatusNotifications(context.Background(), []protocol.StatusEvent{event})
	if len(presenter.presented) != 0 {
		t.Fatalf("focused-suppressed event was presented later = %#v", presenter.presented)
	}
	event.ID = "status_02"
	service.presentStatusNotifications(context.Background(), []protocol.StatusEvent{event})
	if len(presenter.presented) != 1 || presenter.presented[0].ID != event.NotificationKey {
		t.Fatalf("presented = %#v", presenter.presented)
	}
}

func TestStatusNotificationsApplyCooldownAndEmitActivation(t *testing.T) {
	presenter := &statusNotificationPresenterFake{}
	emitter := &statusNotificationEmitterFake{}
	service := &Service{
		events:                emitter,
		notificationPresenter: presenter,
		notificationShown:     map[string]struct{}{},
		notificationEvents:    map[string]protocol.StatusEvent{},
	}
	first := protocol.StatusEvent{
		ID:                   "status_01",
		Kind:                 workitem.StatusKindBlocked,
		Message:              "Waiting on credentials.",
		SessionID:            "sess_01",
		PaneID:               "pane_01",
		RequiresAttention:    true,
		NotificationKey:      "status|session:sess_01|pane:pane_01|actor:codex|kind:blocked",
		NotificationSeverity: workitem.StatusNotificationSeverityWarning,
		CreatedAt:            time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	}
	second := first
	second.ID = "status_02"
	second.CreatedAt = first.CreatedAt.Add(time.Second)

	service.presentStatusNotifications(context.Background(), []protocol.StatusEvent{first, second})
	if len(presenter.presented) != 1 {
		t.Fatalf("presented = %#v", presenter.presented)
	}
	if presenter.presented[0].ID != first.NotificationKey {
		t.Fatalf("notification id = %q, want %q", presenter.presented[0].ID, first.NotificationKey)
	}

	service.handleStatusNotificationActivation(first.NotificationKey)
	if len(emitter.activations) != 1 || emitter.activations[0].Event.ID != first.ID {
		t.Fatalf("activations = %#v", emitter.activations)
	}
}

func TestStatusNotificationsReuseNativeIDAndActivateLatestEventAfterCooldown(t *testing.T) {
	emitter := &statusNotificationEmitterFake{}
	service := &Service{
		events:             emitter,
		notificationShown:  map[string]struct{}{},
		notificationEvents: map[string]protocol.StatusEvent{},
	}
	first := protocol.StatusEvent{
		ID:                   "status_01",
		Kind:                 workitem.StatusKindQuestion,
		Message:              "Need API key.",
		SessionID:            "sess_01",
		PaneID:               "pane_01",
		RequiresAttention:    true,
		NotificationKey:      "status|session:sess_01|pane:pane_01|actor:codex|kind:question",
		NotificationSeverity: workitem.StatusNotificationSeverityAttention,
		CreatedAt:            time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	}
	second := first
	second.ID = "status_02"
	second.Message = "Need production API key."
	second.CreatedAt = first.CreatedAt.Add(defaultStatusNotificationCooldown + time.Second)

	firstNotification, ok := service.prepareStatusNotification(first, first.CreatedAt)
	if !ok {
		t.Fatalf("first notification suppressed")
	}
	secondNotification, ok := service.prepareStatusNotification(second, second.CreatedAt)
	if !ok {
		t.Fatalf("second notification suppressed")
	}
	if firstNotification.ID != first.NotificationKey || secondNotification.ID != first.NotificationKey {
		t.Fatalf("notification ids = %q, %q; want %q", firstNotification.ID, secondNotification.ID, first.NotificationKey)
	}

	service.handleStatusNotificationActivation(first.NotificationKey)
	if len(emitter.activations) != 1 || emitter.activations[0].Event.ID != second.ID {
		t.Fatalf("activations = %#v", emitter.activations)
	}
}

func TestStatusNotificationWatcherRefreshesUnreadEvents(t *testing.T) {
	presenter := &statusNotificationPresenterFake{presentedCh: make(chan statusNotification, 1)}
	runtimeClient := &statusNotificationRuntimeClientFake{
		nextEvents: make(chan protocol.NextEventResponse, 1),
		statusEvents: []protocol.StatusEvent{{
			ID:                   "status_01",
			Kind:                 workitem.StatusKindQuestion,
			Message:              "Need API key.",
			SessionID:            "sess_01",
			PaneID:               "pane_01",
			RequiresAttention:    true,
			NotificationKey:      "status|session:sess_01|pane:pane_01|actor:codex|kind:question",
			NotificationSeverity: workitem.StatusNotificationSeverityAttention,
			CreatedAt:            time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
		}},
	}
	service := &Service{
		client:                   runtimeClient,
		notificationPresenter:    presenter,
		notificationShown:        map[string]struct{}{},
		notificationEvents:       map[string]protocol.StatusEvent{},
		notificationEventTimeout: time.Millisecond,
	}

	ctx := context.Background()
	runtimeClient.nextEvents <- protocol.NextEventResponse{Event: protocol.RuntimeEvent{Type: "status.changed", Seq: 1}}
	service.startStatusNotificationWatcher(ctx)
	defer service.stopStatusNotificationWatcher()

	select {
	case notification := <-presenter.presentedCh:
		if notification.Event.ID != "status_01" {
			t.Fatalf("notification event = %#v", notification.Event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}
	if runtimeClient.nextEventReq.TimeoutMs != 1 {
		t.Fatalf("next event timeout = %d, want 1", runtimeClient.nextEventReq.TimeoutMs)
	}
	if !runtimeClient.listStatusEventsReq.UnreadOnly {
		t.Fatalf("list status events request = %#v, want unread-only", runtimeClient.listStatusEventsReq)
	}
}

type statusNotificationPresenterFake struct {
	presented   []statusNotification
	presentedCh chan statusNotification
}

func (f *statusNotificationPresenterFake) Start(context.Context, application.ServiceOptions, func(string)) error {
	return nil
}

func (f *statusNotificationPresenterFake) Stop() error {
	return nil
}

func (f *statusNotificationPresenterFake) Present(_ context.Context, notification statusNotification) error {
	f.presented = append(f.presented, notification)
	if f.presentedCh != nil {
		f.presentedCh <- notification
	}
	return nil
}

type statusNotificationEmitterFake struct {
	activations []StatusNotificationActivation
}

func (f *statusNotificationEmitterFake) Emit(name string, data ...any) bool {
	if name != EventStatusNotificationActivated || len(data) != 1 {
		return false
	}
	activation, ok := data[0].(StatusNotificationActivation)
	if !ok {
		return false
	}
	f.activations = append(f.activations, activation)
	return true
}

type statusNotificationRuntimeClientFake struct {
	client.RuntimeClient

	nextEvents          chan protocol.NextEventResponse
	statusEvents        []protocol.StatusEvent
	nextEventReq        protocol.NextEventRequest
	listStatusEventsReq protocol.ListStatusEventsRequest
}

func (f *statusNotificationRuntimeClientFake) NextEvent(ctx context.Context, req protocol.NextEventRequest) (protocol.NextEventResponse, error) {
	f.nextEventReq = req
	select {
	case event := <-f.nextEvents:
		return event, nil
	case <-ctx.Done():
		return protocol.NextEventResponse{}, ctx.Err()
	}
}

func (f *statusNotificationRuntimeClientFake) ListStatusEvents(_ context.Context, req protocol.ListStatusEventsRequest) ([]protocol.StatusEvent, error) {
	f.listStatusEventsReq = req
	return f.statusEvents, nil
}
