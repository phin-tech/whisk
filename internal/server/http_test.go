package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend})
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
	if snapshot.Output != "hello" || snapshot.Offset != 5 {
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
}

func TestHTTPServerWorktreeFlow(t *testing.T) {
	backend := &fakeWorktreeBackend{
		status: app.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      app.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []app.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature", Kind: "worktree", Dirty: true, Locked: true}},
		created:   app.CreatedWorktree{Path: "/repo/.worktrees/created"},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Worktrees: backend})
	handler := server.NewHTTP(runtime)

	status := postJSON[protocol.WorktrunkStatus](t, handler, "/v1/worktrunk/detect", protocol.DetectWorktrunkRequest{
		RepoPath:     "/repo",
		OverridePath: "/custom/wt",
	}, http.StatusOK)
	if !status.Available || !status.ConfigFound || status.Binary.Path != "/bin/wt" {
		t.Fatalf("status = %#v", status)
	}
	if backend.detectReq.OverridePath != "/custom/wt" {
		t.Fatalf("detect req = %#v", backend.detectReq)
	}

	worktrees := postJSON[[]protocol.Worktree](t, handler, "/v1/worktrees/list", protocol.ListWorktreesRequest{RepoPath: "/repo"}, http.StatusOK)
	if len(worktrees) != 1 || worktrees[0].Branch != "feature" || !worktrees[0].Dirty || !worktrees[0].Locked {
		t.Fatalf("worktrees = %#v", worktrees)
	}

	created := postJSON[protocol.CreatedWorktree](t, handler, "/v1/worktrees/create", protocol.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	}, http.StatusCreated)
	if created.Path != "/repo/.worktrees/created" || backend.createReq.Base != "main" {
		t.Fatalf("created = %#v req = %#v", created, backend.createReq)
	}

	postNoContent(t, handler, "/v1/worktrees/remove", protocol.RemoveWorktreeRequest{
		RepoPath:     "/repo",
		WorktreePath: "/repo/.worktrees/created",
	})
	if backend.removeReq.WorktreePath != "/repo/.worktrees/created" {
		t.Fatalf("remove req = %#v", backend.removeReq)
	}
}

func TestHTTPServerHTTPForwardFlow(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/nested" || r.URL.RawQuery != "q=1" {
			t.Fatalf("target path = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Forward-Test", "hit")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("target:" + string(body)))
	}))
	t.Cleanup(target.Close)

	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime)

	created := postJSON[protocol.HTTPForward](t, handler, "/v1/http-forwards", protocol.CreateHTTPForwardRequest{
		Name:      "difit",
		TargetURL: target.URL + "/base",
		SessionID: "session_01",
	}, http.StatusCreated)
	if created.ID == "" || created.Name != "difit" || created.SessionID != "session_01" {
		t.Fatalf("created = %#v", created)
	}

	forwards := getJSON[[]protocol.HTTPForward](t, handler, "/v1/http-forwards", http.StatusOK)
	if len(forwards) != 1 || forwards[0].ID != created.ID {
		t.Fatalf("forwards = %#v", forwards)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/http-forwards/"+created.ID+"/proxy/nested?q=1", strings.NewReader("body"))
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted || recorder.Header().Get("X-Forward-Test") != "hit" || recorder.Body.String() != "target:body" {
		t.Fatalf("proxy status=%d header=%q body=%q", recorder.Code, recorder.Header().Get("X-Forward-Test"), recorder.Body.String())
	}

	deleteNoContent(t, handler, "/v1/http-forwards/"+created.ID)
	assertStatus(t, handler, http.MethodGet, "/v1/http-forwards/"+created.ID+"/proxy", "", http.StatusNotFound)
}

func TestHTTPServerReportsBadRequests(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: newFakePTYBackend()})
	handler := server.NewHTTP(runtime)

	assertStatus(t, handler, http.MethodPost, "/v1/sessions", "{", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/sessions/missing/split", `{"targetPaneId":"pane","direction":"diagonal"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodGet, "/v1/ptys/missing/output?from=nope", "", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/write", `{"data":"x"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/resize", `{"cols":0,"rows":24}`, http.StatusBadRequest)

	assertStatus(t, handler, http.MethodPost, "/v1/worktrunk/detect", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrunk/detect", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/list", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/create", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/remove", `{}`, http.StatusBadRequest)

	assertStatus(t, handler, http.MethodPost, "/v1/http-forwards", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/http-forwards", `{"targetUrl":"http://example.com"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodDelete, "/v1/http-forwards/missing", "", http.StatusNotFound)
	assertStatus(t, handler, http.MethodGet, "/v1/http-forwards/missing/proxy", "", http.StatusNotFound)
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
	panic("not used")
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

func (b *fakePTYBackend) Shutdown(context.Context) error {
	return nil
}

type errNotFound string

func (e errNotFound) Error() string {
	return "pty " + string(e) + " not found"
}

type fakeWorktreeBackend struct {
	status    app.WorktrunkStatus
	worktrees []app.Worktree
	created   app.CreatedWorktree

	detectReq app.DetectWorktrunkRequest
	listReq   app.ListWorktreesRequest
	createReq app.CreateWorktreeRequest
	removeReq app.RemoveWorktreeRequest
}

func (b *fakeWorktreeBackend) DetectWorktrunk(_ context.Context, req app.DetectWorktrunkRequest) (app.WorktrunkStatus, error) {
	b.detectReq = req
	return b.status, nil
}

func (b *fakeWorktreeBackend) ListWorktrees(_ context.Context, req app.ListWorktreesRequest) ([]app.Worktree, error) {
	b.listReq = req
	return b.worktrees, nil
}

func (b *fakeWorktreeBackend) CreateWorktree(_ context.Context, req app.CreateWorktreeRequest) (app.CreatedWorktree, error) {
	b.createReq = req
	return b.created, nil
}

func (b *fakeWorktreeBackend) RemoveWorktree(_ context.Context, req app.RemoveWorktreeRequest) error {
	b.removeReq = req
	return nil
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

func deleteNoContent(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
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
