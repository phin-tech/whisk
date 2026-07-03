package wailsapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestDaemonRestartPolicyRequiresOptInManagedDaemonAndBoundsRetries(t *testing.T) {
	policy := newDaemonRestartPolicy(2)

	if attempt, ok := policy.Next(DaemonStatus{Running: true, Managed: true}, false, false); ok || attempt != 0 {
		t.Fatalf("disabled policy attempt = %d, ok = %v, want no restart", attempt, ok)
	}
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, false); ok || attempt != 0 {
		t.Fatalf("disabled-at-loss policy attempt = %d, ok = %v, want no delayed restart", attempt, ok)
	}

	policy = newDaemonRestartPolicy(2)
	if _, ok := policy.Next(DaemonStatus{Running: true, Managed: true}, true, false); ok {
		t.Fatalf("running daemon should only arm the policy")
	}
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, false); !ok || attempt != 1 {
		t.Fatalf("first managed restart attempt = %d, ok = %v", attempt, ok)
	}
	policy.RecordFailure()
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, false); !ok || attempt != 2 {
		t.Fatalf("second managed restart attempt = %d, ok = %v", attempt, ok)
	}
	policy.RecordFailure()
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, false); ok || attempt != 0 {
		t.Fatalf("bounded restart attempt = %d, ok = %v, want exhausted", attempt, ok)
	}

	policy = newDaemonRestartPolicy(2)
	_, _ = policy.Next(DaemonStatus{Running: true, Managed: false}, true, false)
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, false); ok || attempt != 0 {
		t.Fatalf("unmanaged restart attempt = %d, ok = %v, want no restart", attempt, ok)
	}

	policy = newDaemonRestartPolicy(2)
	_, _ = policy.Next(DaemonStatus{Running: true, Managed: true}, true, false)
	if attempt, ok := policy.Next(DaemonStatus{Running: false}, true, true); ok || attempt != 0 {
		t.Fatalf("expected stop restart attempt = %d, ok = %v, want no restart", attempt, ok)
	}
}

func TestDaemonStatusWatcherEmitsAvailabilityManagedAndVersionChanges(t *testing.T) {
	server := newDaemonStatusServer(t)
	supervisor := &daemonSupervisorFake{}
	store := &daemonSettingsStoreFake{settings: appsettings.Default()}
	emitter := &daemonStatusEmitterFake{}

	service := NewServiceWithSettings(client.NewHTTP(server.URL, server.Client()), store)
	service.supervisor = supervisor
	service.daemonStatusInterval = 10 * time.Millisecond
	service.daemonStatusTimeout = 100 * time.Millisecond
	AttachEventEmitter(service, emitter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.startDaemonStatusWatcher(ctx)
	t.Cleanup(service.stopDaemonStatusWatcher)

	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Version == "v1" && !status.Managed
	})

	supervisor.managed.Store(true)
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Version == "v1" && status.Managed
	})

	server.version.Store("v2")
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Version == "v2" && status.Managed
	})

	server.running.Store(false)
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return !status.Running && status.Managed
	})

	server.running.Store(true)
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Version == "v2" && status.Managed
	})
}

func TestDaemonStatusWatcherAutoRestartsManagedDaemonWithBoundedRetries(t *testing.T) {
	server := newDaemonStatusServer(t)
	store := &daemonSettingsStoreFake{settings: appsettings.Settings{
		StartupView:              appsettings.StartupViewSessions,
		AutoRestartManagedDaemon: true,
		KeepDaemonAlive:          true,
		HookLogEnabled:           true,
	}}
	emitter := &daemonStatusEmitterFake{}
	supervisor := &daemonSupervisorFake{}
	supervisor.managed.Store(true)
	supervisor.ensure = func(context.Context, string) (bool, error) {
		call := supervisor.ensureCalls.Add(1)
		if call == 1 {
			return false, errors.New("spawn failed")
		}
		server.running.Store(true)
		supervisor.managed.Store(true)
		return true, nil
	}

	service := NewServiceWithSettings(client.NewHTTP(server.URL, server.Client()), store)
	service.supervisor = supervisor
	service.daemonStatusInterval = 10 * time.Millisecond
	service.daemonStatusTimeout = 100 * time.Millisecond
	service.daemonControlTimeout = 100 * time.Millisecond
	service.daemonRestartMaxAttempts = 2
	AttachEventEmitter(service, emitter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.startDaemonStatusWatcher(ctx)
	t.Cleanup(service.stopDaemonStatusWatcher)

	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Managed
	})

	server.running.Store(false)
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return !status.Running && status.Restarting && status.RestartAttempt == 1
	})
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return !status.Running && status.RestartAttempt == 1 && !status.Restarting
	})
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return !status.Running && status.Restarting && status.RestartAttempt == 2
	})
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && status.Managed && status.RestartAttempt == 0
	})
	if got := supervisor.ensureCalls.Load(); got != 2 {
		t.Fatalf("ensure calls = %d, want 2", got)
	}
}

