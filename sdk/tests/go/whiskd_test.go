package whiskd_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	whiskd "github.com/phin-tech/whisk/sdk/go/whiskd"
)

func TestCompatibilityHandshake(t *testing.T) {
	client := whiskd.New(startDaemon(t))

	compat, err := client.Compatibility(context.Background())
	if err != nil {
		t.Fatalf("compatibility: %v", err)
	}
	if compat.APIVersion < 1 {
		t.Fatalf("api version = %d", compat.APIVersion)
	}
	if compat.GitSHA == "" {
		t.Fatalf("git sha is empty")
	}
}

func TestWorkItemRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := whiskd.New(startDaemon(t))

	project, err := client.CreateProject(ctx, whiskd.CreateProjectRequest{
		Name:    "Go SDK Integration",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID == "" || project.Slug == "" {
		t.Fatalf("project = %#v", project)
	}

	project, err = client.AddProjectAttachment(ctx, whiskd.AddProjectAttachmentRequest{
		ProjectID:        project.ID,
		Kind:             whiskd.AttachmentKindNote,
		Title:            "Context note",
		Note:             "remember this",
		IncludeInContext: true,
	})
	if err != nil {
		t.Fatalf("add project attachment: %v", err)
	}
	if len(project.Attachments) != 1 {
		t.Fatalf("attachments = %#v", project.Attachments)
	}

	projectContext, err := client.GetProjectContext(ctx, project.ID)
	if err != nil {
		t.Fatalf("project context: %v", err)
	}
	if len(projectContext.Items) != 1 || projectContext.Items[0].Content != "remember this" {
		t.Fatalf("context = %#v", projectContext)
	}

	item, err := client.CreateWorkItem(ctx, whiskd.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "hello from go",
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	if item.ProjectID != project.ID || item.Number < 1 {
		t.Fatalf("item = %#v", item)
	}

	items, err := client.ListWorkItems(ctx, project.ID)
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	for _, listed := range items {
		if listed.ID == item.ID {
			return
		}
	}
	t.Fatalf("created item %s not listed: %#v", item.ID, items)
}

func startDaemon(t *testing.T) string {
	t.Helper()
	binary := os.Getenv("WHISKD_BIN")
	if binary == "" {
		t.Skip("WHISKD_BIN not set to a built daemon binary; run via `task sdk:test:go`")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("WHISKD_BIN is not usable: %v", err)
	}

	addr := "127.0.0.1:" + freePort(t)
	state := t.TempDir()
	stateHome := filepath.Join(state, "state")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(state, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(state, "data"))
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(state, "cache"))
	cmd := exec.Command(binary, "daemon", "run", "-addr", addr)
	cmd.Env = append(os.Environ(),
		"WHISKD_ADDR="+addr,
		"XDG_CONFIG_HOME="+filepath.Join(state, "config"),
		"XDG_DATA_HOME="+filepath.Join(state, "data"),
		"XDG_STATE_HOME="+stateHome,
		"XDG_CACHE_HOME="+filepath.Join(state, "cache"),
	)
	output := new(outputBuffer)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
	})

	url := "http://" + addr
	tokenPath := filepath.Join(stateHome, "whisk", "control-token")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Fatalf("daemon exited early:\n%s", output.String())
		}
		token, readErr := os.ReadFile(tokenPath)
		if readErr == nil {
			req, err := http.NewRequest(http.MethodGet, url+"/v1/compat", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return url
				}
			}
		} else {
			resp, err := http.Get(url + "/v1/health")
			if err == nil {
				_ = resp.Body.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon did not become ready:\n%s", output.String())
	return ""
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	return fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
}

type outputBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *outputBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data = append(b.data, data...)
	return len(data), nil
}

func (b *outputBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
