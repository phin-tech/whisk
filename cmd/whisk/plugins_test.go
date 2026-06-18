package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunPluginRegistryListsAvailable(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"github-issues","name":"GitHub Issues","sourceType":"path","installed":false,"trusted":false}]`)
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
	if !strings.Contains(out, "github-issues") {
		t.Fatalf("output = %q", out)
	}
}

func TestRunPluginInstallPostsToRegistry(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"github-issues","name":"GitHub Issues","valid":true,"trusted":false,"dir":"/cfg/plugins/github-issues"}`)
	}))
	defer server.Close()

	out, err := captureStdout(func() error {
		return runPluginInstall([]string{"-json", "-url", server.URL, "github-issues"})
	})
	if err != nil {
		t.Fatalf("runPluginInstall: %v", err)
	}
	if gotPath != "POST /v1/plugin-registry/github-issues/install" {
		t.Fatalf("request = %q", gotPath)
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