func TestDaemonStatusWatcherDoesNotAutoRestartUnmanagedDaemon(t *testing.T) {
	server := newDaemonStatusServer(t)
	store := &daemonSettingsStoreFake{settings: appsettings.Settings{
		StartupView:              appsettings.StartupViewSessions,
		AutoRestartManagedDaemon: true,
		KeepDaemonAlive:          true,
		HookLogEnabled:           true,
	}}
	emitter := &daemonStatusEmitterFake{}
	supervisor := &daemonSupervisorFake{}

	service := NewServiceWithSettings(client.NewHTTP(server.URL, server.Client()), store)
	service.supervisor = supervisor
	service.daemonStatusInterval = 10 * time.Millisecond
	service.daemonStatusTimeout = 100 * time.Millisecond
	service.daemonRestartMaxAttempts = 2
	AttachEventEmitter(service, emitter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.startDaemonStatusWatcher(ctx)
	t.Cleanup(service.stopDaemonStatusWatcher)

	emitter.waitFor(t, func(status DaemonStatus) bool {
		return status.Running && !status.Managed
	})
	server.running.Store(false)
	emitter.waitFor(t, func(status DaemonStatus) bool {
		return !status.Running && !status.Managed
	})
	time.Sleep(60 * time.Millisecond)
	if got := supervisor.ensureCalls.Load(); got != 0 {
		t.Fatalf("ensure calls = %d, want 0 for unmanaged daemon", got)
	}
}

type daemonStatusServer struct {
	*httptest.Server
	running atomic.Bool
	version atomic.Value
}

func newDaemonStatusServer(t *testing.T) *daemonStatusServer {
	t.Helper()
	status := &daemonStatusServer{}
	status.running.Store(true)
	status.version.Store("v1")
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		if !status.running.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, _ *http.Request) {
		if !status.running.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"apiVersion":%d,"gitSha":"abc123","version":%q,"dirty":false}`, protocol.DaemonAPIVersion, status.version.Load().(string))
	})
	status.Server = httptest.NewServer(mux)
	t.Cleanup(status.Close)
	return status
}

type daemonSupervisorFake struct {
	managed     atomic.Bool
	ensureCalls atomic.Int32
	ensure      func(context.Context, string) (bool, error)
}

func (f *daemonSupervisorFake) Status(ctx context.Context, baseURL string) daemon.StatusReport {
	status := daemon.StatusReport{Address: baseURL, Managed: f.managed.Load()}
	daemonClient := client.NewHTTP(baseURL, nil)
	if err := daemonClient.Health(ctx); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Running = true
	compat, err := daemonClient.Compatibility(ctx)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.APIVersion = compat.APIVersion
	status.GitSHA = compat.GitSHA
	status.Version = compat.Version
	status.Dirty = compat.Dirty
	return status
}

func (f *daemonSupervisorFake) Ensure(ctx context.Context, baseURL string) (bool, error) {
	if f.ensure != nil {
		return f.ensure(ctx, baseURL)
	}
	f.ensureCalls.Add(1)
	return true, nil
}

func (f *daemonSupervisorFake) Stop(context.Context, string) error {
	f.managed.Store(false)
	return nil
}

type daemonSettingsStoreFake struct {
	settings appsettings.Settings
}

func (f *daemonSettingsStoreFake) Load(context.Context) (appsettings.Settings, error) {
	return f.settings, nil
}

func (f *daemonSettingsStoreFake) Save(_ context.Context, settings appsettings.Settings) (appsettings.Settings, error) {
	f.settings = settings
	return settings, nil
}

type daemonStatusEmitterFake struct {
	mu     sync.Mutex
	events []DaemonStatus
}

func (f *daemonStatusEmitterFake) Emit(name string, data ...any) bool {
	if name != EventDaemonStatusChanged || len(data) != 1 {
		return false
	}
	status, ok := data[0].(DaemonStatus)
	if !ok {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, status)
	return false
}

func (f *daemonStatusEmitterFake) waitFor(t *testing.T, match func(DaemonStatus) bool) DaemonStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		for _, event := range f.events {
			if match(event) {
				f.mu.Unlock()
				return event
			}
		}
		f.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t.Fatalf("timed out waiting for daemon status event; events = %#v", f.events)
	return DaemonStatus{}
}
