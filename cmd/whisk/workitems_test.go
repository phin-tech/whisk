package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunProjectCreatePrintsJSONContract(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/projects" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Name != "App" || req.RootDir != root {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.Project{ID: "proj_01", Name: req.Name, Slug: "app", RootDir: req.RootDir})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"project", "create", "-url", server.URL, "-name", "App", "-root", root, "-json"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var project protocol.Project
	if err := json.Unmarshal([]byte(output), &project); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if project.ID != "proj_01" || project.RootDir != root {
		t.Fatalf("project = %#v", project)
	}
}

func TestRunProjectListPrintsJSONContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/projects" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]protocol.Project{{ID: "proj_01", Name: "App"}})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"project", "list", "-url", server.URL, "-json"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var projects []protocol.Project
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if len(projects) != 1 || projects[0].ID != "proj_01" {
		t.Fatalf("projects = %#v", projects)
	}
}

func TestRunWorkItemCreateListAndActionsUseDaemonAPI(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.Method+" "+r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items":
			var req protocol.CreateWorkItemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			if req.ProjectID != "proj_01" || req.Title != "Task" || req.Actor != "agent" {
				t.Fatalf("create request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", ProjectID: req.ProjectID, Number: 1, Title: req.Title, StageID: "backlog", RunState: workitem.RunStateIdle})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/work-items":
			if r.URL.Query().Get("projectId") != "proj_01" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]protocol.WorkItem{{ID: "wi_01", ProjectID: "proj_01", Number: 1, Title: "Task"}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_01/bind-worktree":
			var req protocol.BindWorkItemWorktreeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode bind: %v", err)
			}
			if req.Branch != "whisk/app-1-task" || req.WorktreePath == "" {
				t.Fatalf("bind request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", Number: 1, Worktree: &protocol.WorktreeBinding{Branch: req.Branch, WorktreePath: req.WorktreePath}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_01/move":
			var req protocol.MoveWorkItemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode move: %v", err)
			}
			if req.StageID != "ready" {
				t.Fatalf("move request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", Number: 1, StageID: req.StageID})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_01/attachments":
			var req protocol.AddWorkItemAttachmentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode attach: %v", err)
			}
			if req.Kind != workitem.AttachmentKindFile || req.Path != "docs/spec.md" {
				t.Fatalf("attach request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", Number: 1, Attachments: []protocol.Attachment{{ID: "att_01", Kind: req.Kind, Path: req.Path}}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_01/delete":
			var req protocol.DeleteWorkItemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode delete: %v", err)
			}
			if req.Actor != "agent" {
				t.Fatalf("delete request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", Number: 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"work-item", "create", "-url", server.URL, "-project", "proj_01", "-title", "Task", "-actor", "agent", "-json"})
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var item protocol.WorkItem
	if err := json.Unmarshal([]byte(output), &item); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if item.ID != "wi_01" || item.Number != 1 {
		t.Fatalf("item = %#v", item)
	}

	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "list", "-url", server.URL, "-project", "proj_01", "-json"})
	}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "bind-worktree", "-url", server.URL, "-branch", "whisk/app-1-task", "-path", ".", "-json", "wi_01"})
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "move", "-url", server.URL, "-stage", "ready", "-json", "wi_01"})
	}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "attach-file", "-url", server.URL, "-json", "wi_01", "docs/spec.md"})
	}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "delete", "-url", server.URL, "-actor", "agent", "-json", "wi_01"})
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if requests["POST /v1/work-items"] != 1 || requests["GET /v1/work-items"] != 1 || requests["POST /v1/work-items/wi_01/delete"] != 1 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestRunWorkItemUsesEnvProjectDefault(t *testing.T) {
	t.Setenv("WHISK_PROJECT", "proj_env")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.CreateWorkItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.ProjectID != "proj_env" {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_01", ProjectID: req.ProjectID, Number: 1})
	}))
	defer server.Close()

	if _, err := captureStdout(func() error {
		return run([]string{"work-item", "create", "-url", server.URL, "-title", "Task", "-json"})
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunCommandsUseDaemonAPI(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.Method+" "+r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/work-item-runs":
			if r.URL.Query().Get("workItemId") != "wi_01" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]protocol.WorkItemRun{{ID: "run_01", WorkItemID: "wi_01", Status: "queued", Preset: "writer", PromptTemplateID: "implement"}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-item-runs":
			var req protocol.StartWorkItemRunRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode start: %v", err)
			}
			if req.WorkItemID != "wi_01" ||
				req.Preset != "writer" ||
				req.PromptTemplateID != "implement" ||
				req.SessionID != "sess_01" ||
				req.PTYID != "pty_01" ||
				!req.Launch ||
				req.AgentProfileID != "codex" ||
				req.SystemPrompt != "Be direct." ||
				req.Actor != "agent" {
				t.Fatalf("start request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItemRun{ID: "run_01", WorkItemID: req.WorkItemID, Status: "running", Preset: req.Preset, PromptTemplateID: req.PromptTemplateID, SessionID: "sess_02", PTYID: "pty_02"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-item-runs/run_01/cancel":
			var req protocol.CancelWorkItemRunRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode cancel: %v", err)
			}
			if req.Actor != "agent" {
				t.Fatalf("cancel request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItemRun{ID: "run_01", Status: "cancelled"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"run", "start", "-url", server.URL, "-work-item", "wi_01", "-preset", "writer", "-template", "implement", "-session", "sess_01", "-pty", "pty_01", "-agent-profile", "codex", "-system-prompt", "Be direct.", "-actor", "agent", "-json"})
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	var runResult protocol.WorkItemRun
	if err := json.Unmarshal([]byte(output), &runResult); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if runResult.ID != "run_01" || runResult.Preset != "writer" || runResult.Status != "running" || runResult.SessionID != "sess_02" || runResult.PTYID != "pty_02" {
		t.Fatalf("run = %#v", runResult)
	}

	if _, err := captureStdout(func() error {
		return run([]string{"run", "list", "-url", server.URL, "-work-item", "wi_01", "-json"})
	}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"run", "cancel", "-url", server.URL, "-actor", "agent", "-json", "run_01"})
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if requests["POST /v1/work-item-runs"] != 1 || requests["GET /v1/work-item-runs"] != 1 || requests["POST /v1/work-item-runs/run_01/cancel"] != 1 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestRunStatusUsesInjectedEnvironmentContext(t *testing.T) {
	t.Setenv("WHISK_RUN_ID", "run_01")
	t.Setenv("WHISK_SESSION_ID", "sess_01")
	t.Setenv("WHISK_PTY_ID", "pty_01")
	t.Setenv("WHISK_ACTOR", "agent")
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.Method+" "+r.URL.Path]++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/status" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req protocol.ReportStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		if req.Kind != workitem.StatusKindQuestion ||
			req.Message != "Need the staging API key." ||
			req.Actor != "agent" ||
			req.RunID != "run_01" ||
			req.SessionID != "sess_01" ||
			req.PTYID != "pty_01" {
			t.Fatalf("status request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.ReportStatusResponse{
			Event: protocol.StatusEvent{
				ID:                "status_01",
				Kind:              workitem.StatusKindQuestion,
				Message:           req.Message,
				Actor:             req.Actor,
				RunID:             req.RunID,
				SessionID:         req.SessionID,
				PTYID:             req.PTYID,
				RequiresAttention: true,
			},
			Run: &protocol.WorkItemRun{ID: req.RunID, Status: workitem.RunStateAwaitingInput},
		})
	}))
	defer server.Close()
	t.Setenv("WHISKD_URL", server.URL)

	output, err := captureStdout(func() error {
		return run([]string{"status", "question", "-message", "Need the staging API key.", "-json"})
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var report protocol.ReportStatusResponse
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if report.Event.ID != "status_01" || !report.Event.RequiresAttention || report.Run == nil || report.Run.Status != workitem.RunStateAwaitingInput {
		t.Fatalf("report = %#v", report)
	}
	if requests["POST /v1/status"] != 1 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestRunWorkflowActionCommandsUseDaemonAPIAndEnvDefaults(t *testing.T) {
	t.Setenv("WHISK_WORK_ITEM_ID", "wi_env")
	t.Setenv("WHISK_RUN_ID", "run_env")
	t.Setenv("WHISK_ACTOR", "agent")
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.Method+" "+r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_env/start-planning":
			var req protocol.StartPlanningRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode planning: %v", err)
			}
			if req.WorkItemID != "wi_env" || req.Actor != "agent" {
				t.Fatalf("planning request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItemRun{ID: "run_plan", WorkItemID: req.WorkItemID, PromptTemplateID: workitem.PromptTemplatePlan})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_env/plan-drafts":
			var req protocol.SubmitDraftPlanRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode draft: %v", err)
			}
			if req.WorkItemID != "wi_env" || req.RunID != "run_env" || req.Body != "Do it." || req.Actor != "agent" {
				t.Fatalf("draft request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.Artifact{ID: "artifact_plan", WorkItemID: req.WorkItemID, Kind: workitem.ArtifactKindPlan, Status: workitem.ArtifactStatusDraft})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_env/approve-plan":
			var req protocol.ApprovePlanRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode approve: %v", err)
			}
			if req.ArtifactID != "artifact_plan" || req.Actor != "agent" {
				t.Fatalf("approve request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.WorkItem{ID: "wi_env", StageID: workitem.StageReady})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_env/start-execution":
			_ = json.NewEncoder(w).Encode(protocol.WorkItemRun{ID: "run_exec", WorkItemID: "wi_env", PromptTemplateID: workitem.PromptTemplateImplement})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/questions":
			var req protocol.AskQuestionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode question: %v", err)
			}
			if req.WorkItemID != "wi_env" || req.RunID != "run_env" || req.Prompt != "Which key?" {
				t.Fatalf("question request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.Question{ID: "question_01", WorkItemID: req.WorkItemID, RunID: req.RunID, Prompt: req.Prompt, Status: workitem.QuestionStatusOpen})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/questions/question_01/answer":
			var req protocol.AnswerQuestionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode answer: %v", err)
			}
			if req.Answer != "Staging." {
				t.Fatalf("answer request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.Question{ID: "question_01", Answer: req.Answer, Status: workitem.QuestionStatusAnswered})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/work-items/wi_env/review-feedback":
			var req protocol.SubmitReviewFeedbackRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode feedback: %v", err)
			}
			if req.RunID != "run_env" || req.Body != "Fix validation." {
				t.Fatalf("feedback request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(protocol.Artifact{ID: "feedback_01", Kind: workitem.ArtifactKindFeedback, Status: workitem.ArtifactStatusApproved})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	t.Setenv("WHISKD_URL", server.URL)

	cases := [][]string{
		{"workflow", "start-planning", "-json"},
		{"workflow", "submit-plan", "-body", "Do it.", "-json"},
		{"workflow", "approve-plan", "-artifact", "artifact_plan", "-json"},
		{"workflow", "start-execution", "-json"},
		{"question", "ask", "-prompt", "Which key?", "-json"},
		{"question", "answer", "question_01", "-answer", "Staging.", "-json"},
		{"workflow", "feedback", "-body", "Fix validation.", "-json"},
	}
	for _, args := range cases {
		if _, err := captureStdout(func() error { return run(args) }); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if requests["POST /v1/work-items/wi_env/start-planning"] != 1 ||
		requests["POST /v1/questions"] != 1 ||
		requests["POST /v1/work-items/wi_env/review-feedback"] != 1 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestRunStatusRejectsMissingContext(t *testing.T) {
	if err := run([]string{"status", "question", "-message", "Need input"}); err == nil {
		t.Fatalf("expected missing context error")
	}
}

func TestRunWorkItemRejectsInvalidUsage(t *testing.T) {
	if err := run([]string{"project", "create"}); err == nil {
		t.Fatalf("expected project create usage error")
	}
	if err := run([]string{"work-item", "create"}); err == nil {
		t.Fatalf("expected work item create usage error")
	}
	if err := run([]string{"work-item", "move", "wi_01"}); err == nil {
		t.Fatalf("expected move usage error")
	}
}

func captureStdout(fn func() error) (string, error) {
	original := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = write
	defer func() {
		os.Stdout = original
	}()
	runErr := fn()
	if err := write.Close(); err != nil && runErr == nil {
		runErr = err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, read); err != nil && runErr == nil {
		runErr = err
	}
	if err := read.Close(); err != nil && runErr == nil {
		runErr = err
	}
	return buf.String(), runErr
}
