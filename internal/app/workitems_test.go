package app_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestRuntimeWorkItemRunLifecyclePersistsAndPublishes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		EventSink:     sink,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire runs"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		SessionID:        "sess_01",
		PTYID:            "pty_01",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != workitem.RunStateQueued || run.PromptSnapshot == "" {
		t.Fatalf("run = %#v", run)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("runs = %#v, err = %v", runs, err)
	}
	cancelled, err := runtime.CancelWorkItemRun(ctx, app.CancelWorkItemRunRequest{ID: run.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	if cancelled.Status != workitem.RunStateCancelled {
		t.Fatalf("cancelled = %#v", cancelled)
	}
	if store.saved.Runs[len(store.saved.Runs)-1].Status != workitem.RunStateCancelled {
		t.Fatalf("saved runs = %#v", store.saved.Runs)
	}
	if len(sink.events) != 4 {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeStartWorkItemRunLaunchesAgentPTY(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	worktreeDir := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire agent"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	item, err = runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/app-1-wire-agent",
		WorktreePath: worktreeDir,
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}

	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "codex",
		SystemPrompt:     "Be direct.",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != workitem.RunStateRunning || run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}
	if len(ptyBackend.spawns) != 1 || ptyBackend.spawns[0].WorkingDir != worktreeDir {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	writes := ptyBackend.writes[run.PTYID]
	if len(writes) != 2 {
		t.Fatalf("writes = %#v", writes)
	}
	if !strings.Contains(string(writes[0]), "'codex'") || !strings.Contains(string(writes[0]), "instructions=Be direct.") {
		t.Fatalf("command write = %q", string(writes[0]))
	}
	if !strings.Contains(string(writes[1]), "Implement the work item.") || !strings.Contains(string(writes[1]), "Wire agent") {
		t.Fatalf("prompt write = %q", string(writes[1]))
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil || len(sessions) != 1 || sessions[0].RootDir != worktreeDir {
		t.Fatalf("sessions = %#v, err = %v", sessions, err)
	}
}

type memoryWorkItemStore struct {
	saved workitem.Snapshot
}

func (s *memoryWorkItemStore) LoadWorkItems(context.Context) (workitem.Snapshot, error) {
	return s.saved, nil
}

func (s *memoryWorkItemStore) SaveWorkItems(_ context.Context, snapshot workitem.Snapshot) error {
	s.saved = snapshot
	return nil
}

type memoryEventSink struct {
	events []app.RuntimeEvent
}

func (s *memoryEventSink) Publish(_ context.Context, event app.RuntimeEvent) error {
	s.events = append(s.events, event)
	return nil
}

type memoryPTYBackend struct {
	records map[string]app.PTYRecord
	spawns  []app.SpawnPTYRequest
	writes  map[string][][]byte
}

func newMemoryPTYBackend() *memoryPTYBackend {
	return &memoryPTYBackend{
		records: map[string]app.PTYRecord{},
		writes:  map[string][][]byte{},
	}
}

func (b *memoryPTYBackend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	b.spawns = append(b.spawns, req)
	record := app.PTYRecord{
		ID:         req.ID,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
		Running:    true,
	}
	b.records[record.ID] = record
	return record, nil
}

func (b *memoryPTYBackend) Write(_ context.Context, ptyID string, data []byte) error {
	if _, ok := b.records[ptyID]; !ok {
		return fmt.Errorf("pty %s not found", ptyID)
	}
	b.writes[ptyID] = append(b.writes[ptyID], append([]byte(nil), data...))
	return nil
}

func (b *memoryPTYBackend) Resize(context.Context, string, app.PTYSize) error {
	return nil
}

func (b *memoryPTYBackend) Kill(_ context.Context, ptyID string) (app.PTYRecord, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYRecord{}, fmt.Errorf("pty %s not found", ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	return record, nil
}

func (b *memoryPTYBackend) Attach(context.Context, app.AttachPTYRequest) (*app.PTYAttach, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *memoryPTYBackend) Output(_ context.Context, ptyID string, _ uint64) (app.PTYOutputSnapshot, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, fmt.Errorf("pty %s not found", ptyID)
	}
	return app.PTYOutputSnapshot{Record: record}, nil
}

func (b *memoryPTYBackend) List(context.Context) ([]app.PTYRecord, error) {
	out := make([]app.PTYRecord, 0, len(b.records))
	for _, record := range b.records {
		out = append(out, record)
	}
	return out, nil
}

func (b *memoryPTYBackend) Shutdown(context.Context) error {
	return nil
}
