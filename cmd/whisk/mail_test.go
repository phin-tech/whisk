package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunMailSendUsesEnvDefaultsAndPrintsJSON(t *testing.T) {
	t.Setenv("WHISK_PTY_ID", "pty_env")
	t.Setenv("WHISK_PROJECT_ID", "proj_env")
	t.Setenv("WHISK_WORK_ITEM_ID", "wi_env")
	t.Setenv("WHISK_RUN_ID", "run_env")
	t.Setenv("WHISK_SESSION_ID", "sess_env")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/mail" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.SendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.From.Kind != mailbox.AddressKindPTY || req.From.ID != "pty_env" ||
			len(req.To) != 1 || req.To[0].ID != "run_01" ||
			req.Type != mailbox.TypeDispatch ||
			req.ProjectID != "proj_env" ||
			req.WorkItemID != "wi_env" ||
			req.RunID != "run_env" ||
			req.SessionID != "sess_env" ||
			req.PTYID != "pty_env" {
			t.Fatalf("request = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(protocol.MailMessage{
			ID:       "mail_01",
			From:     req.From,
			Type:     req.Type,
			Priority: req.Priority,
			Subject:  req.Subject,
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"mail", "send", "-url", server.URL, "-to", "run:run_01", "-type", "dispatch", "-subject", "Task", "-body", "Do it", "-json"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var message protocol.MailMessage
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if message.ID != "mail_01" || message.Type != mailbox.TypeDispatch {
		t.Fatalf("message = %#v", message)
	}
}

func TestRunMailListEncodesFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/mail" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("to") != "pty:pty_01,run:run_01" ||
			r.URL.Query().Get("unread") != "true" ||
			r.URL.Query().Get("types") != "status,heartbeat" ||
			r.URL.Query().Get("projectId") != "proj_01" ||
			r.URL.Query().Get("limit") != "5" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]protocol.MailMessage{{ID: "mail_01", Type: mailbox.TypeStatus}})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"mail", "list", "-url", server.URL, "-to", "pty:pty_01,run:run_01", "-unread", "-types", "status,heartbeat", "-project", "proj_01", "-limit", "5", "-json"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var messages []protocol.MailMessage
	if err := json.Unmarshal([]byte(output), &messages); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if len(messages) != 1 || messages[0].ID != "mail_01" {
		t.Fatalf("messages = %#v", messages)
	}
}

func TestRunMailCheckDefaultsToEnvRecipients(t *testing.T) {
	t.Setenv("WHISK_PTY_ID", "pty_env")
	t.Setenv("WHISK_RUN_ID", "run_env")
	t.Setenv("WHISK_SESSION_ID", "sess_env")
	t.Setenv("WHISK_WORK_ITEM_ID", "wi_env")
	t.Setenv("WHISK_PROJECT_ID", "proj_env")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/mail/next" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		to := r.URL.Query().Get("to")
		for _, want := range []string{"pty:pty_env", "run:run_env", "session:sess_env", "work-item:wi_env", "project:proj_env"} {
			if !strings.Contains(to, want) {
				t.Fatalf("to query = %q, missing %q", to, want)
			}
		}
		_ = json.NewEncoder(w).Encode(protocol.NextMailResponse{Timeout: true})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"mail", "check", "-url", server.URL, "-json"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var response protocol.NextMailResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if !response.Timeout {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunMailReadAndReplyUseDaemonAPI(t *testing.T) {
	t.Setenv("WHISK_PTY_ID", "pty_env")
	var readReq protocol.MarkMailReadRequest
	var replyReq protocol.ReplyMailRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/mail/mail_01/read":
			if err := json.NewDecoder(r.Body).Decode(&readReq); err != nil {
				t.Fatalf("decode read: %v", err)
			}
			_ = json.NewEncoder(w).Encode(protocol.MailMessage{ID: "mail_01", Recipients: []protocol.MailRecipient{{Address: protocol.MailAddress{Kind: mailbox.AddressKindPTY, ID: "pty_env"}}}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/mail/mail_01/reply":
			if err := json.NewDecoder(r.Body).Decode(&replyReq); err != nil {
				t.Fatalf("decode reply: %v", err)
			}
			_ = json.NewEncoder(w).Encode(protocol.MailMessage{ID: "mail_02", ReplyToID: "mail_01", From: replyReq.From})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if _, err := captureStdout(func() error {
		return run([]string{"mail", "read", "mail_01", "-url", server.URL, "-json"})
	}); err != nil {
		t.Fatalf("read: %v", err)
	}
	if readReq.To == nil || readReq.To.Kind != mailbox.AddressKindPTY || readReq.To.ID != "pty_env" {
		t.Fatalf("read request = %#v", readReq)
	}
	if _, err := captureStdout(func() error {
		return run([]string{"mail", "reply", "mail_01", "-url", server.URL, "-body", "Done", "-json"})
	}); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if replyReq.From.Kind != mailbox.AddressKindPTY || replyReq.From.ID != "pty_env" || replyReq.Type != mailbox.TypeStatus || replyReq.Body != "Done" {
		t.Fatalf("reply request = %#v", replyReq)
	}
}
