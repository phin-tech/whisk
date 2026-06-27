package main

import (
	"strings"
	"testing"
)

func TestRenderSDKExposesClientConstructorsAndDTOAliases(t *testing.T) {
	data, err := renderSDK(sdkTemplateData{Packages: []packageExport{
		{
			ImportAlias: "protocol",
			ImportPath:  "github.com/phin-tech/whisk/internal/protocol",
			Types:       []string{"CompatibilityResponse", "CreateProjectRequest"},
			Consts:      []string{"DaemonAPIVersion"},
		},
		{
			ImportAlias: "session",
			ImportPath:  "github.com/phin-tech/whisk/internal/domain/session",
			Types:       []string{"Session"},
		},
	}})
	if err != nil {
		t.Fatalf("render sdk: %v", err)
	}
	source := string(data)
	for _, want := range []string{
		"type Client = daemonclient.HTTPClient",
		"func New(baseURL string) *Client",
		"func NewWithHTTPClient(baseURL string, httpClient *http.Client) *Client",
		"type CreateProjectRequest = protocol.CreateProjectRequest",
		"type Session = session.Session",
		"DaemonAPIVersion = protocol.DaemonAPIVersion",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, source)
		}
	}
}

func TestSDKExportsIncludesProtocolAndDomainTypes(t *testing.T) {
	packages, err := sdkExports()
	if err != nil {
		t.Fatalf("collect sdk exports: %v", err)
	}
	source, err := renderSDK(sdkTemplateData{Packages: packages})
	if err != nil {
		t.Fatalf("render sdk: %v", err)
	}
	for _, want := range []string{
		"type CreateProjectRequest = protocol.CreateProjectRequest",
		"type Project = protocol.Project",
		"type Session = session.Session",
		"type ProjectWorkflow = workitem.ProjectWorkflow",
		"type WorkflowStage = workitem.WorkflowStage",
		"StageDone",
	} {
		if !strings.Contains(string(source), want) {
			t.Fatalf("generated sdk missing %q", want)
		}
	}
	if strings.Contains(string(source), "Bookmark") || strings.Contains(string(source), "ptybookmark") {
		t.Fatalf("generated sdk still exposes bookmark symbols")
	}
	if strings.Contains(string(source), "type State =") {
		t.Fatalf("generated sdk exposes domain state internals")
	}
}
