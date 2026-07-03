package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/controlauth"
)

func TestHTTPClientAddsControlBearerTokenFromStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	token, err := controlauth.EnsureToken()
	if err != nil {
		t.Fatalf("ensure token: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer "+token; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"apiVersion":1,"gitSha":"abc","version":"dev","dirty":false}`)
	}))
	defer server.Close()

	daemon := client.NewHTTP(server.URL, server.Client())
	if _, err := daemon.Compatibility(context.Background()); err != nil {
		t.Fatalf("compatibility: %v", err)
	}
}

func TestHTTPClientAddsExplicitControlBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer explicit-secret"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"apiVersion":1,"gitSha":"abc","version":"dev","dirty":false}`)
	}))
	defer server.Close()

	daemon := client.NewHTTP(server.URL, server.Client(), client.WithControlToken("explicit-secret"))
	if _, err := daemon.Compatibility(context.Background()); err != nil {
		t.Fatalf("compatibility: %v", err)
	}
}

func TestHTTPClientHealthWorksWithoutTokenFile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("authorization = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	daemon := client.NewHTTP(server.URL, server.Client())
	if err := daemon.Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
}
