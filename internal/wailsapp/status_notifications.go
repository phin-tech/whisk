package wailsapp

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainnotification "github.com/phin-tech/whisk/internal/domain/notification"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/wailsapp/wails/v3/pkg/application"
	wailsnotifications "github.com/wailsapp/wails/v3/pkg/services/notifications"
)

const (
	EventStatusNotificationActivated = "status-notification:activated"

	defaultStatusNotificationCooldown     = 5 * time.Second
	defaultStatusNotificationEventTimeout = 30 * time.Second
)

type NotificationFocusContext struct {
	ActiveMain    string `json:"activeMain,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
	PaneID        string `json:"paneId,omitempty"`
	WindowFocused bool   `json:"windowFocused"`
}

type StatusNotificationActivation struct {
	Event protocol.StatusEvent `json:"event"`
}

type statusNotification struct {
	ID       string
	Title    string
	Subtitle string
	Body     string
	Event    protocol.StatusEvent
}

type statusNotificationPresenter interface {
	Start(context.Context, application.ServiceOptions, func(string)) error
	Stop() error
	Present(context.Context, statusNotification) error
}

type desktopStatusNotificationPresenter struct {
	mu              sync.RWMutex
	native          *wailsnotifications.NotificationService
	nativeAvailable bool
	activate        func(string)
}

func newDesktopStatusNotificationPresenter() statusNotificationPresenter {
	return &desktopStatusNotificationPresenter{native: wailsnotifications.New()}
}

func (p *desktopStatusNotificationPresenter) Start(ctx context.Context, options application.ServiceOptions, activate func(string)) error {
	p.mu.Lock()
	p.activate = activate
	native := p.native
	p.mu.Unlock()
	if native == nil {
		return nil
	}
	if err := native.ServiceStartup(ctx, options); err != nil {
		return nil
	}
	native.OnNotificationResponse(func(result wailsnotifications.NotificationResult) {
		if result.Error != nil {
			return
		}
		id := strings.TrimSpace(result.Response.ID)
		if id == "" {
			if value, ok := result.Response.UserInfo["notificationId"].(string); ok {
				id = strings.TrimSpace(value)
			}
		}
		if id == "" {
			return
		}
		p.mu.RLock()
		callback := p.activate
		p.mu.RUnlock()
		if callback != nil {
			callback(id)
		}
	})
	p.mu.Lock()
	p.nativeAvailable = true
	p.mu.Unlock()
	return nil
}

func (p *desktopStatusNotificationPresenter) Stop() error {
	p.mu.Lock()
	native := p.native
	available := p.nativeAvailable
	p.nativeAvailable = false
	p.activate = nil
	p.mu.Unlock()
	if native != nil && available {
		return native.ServiceShutdown()
	}
	return nil
}

func (p *desktopStatusNotificationPresenter) Present(ctx context.Context, notification statusNotification) error {
	p.mu.RLock()
	native := p.native
	available := p.nativeAvailable
	p.mu.RUnlock()
	if native != nil && available {
		err := native.SendNotification(wailsnotifications.NotificationOptions{
			ID:       notification.ID,
			Title:    notification.Title,
			Subtitle: notification.Subtitle,
			Body:     notification.Body,
			Data: map[string]interface{}{
				"notificationId": notification.ID,
				"eventId":        notification.Event.ID,
				"sessionId":      notification.Event.SessionID,
				"paneId":         notification.Event.PaneID,
				"workItemId":     notification.Event.WorkItemID,
			},
		})
		if err == nil {
			return nil
		}
	}
	return presentStatusNotificationWithCommand(ctx, notification)
}

func presentStatusNotificationWithCommand(ctx context.Context, notification statusNotification) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "darwin":
		script := "display notification " + appleScriptString(notification.Body) + " with title " + appleScriptString(notification.Title)
		if notification.Subtitle != "" {
			script += " subtitle " + appleScriptString(notification.Subtitle)
		}
		return exec.CommandContext(cmdCtx, "osascript", "-e", script).Run()
	case "linux":
		return exec.CommandContext(cmdCtx, "notify-send", notification.Title, notification.Body).Run()
	default:
		return fmt.Errorf("native notifications unavailable")
	}
}

func appleScriptString(value string) string {
	quoted := strconv.Quote(value)
	return strings.ReplaceAll(quoted, `\n`, `\\n`)
}

func (s *Service) SetNotificationFocusContext(_ context.Context, focus NotificationFocusContext) error {
	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	s.notificationFocus = NotificationFocusContext{
		ActiveMain:    strings.TrimSpace(focus.ActiveMain),
		SessionID:     strings.TrimSpace(focus.SessionID),
		PaneID:        strings.TrimSpace(focus.PaneID),
		WindowFocused: focus.WindowFocused,
	}
	return nil
}

func (s *Service) startStatusNotificationWatcher(ctx context.Context) {
	if s.client == nil || s.notificationPresenter == nil {
		return
	}

	s.notificationWatchMu.Lock()
	if s.notificationWatchCancel != nil {
		s.notificationWatchMu.Unlock()
		return
	}
	watchCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.notificationWatchCancel = cancel
	s.notificationWatchDone = done
	s.notificationWatchMu.Unlock()

	go s.runStatusNotificationWatcher(watchCtx, done)
}

func (s *Service) stopStatusNotificationWatcher() {
	s.notificationWatchMu.Lock()
	cancel := s.notificationWatchCancel
	done := s.notificationWatchDone
	s.notificationWatchCancel = nil
	s.notificationWatchDone = nil
	s.notificationWatchMu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (s *Service) runStatusNotificationWatcher(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	timeoutMs := int(s.notificationEventTimeout / time.Millisecond)
	if timeoutMs <= 0 {
		timeoutMs = int(defaultStatusNotificationEventTimeout / time.Millisecond)
	}
	for {
		event, err := s.client.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: timeoutMs})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			continue
		}
		if event.Type == protocol.RuntimeEventNone {
			continue
		}
		if event.Type != "status.changed" {
			continue
		}
		events, err := s.client.ListStatusEvents(ctx, protocol.ListStatusEventsRequest{UnreadOnly: true})
		if err != nil {
			continue
		}
		s.presentStatusNotifications(ctx, events)
	}
}

func (s *Service) presentStatusNotifications(ctx context.Context, events []protocol.StatusEvent) {
	if s.notificationPresenter == nil {
		return
	}
	ordered := append([]protocol.StatusEvent(nil), events...)
	sort.Slice(ordered, func(i, j int) bool {
		if !ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, event := range ordered {
		notification, ok := s.prepareStatusNotification(event, time.Now().UTC())
		if !ok {
			continue
		}
		_ = s.notificationPresenter.Present(ctx, notification)
	}
}

func (s *Service) prepareStatusNotification(event protocol.StatusEvent, now time.Time) (statusNotification, bool) {
	if !event.RequiresAttention || event.ReadAt != nil {
		return statusNotification{}, false
	}
	key := statusNotificationKey(event)
	if key == "" {
		return statusNotification{}, false
	}

	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	if s.notificationShown == nil {
		s.notificationShown = map[string]struct{}{}
	}
	if s.notificationEvents == nil {
		s.notificationEvents = map[string]protocol.StatusEvent{}
	}
	if _, shown := s.notificationShown[event.ID]; shown {
		return statusNotification{}, false
	}
	if statusNotificationFocused(event, s.notificationFocus) {
		s.notificationShown[event.ID] = struct{}{}
		return statusNotification{}, false
	}
	next, allowed := domainnotification.ApplyCooldown(s.notificationCooldown, key, now, defaultStatusNotificationCooldown)
	s.notificationCooldown = next
	if !allowed {
		s.notificationShown[event.ID] = struct{}{}
		return statusNotification{}, false
	}
	s.notificationShown[event.ID] = struct{}{}
	s.notificationEvents[key] = event
	return statusNotification{
		ID:       key,
		Title:    statusNotificationTitle(event),
		Subtitle: statusNotificationSubtitle(event),
		Body:     statusNotificationBody(event),
		Event:    event,
	}, true
}

func (s *Service) handleStatusNotificationActivation(notificationID string) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return
	}
	s.notificationMu.Lock()
	event, ok := s.notificationEvents[notificationID]
	if ok {
		delete(s.notificationEvents, notificationID)
	}
	s.notificationMu.Unlock()
	if !ok || s.events == nil {
		return
	}
	s.events.Emit(EventStatusNotificationActivated, StatusNotificationActivation{Event: event})
}

func statusNotificationFocused(event protocol.StatusEvent, focus NotificationFocusContext) bool {
	if !focus.WindowFocused || focus.ActiveMain != "session" {
		return false
	}
	if event.SessionID != "" && event.SessionID != focus.SessionID {
		return false
	}
	if event.PaneID != "" {
		return event.PaneID == focus.PaneID
	}
	return event.SessionID != "" && event.SessionID == focus.SessionID
}

func statusNotificationKey(event protocol.StatusEvent) string {
	if key := strings.TrimSpace(event.NotificationKey); key != "" {
		return key
	}
	return strings.TrimSpace(event.ID)
}

func statusNotificationTitle(event protocol.StatusEvent) string {
	switch event.Kind {
	case workitem.StatusKindQuestion:
		return "Whisk needs input"
	case workitem.StatusKindBlocked:
		return "Whisk run blocked"
	default:
		return "Whisk status"
	}
}

func statusNotificationSubtitle(event protocol.StatusEvent) string {
	if event.Actor != "" {
		return event.Actor
	}
	return event.NotificationSeverity
}

func statusNotificationBody(event protocol.StatusEvent) string {
	if strings.TrimSpace(event.Message) != "" {
		return event.Message
	}
	return event.Kind
}
