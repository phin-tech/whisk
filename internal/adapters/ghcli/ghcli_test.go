package ghcli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestVersionParsesGhVersion(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{{StatusCode: 0, Stdout: []byte("gh version 2.60.1 (2024-12-11)\n")}}}
	client := NewClient(Binary{Path: "/bin/gh"}, runner)

	version, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("Version error: %v", err)
	}
	if version.Path != "/bin/gh" || version.Version != "2.60.1" {
		t.Fatalf("version = %#v", version)
	}
}

func TestRepoAndPRCommandsUseGhJSONShape(t *testing.T) {
	runner := &fakeRunner{outputs: []Output{
		{StatusCode: 0, Stdout: []byte(`{"nameWithOwner":"phin-tech/roux"}`)},
		{StatusCode: 0, Stdout: []byte(`{"number":7}`)},
		{StatusCode: 0, Stdout: []byte(`[{"number":8}]`)},
	}}
	client := NewClient(Binary{Path: "/bin/gh"}, runner)

	if _, err := client.RepoViewNameWithOwner(context.Background(), "/repo"); err != nil {
		t.Fatalf("RepoView error: %v", err)
	}
	if _, err := client.PRView(context.Background(), PRViewRequest{Repo: "phin-tech/roux", Number: 7, Fields: []string{"number", "title"}}); err != nil {
		t.Fatalf("PRView error: %v", err)
	}
	if _, err := client.PRListByHead(context.Background(), PRListByHeadRequest{RepoPath: "/repo", Head: "feature", Fields: []string{"number"}}); err != nil {
		t.Fatalf("PRListByHead error: %v", err)
	}

	if !reflect.DeepEqual(runner.commands[0].Args, []string{"repo", "view", "--json", "nameWithOwner"}) || runner.commands[0].Dir != "/repo" {
		t.Fatalf("repo args = %#v dir=%q", runner.commands[0].Args, runner.commands[0].Dir)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"pr", "view", "7", "--repo", "phin-tech/roux", "--json", "number,title"}) {
		t.Fatalf("pr view args = %#v", runner.commands[1].Args)
	}
	if !reflect.DeepEqual(runner.commands[2].Args, []string{"pr", "list", "--head", "feature", "--state", "open", "--limit", "1", "--json", "number"}) || runner.commands[2].Dir != "/repo" {
		t.Fatalf("pr list args = %#v dir=%q", runner.commands[2].Args, runner.commands[2].Dir)
	}
}

func TestGhErrorClassification(t *testing.T) {
	tests := []struct {
		stderr string
		check  func(error) bool
	}{
		{stderr: "authentication required", check: func(err error) bool {
			var target *NotAuthenticatedError
			return errors.As(err, &target)
		}},
		{stderr: "HTTP 404: Not Found", check: func(err error) bool {
			var target *NotFoundError
			return errors.As(err, &target)
		}},
		{stderr: strings.Repeat("x", 500), check: func(err error) bool {
			var target *OtherError
			return errors.As(err, &target) && len(target.Message) == 200
		}},
	}
	for _, tt := range tests {
		runner := &fakeRunner{outputs: []Output{{StatusCode: 1, Stderr: []byte(tt.stderr)}}}
		client := NewClient(Binary{Path: "/bin/gh"}, runner)
		_, err := client.RepoViewNameWithOwner(context.Background(), "/repo")
		if !tt.check(err) {
			t.Fatalf("classified %q as %T %[2]v", tt.stderr, err)
		}
	}
}

func TestValidationAndErrorStrings(t *testing.T) {
	client := NewClient(Binary{}, &fakeRunner{})
	if client.binary.Path != "gh" {
		t.Fatalf("default binary = %#v", client.binary)
	}
	if _, err := client.Version(context.Background()); err == nil {
		t.Fatalf("expected invalid version error")
	}
	if _, err := client.PRView(context.Background(), PRViewRequest{}); err == nil {
		t.Fatalf("expected missing repo error")
	}
	if _, err := client.PRView(context.Background(), PRViewRequest{Repo: "owner/repo"}); err == nil {
		t.Fatalf("expected missing number error")
	}
	if _, err := client.PRListByHead(context.Background(), PRListByHeadRequest{}); err == nil {
		t.Fatalf("expected missing repo path error")
	}
	if _, err := client.PRListByHead(context.Background(), PRListByHeadRequest{RepoPath: "/repo"}); err == nil {
		t.Fatalf("expected missing head error")
	}

	if (&NotAuthenticatedError{}).Error() == "" || (&NotFoundError{}).Error() == "" || (&OtherError{Message: "x"}).Error() == "" {
		t.Fatalf("error strings must not be empty")
	}
	if got := truncate("short", 200); got != "short" {
		t.Fatalf("truncate short = %q", got)
	}
}

type fakeRunner struct {
	outputs  []Output
	commands []Command
}

func (r *fakeRunner) Run(_ context.Context, command Command) (Output, error) {
	r.commands = append(r.commands, Command{Path: command.Path, Args: append([]string(nil), command.Args...), Dir: command.Dir})
	if len(r.outputs) == 0 {
		return Output{}, nil
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return output, nil
}
