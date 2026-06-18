package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunPluginRegistryListsAvailable(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"registry":"phin-tech","id":"github-issues","name":"GitHub Issues","sourceType":"path","installed":false,"trusted":false}]`)
	}))
	defer server.Close()

	out, err := captureStdout(func() error {
		return runPluginRegistry([]string{"-json", "-url", server.URL})
	})
	if err != nil {
		t.Fatalf("runPluginRegistry: %v", err)
	}
	if gotPath != "GET /v1/plugin-registry" {
		t.Fatalf("request = %q", gotPath)
	}
	if !strings.Contains(out, "github-issues") || !strings.Contains(out, "phin-tech") {
		t.Fatalf("output = %q", out)
	}
}

func TestRunPluginInstallPostsRegistryAndID(t *testing.T) {
	var gotPath string
	var gotReq protocol.InstallRegistryPluginRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"github-issues","registry":"acme","valid":true,"trusted":false,"dir":"/cfg/plugins/acme/github-issues"}`)
	}))
	defer server.Close()

	// "<id>@<registry>" shorthand routes the registry into the request body.
	out, err := captureStdout(func() error {
		return runPluginInstall([]string{"-json", "-url", server.URL, "github-issues@acme"})
	})
	if err != nil {
		t.Fatalf("runPluginInstall: %v", err)
	}
	if gotPath != "POST /v1/plugin-registry/install" {
		t.Fatalf("request = %q", gotPath)
	}
	if gotReq.Registry != "acme" || gotReq.ID != "github-issues" {
		t.Fatalf("request body = %#v", gotReq)
	}
	// Installed plugins land untrusted.
	if !strings.Contains(out, `"trusted": false`) && !strings.Contains(out, `"trusted":false`) {
		t.Fatalf("output = %q", out)
	}
}

func TestRunPluginInstallRequiresID(t *testing.T) {
	err := runPluginInstall([]string{"-url", "http://127.0.0.1:8787"})
	if err == nil || !strings.Contains(err.Error(), "usage: whisk plugin install") {
		t.Fatalf("err = %v", err)
	}
}
