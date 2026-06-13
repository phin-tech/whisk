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
	eventBus := newFakeEventBus()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: eventBus})
	handler := server.NewHTTP(runtime)

	health := getJSON[map[string]bool](t, handler, "/v1/health", http.StatusOK)
	if !health["ok"] {
		t.Fatalf("health = %#v", health)
	}
	compatibility := getJSON[protocol.CompatibilityResponse](t, handler, "/v1/compat", http.StatusOK)
	if compatibility.APIVersion != protocol.DaemonAPIVersion {
		t.Fatalf("compatibility = %#v", compatibility)
	}
	if compatibility.GitSHA == "" {
		t.Fatalf("compatibility missing git sha: %#v", compatibility)
	}

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	if created.Session.ID == "" || created.WindowID == "" || created.PaneID == "" || created.PTYID == nil || created.MainPtyID == "" {
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
	bookmark := postJSON[protocol.PTYBookmark](t, handler, "/v1/ptys/"+created.MainPtyID+"/bookmarks", protocol.AddPTYBookmarkRequest{
		Offset: 3,
		Kind:   "prompt",
		Label:  "Prompt",
	}, http.StatusCreated)
	if bookmark.ID == "" || bookmark.PTYID != created.MainPtyID || bookmark.Offset != 3 {
		t.Fatalf("bookmark = %#v", bookmark)
	}
	bookmarks := getJSON[[]protocol.PTYBookmark](t, handler, "/v1/ptys/"+created.MainPtyID+"/bookmarks", http.StatusOK)
	if len(bookmarks) != 1 || bookmarks[0].ID != bookmark.ID {
		t.Fatalf("bookmarks = %#v", bookmarks)
	}
	deleteNoContent(t, handler, "/v1/pty-bookmarks/"+bookmark.ID)
	bookmarks = getJSON[[]protocol.PTYBookmark](t, handler, "/v1/ptys/"+created.MainPtyID+"/bookmarks", http.StatusOK)
	if len(bookmarks) != 0 {
		t.Fatalf("bookmarks after delete = %#v", bookmarks)
	}

	split := postJSON[protocol.SplitPaneResult](t, handler, "/v1/sessions/"+created.Session.ID+"/split", protocol.SplitPaneRequest{
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    "vertical",
		InitialPTY:   &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusOK)
	if split.PaneID == "" || split.PTYID == nil || split.PtyID == "" {
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
	if byID[created.MainPtyID].SessionID != created.Session.ID || byID[created.MainPtyID].WindowID != created.WindowID || byID[created.MainPtyID].PaneID != created.PaneID {
		t.Fatalf("main pty = %#v", byID[created.MainPtyID])
	}

	event := getJSON[protocol.RuntimeEvent](t, handler, "/v1/events/next?timeoutMs=10", http.StatusOK)
	if event.Type != "session.changed" {
		t.Fatalf("event = %#v", event)
	}

	cleared := postJSON[protocol.ClearDaemonResponse](t, handler, "/v1/daemon/clear", protocol.ClearDaemonRequest{}, http.StatusOK)
	if cleared.SessionsCleared != 1 || cleared.PTYsCleared != 2 {
		t.Fatalf("cleared = %#v", cleared)
	}
	sessions = getJSON[[]map[string]any](t, handler, "/v1/sessions", http.StatusOK)
	if len(sessions) != 0 {
		t.Fatalf("sessions after clear = %#v", sessions)
	}
	ptys = getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	if len(ptys) != 0 {
		t.Fatalf("ptys after clear = %#v", ptys)
	}

}

func TestHTTPServerSessionLifecycleActions(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend})
	handler := server.NewHTTP(runtime)

	rootDir := t.TempDir()
	nextRoot := t.TempDir()
	paneDir := t.TempDir()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Empty",
		RootDir: rootDir,
	}, http.StatusCreated)
	if created.PTYID != nil {
		t.Fatalf("created should be empty: %#v", created)
	}

	updated := postJSON[map[string]any](t, handler, "/v1/sessions/"+created.Session.ID+"/set-root-dir", protocol.SetSessionRootDirRequest{
		RootDir: nextRoot,
	}, http.StatusOK)
	if updated["rootDir"] != nextRoot {
		t.Fatalf("updated root = %#v", updated)
	}

	updated = postJSON[map[string]any](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/set-working-dir", protocol.SetPaneWorkingDirRequest{
		WorkingDir: paneDir,
	}, http.StatusOK)
	panes := updated["panes"].(map[string]any)
	pane := panes[created.PaneID].(map[string]any)
	if pane["workingDir"] != paneDir {
		t.Fatalf("pane = %#v", pane)
	}

	started := postJSON[protocol.StartedPanePTY](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/start-pty", protocol.StartPanePTYRequest{
		Options: protocol.StartPTYOptions{Cols: 90, Rows: 30, Command: "echo server-command"},
	}, http.StatusCreated)
	if started.PTYID == "" || backend.records[started.PTYID].WorkingDir != paneDir {
		t.Fatalf("started = %#v record = %#v", started, backend.records[started.PTYID])
	}
	if string(backend.outputs[started.PTYID]) != "echo server-command\n" {
		t.Fatalf("initial command output = %q", string(backend.outputs[started.PTYID]))
	}
	execCreated := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Exec",
		RootDir: rootDir,
		InitialPTY: &protocol.StartPTYOptions{
			Cols:    81,
			Rows:    25,
			Command: "codex",
			Args:    []string{"--ask-for-approval=never", "Plan"},
			Env:     map[string]string{"WHISK_TEST": "1"},
			Exec:    true,
		},
	}, http.StatusCreated)
	if execCreated.PTYID == nil {
		t.Fatalf("exec created missing pty: %#v", execCreated)
	}
	execSpawn := backend.spawns[*execCreated.PTYID]
	if execSpawn.Command != "codex" || len(execSpawn.Args) != 2 || execSpawn.Args[1] != "Plan" || execSpawn.Env["WHISK_TEST"] != "1" {
		t.Fatalf("exec spawn = %#v", execSpawn)
	}
	if string(backend.outputs[*execCreated.PTYID]) != "" {
		t.Fatalf("exec pty should not receive command via stdin: %q", string(backend.outputs[*execCreated.PTYID]))
	}
	detached := postJSON[protocol.DetachedPanePTY](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/detach-pty", protocol.DetachPanePTYRequest{}, http.StatusOK)
	if detached.PTYID != started.PTYID {
		t.Fatalf("detached = %#v", detached)
	}
	ptys := getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	byPTY := map[string]protocol.PTYInfo{}
	for _, pty := range ptys {
		byPTY[pty.ID] = pty
	}
	if byPTY[started.PTYID].SessionID != created.Session.ID || byPTY[started.PTYID].PaneID != "" || byPTY[started.PTYID].OriginPaneID != created.PaneID || byPTY[started.PTYID].Status != "running" {
		t.Fatalf("detached pty = %#v", byPTY[started.PTYID])
	}

	restartSession := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Restart",
		RootDir:    rootDir,
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	killed := postJSON[protocol.PTYInfo](t, handler, "/v1/ptys/"+restartSession.MainPtyID+"/kill", protocol.KillPTYRequest{}, http.StatusOK)
	if killed.Status != "killed" || killed.Running {
		t.Fatalf("killed = %#v", killed)
	}
	restarted := postJSON[protocol.RestartedPanePTY](t, handler, "/v1/sessions/"+restartSession.Session.ID+"/panes/"+restartSession.PaneID+"/restart-pty", protocol.RestartPanePTYRequest{
		Options: protocol.StartPTYOptions{Cols: 100, Rows: 40},
	}, http.StatusCreated)
	if restarted.PTYID == "" || restarted.OldPTYID != restartSession.MainPtyID || backend.records[restarted.PTYID].Cols != 100 {
		t.Fatalf("restarted = %#v record = %#v", restarted, backend.records[restarted.PTYID])
	}

	closeSession := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Close",
		RootDir: rootDir,
	}, http.StatusCreated)
	split := postJSON[protocol.SplitPaneResult](t, handler, "/v1/sessions/"+closeSession.Session.ID+"/split", protocol.SplitPaneRequest{
		WindowID:     closeSession.WindowID,
		TargetPaneID: closeSession.PaneID,
		Direction:    "horizontal",
	}, http.StatusOK)
	closed := postJSON[map[string]any](t, handler, "/v1/sessions/"+closeSession.Session.ID+"/windows/"+closeSession.WindowID+"/panes/"+split.PaneID+"/close", protocol.ClosePaneRequest{}, http.StatusOK)
	closedPanes := closed["panes"].(map[string]any)
	if _, ok := closedPanes[split.PaneID]; ok {
		t.Fatalf("closed pane still present: %#v", closedPanes)
	}

	sessionToClose := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Close session",
		RootDir:    rootDir,
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	remaining := deleteJSON[[]map[string]any](t, handler, "/v1/sessions/"+sessionToClose.Session.ID, http.StatusOK)
	for _, candidate := range remaining {
		if candidate["id"] == sessionToClose.Session.ID {
			t.Fatalf("closed session still present: %#v", remaining)
		}
	}
	ptys = getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	for _, pty := range ptys {
		if pty.SessionID == sessionToClose.Session.ID && pty.Status != "killed" {
			t.Fatalf("closed session pty not killed: %#v", pty)
		}
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
	spawns  map[string]app.SpawnPTYRequest
}

func newFakePTYBackend() *fakePTYBackend {
	return &fakePTYBackend{
		records: map[string]app.PTYRecord{},
		outputs: map[string][]byte{},
		spawns:  map[string]app.SpawnPTYRequest{},
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
	b.spawns[req.ID] = req
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

func (b *fakePTYBackend) Kill(_ context.Context, ptyID string) (app.PTYRecord, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYRecord{}, errNotFound(ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	return record, nil
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
	b.records = map[string]app.PTYRecord{}
	b.outputs = map[string][]byte{}
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

func deleteJSON[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
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
