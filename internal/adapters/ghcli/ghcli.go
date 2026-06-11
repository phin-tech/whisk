package ghcli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Binary struct {
	Path string
}

type Command struct {
	Path string
	Args []string
	Dir  string
}

type Output struct {
	StatusCode int
	Stdout     []byte
	Stderr     []byte
}

type Runner interface {
	Run(ctx context.Context, command Command) (Output, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, command Command) (Output, error) {
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	stdout, err := cmd.Output()
	if err == nil {
		return Output{Stdout: stdout}, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return Output{StatusCode: exit.ExitCode(), Stdout: stdout, Stderr: exit.Stderr}, nil
	}
	return Output{}, err
}

type Client struct {
	binary Binary
	runner Runner
}

type VersionInfo struct {
	Path    string
	Version string
}

type PRViewRequest struct {
	Repo   string
	Number int
	Fields []string
}

type PRListByHeadRequest struct {
	RepoPath string
	Head     string
	Fields   []string
}

type NotAuthenticatedError struct{}

func (*NotAuthenticatedError) Error() string {
	return "gh is not authenticated"
}

type NotFoundError struct{}

func (*NotFoundError) Error() string {
	return "not found"
}

type OtherError struct {
	Message string
}

func (e *OtherError) Error() string {
	return "gh failed: " + e.Message
}

func NewClient(binary Binary, runner Runner) *Client {
	if binary.Path == "" {
		binary.Path = "gh"
	}
	if runner == nil {
		runner = OSRunner{}
	}
	return &Client{binary: binary, runner: runner}
}

func (c *Client) Version(ctx context.Context) (VersionInfo, error) {
	out, err := c.run(ctx, "", "--version")
	if err != nil {
		return VersionInfo{}, err
	}
	fields := strings.Fields(strings.Split(out, "\n")[0])
	if len(fields) < 3 {
		return VersionInfo{}, fmt.Errorf("invalid gh version output: %q", out)
	}
	return VersionInfo{Path: c.binary.Path, Version: fields[2]}, nil
}

func (c *Client) RepoViewNameWithOwner(ctx context.Context, cwd string) (string, error) {
	return c.run(ctx, cwd, "repo", "view", "--json", "nameWithOwner")
}

func (c *Client) PRView(ctx context.Context, req PRViewRequest) (string, error) {
	if req.Repo == "" {
		return "", fmt.Errorf("repo required")
	}
	if req.Number <= 0 {
		return "", fmt.Errorf("number required")
	}
	return c.run(ctx, "", "pr", "view", strconv.Itoa(req.Number), "--repo", req.Repo, "--json", strings.Join(req.Fields, ","))
}

func (c *Client) PRListByHead(ctx context.Context, req PRListByHeadRequest) (string, error) {
	if req.RepoPath == "" {
		return "", fmt.Errorf("repo path required")
	}
	if req.Head == "" {
		return "", fmt.Errorf("head required")
	}
	return c.run(ctx, req.RepoPath, "pr", "list", "--head", req.Head, "--state", "open", "--limit", "1", "--json", strings.Join(req.Fields, ","))
}

func (c *Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := c.runner.Run(ctx, Command{Path: c.binary.Path, Args: append([]string(nil), args...), Dir: dir})
	if err != nil {
		return "", err
	}
	if output.StatusCode != 0 {
		return "", classify(string(output.Stderr))
	}
	return string(output.Stdout), nil
}

func classify(stderr string) error {
	lower := strings.ToLower(stderr)
	switch {
	case strings.Contains(lower, "authentication") || strings.Contains(lower, "gh auth") || strings.Contains(lower, "not logged in"):
		return &NotAuthenticatedError{}
	case strings.Contains(lower, "could not resolve") || strings.Contains(lower, "not found") || strings.Contains(lower, "404"):
		return &NotFoundError{}
	default:
		return &OtherError{Message: truncate(strings.TrimSpace(stderr), 200)}
	}
}

func truncate(value string, limit int) string {
	if len([]rune(value)) <= limit {
		return value
	}
	out := make([]rune, 0, limit)
	for _, r := range value {
		if len(out) == limit {
			break
		}
		out = append(out, r)
	}
	return string(out)
}
