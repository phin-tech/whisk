package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPServerSessionAndPTYFlow(t *testing.T) {
	backend := newFakePTYBackend()
	eventBus := newFakeEventBus()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: eventBus})
	handler := server.NewHTTP(runtime)

	health := getJSON[map[string]bool](t, handler, "/v1/health", http.StatusOK)
	if !health["ok"] {
		t.Fatalf("health = %#v", health)
	}

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	}, http.StatusCreated)
	if created.Session.ID == "" || created.MainPtyID == "" {
		t.Fatalf("created session missing ids: %#v", created)
	}

	sessions := getJSON[[]map[string]any](t, handler, "/v1/sessions", http.StatusOK)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %#v", sessions)
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/resize", protocol.ResizePTYRequest{Cols: 120, Rows: 40})
	if backend.records[created.MainPtyID].Cols != 120 || backend.records[created.MainPtyID].Rows != 40 {
		t.Fatalf("backend size = %#v", backend.records[created.MainPtyID])
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "hello"})
	snapshot := getJSON[protocol.OutputSnapshot](t, handler, "/v1/ptys/"+created.MainPtyID+"/output?from=0", http.StatusOK)
	if snapshot.Output != "hello" || snapshot.OutputBase64 != "aGVsbG8=" || snapshot.Offset != 5 {
		t.Fatalf("snapshot = %#v", snapshot)
	}

	split := postJSON[protocol.SplitPaneResult](t, handler, "/v1/sessions/"+created.Session.ID+"/split", protocol.SplitPaneRequest{
		TargetPaneID: created.Session.FocusedPaneID,
		Direction:    "vertical",
		Cols:         80,
		Rows:         24,
	}, http.StatusOK)
	if split.PaneID == "" || split.PtyID == "" {
		t.Fatalf("split = %#v", split)
	}

	ptys := getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	if len(ptys) != 2 {
		t.Fatalf("ptys = %#v", ptys)
	}
	byID := map[string]protocol.PTYInfo{}
	for _, pty := range ptys {
		byID[pty.ID] = pty
	}
	if byID[created.MainPtyID].SessionID != created.Session.ID || byID[created.MainPtyID].PaneID != created.Session.FocusedPaneID {
		t.Fatalf("main pty = %#v", byID[created.MainPtyID])
	}

	event := getJSON[protocol.RuntimeEvent](t, handler, "/v1/events/next?timeoutMs=10", http.StatusOK)
	if event.Type != "session.changed" {
		t.Fatalf("event = %#v", event)
	}

}

func TestHTTPServerNextEventTimeoutReturnsNoop(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: newFakePTYBackend(),
		EventSink:  newFakeEventBus(),
	})
	handler := server.NewHTTP(runtime)

	event := getJSON[protocol.RuntimeEvent](t, handler, "/v1/events/next?timeoutMs=1", http.StatusOK)
	if event.Type != protocol.RuntimeEventNone {
		t.Fatalf("event = %#v", event)
	}
}

func TestHTTPServerReportsBadRequests(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: newFakePTYBackend()})
	handler := server.NewHTTP(runtime)

	assertStatus(t, handler, http.MethodPost, "/v1/sessions", "{", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/sessions/missing/split", `{"targetPaneId":"pane","direction":"diagonal"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodGet, "/v1/ptys/missing/output?from=nope", "", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/write", `{"data":"x"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/resize", `{"cols":0,"rows":24}`, http.StatusBadRequest)
}

type fakePTYBackend struct {
	records map[string]app.PTYRecord
	outputs map[string][]byte
}

func newFakePTYBackend() *fakePTYBackend {
	return &fakePTYBackend{
		records: map[string]app.PTYRecord{},
		outputs: map[string][]byte{},
	}
}

func (b *fakePTYBackend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	record := app.PTYRecord{
		ID:         req.ID,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
		Running:    true,
	}
	b.records[req.ID] = record
	return record, nil
}

func (b *fakePTYBackend) Write(_ context.Context, ptyID string, data []byte) error {
	if _, ok := b.records[ptyID]; !ok {
		return errNotFound(ptyID)
	}
	b.outputs[ptyID] = append(b.outputs[ptyID], data...)
	return nil
}

func (b *fakePTYBackend) Resize(_ context.Context, ptyID string, size app.PTYSize) error {
	record, ok := b.records[ptyID]
	if !ok {
		return errNotFound(ptyID)
	}
	record.Cols = size.Cols
	record.Rows = size.Rows
	b.records[ptyID] = record
	return nil
}

func (b *fakePTYBackend) Attach(context.Context, app.AttachPTYRequest) (*app.PTYAttach, error) {
	ch := make(chan app.PTYEvent)
	close(ch)
	return &app.PTYAttach{Events: ch}, nil
}

func (b *fakePTYBackend) Output(_ context.Context, ptyID string, fromOffset uint64) (app.PTYOutputSnapshot, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, errNotFound(ptyID)
	}
	output := b.outputs[ptyID]
	if fromOffset > uint64(len(output)) {
		fromOffset = uint64(len(output))
	}
	return app.PTYOutputSnapshot{
		Record:      record,
		Offset:      fromOffset,
		OutputBytes: append([]byte(nil), output[fromOffset:]...),
	}, nil
}

func (b *fakePTYBackend) List(context.Context) ([]app.PTYRecord, error) {
	out := make([]app.PTYRecord, 0, len(b.records))
	for _, record := range b.records {
		out = append(out, record)
	}
	return out, nil
}

func (b *fakePTYBackend) Shutdown(context.Context) error {
	return nil
}

type fakeEventBus struct {
	ch chan app.RuntimeEvent
}

func newFakeEventBus() *fakeEventBus {
	return &fakeEventBus{ch: make(chan app.RuntimeEvent, 64)}
}

func (b *fakeEventBus) Publish(_ context.Context, event app.RuntimeEvent) error {
	b.ch <- event
	return nil
}

func (b *fakeEventBus) Next(ctx context.Context) (app.RuntimeEvent, error) {
	select {
	case event := <-b.ch:
		return event, nil
	case <-ctx.Done():
		return app.RuntimeEvent{}, ctx.Err()
	}
}

type errNotFound string

func (e errNotFound) Error() string {
	return "pty " + string(e) + " not found"
}

func postJSON[T any](t *testing.T, handler http.Handler, path string, body any, wantStatus int) T {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded)))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func postNoContent(t *testing.T, handler http.Handler, path string, body any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded)))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
}

func getJSON[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func assertStatus(t *testing.T, handler http.Handler, method string, path string, body string, wantStatus int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d body = %s", method, path, recorder.Code, recorder.Body.String())
	}
}
