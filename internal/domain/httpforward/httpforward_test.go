package httpforward

import "testing"

func TestStateCreatesListsGetsAndDeletesForwardRecords(t *testing.T) {
	state := NewState()

	created, err := state.Create(CreateRequest{
		ID:        "fwd_01",
		Name:      "difit",
		TargetURL: "http://127.0.0.1:4966",
		SessionID: "session_01",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "fwd_01" || created.Name != "difit" || created.TargetURL != "http://127.0.0.1:4966" || created.SessionID != "session_01" {
		t.Fatalf("created = %#v", created)
	}

	listed := state.List()
	if len(listed) != 1 || listed[0].ID != "fwd_01" {
		t.Fatalf("listed = %#v", listed)
	}
	listed[0].Name = "mutated"
	got, ok := state.Get("fwd_01")
	if !ok || got.Name != "difit" {
		t.Fatalf("get after list mutation = %#v, %v", got, ok)
	}

	if !state.Delete("fwd_01") {
		t.Fatalf("delete returned false")
	}
	if _, ok := state.Get("fwd_01"); ok {
		t.Fatalf("record still exists after delete")
	}
}

func TestStateRejectsInvalidForwardTargets(t *testing.T) {
	state := NewState()
	for _, targetURL := range []string{
		"",
		"://bad",
		"https://127.0.0.1:4966",
		"http://example.com:4966",
		"http://10.0.0.4:4966",
		"http://192.168.1.3:4966",
	} {
		t.Run(targetURL, func(t *testing.T) {
			if _, err := state.Create(CreateRequest{ID: "fwd_01", TargetURL: targetURL}); err == nil {
				t.Fatalf("expected create error for %q", targetURL)
			}
		})
	}
}

func TestProxyPathPreservesTargetBasePath(t *testing.T) {
	path, err := ProxyPath("/ui/base", "/assets/app.js")
	if err != nil {
		t.Fatalf("proxy path: %v", err)
	}
	if path != "/ui/base/assets/app.js" {
		t.Fatalf("path = %q", path)
	}

	root, err := ProxyPath("", "")
	if err != nil {
		t.Fatalf("proxy root: %v", err)
	}
	if root != "/" {
		t.Fatalf("root = %q", root)
	}
}
