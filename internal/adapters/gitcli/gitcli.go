package gitcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type CloneRequest struct {
	URL    string
	Branch string
	Target string
}

type RepoStatus struct {
	Branch         string
	TrackingBranch string
	DefaultBranch  string
	Dirty          bool
	Ahead          uint32
	Behind         uint32
	BehindDefault  uint32
	RemoteState    RemoteState
}

type RemoteState string

const (
	RemoteUpToDate RemoteState = "up_to_date"
	RemoteAhead    RemoteState = "ahead"
	RemoteBehind   RemoteState = "behind"
	RemoteDiverged RemoteState = "diverged"
	RemoteUnknown  RemoteState = "unknown"
)

type Worktree struct {
	Path     string
	Branch   string
	Head     string
	Bare     bool
	Detached bool
}

type WorktreeAddRequest struct {
	RepoPath    string
	Path        string
	Branch      string
	StartPoint  string
	ExistingRef bool
}

type WorktreeRemoveRequest struct {
	RepoPath string
	Path     string
	Force    bool
}

type ExitError struct {
	StatusCode int
	Stderr     string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("git exited with status %d: %s", e.StatusCode, e.Stderr)
}

type NotRepoError struct {
	Path string
}

func (e *NotRepoError) Error() string {
	return "not a git repo: " + e.Path
}

func NewClient(binary Binary, runner Runner) *Client {
	if binary.Path == "" {
		binary.Path = "git"
	}
	if runner == nil {
		runner = OSRunner{}
	}
	return &Client{binary: binary, runner: runner}
}

func (c *Client) CloneRepo(ctx context.Context, req CloneRequest) error {
	if req.URL == "" {
		return fmt.Errorf("url required")
	}
	if req.Target == "" {
		return fmt.Errorf("target required")
	}
	if _, err := os.Stat(req.Target); err == nil {
		return fmt.Errorf("target path already exists: %s", req.Target)
	}
	if parent := filepath.Dir(req.Target); parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	args := []string{"clone"}
	if strings.TrimSpace(req.Branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(req.Branch), "--single-branch")
	}
	args = append(args, req.URL, req.Target)
	if err := c.run(ctx, "", args...); err != nil {
		_ = os.RemoveAll(req.Target)
		return err
	}
	return nil
}

func (c *Client) Status(ctx context.Context, repo string) (RepoStatus, error) {
	if _, err := c.output(ctx, repo, "rev-parse", "--is-inside-work-tree"); err != nil {
		return RepoStatus{}, &NotRepoError{Path: repo}
	}
	branch, _ := c.output(ctx, repo, "branch", "--show-current")
	dirtyOut, err := c.output(ctx, repo, "status", "--porcelain")
	if err != nil {
		return RepoStatus{}, err
	}
	status := RepoStatus{
		Branch: strings.TrimSpace(branch),
		Dirty:  strings.TrimSpace(dirtyOut) != "",
	}
	tracking, err := c.output(ctx, repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err == nil && strings.TrimSpace(tracking) != "" {
		status.TrackingBranch = strings.TrimSpace(tracking)
		ahead, behind, err := c.aheadBehind(ctx, repo, "HEAD...@{u}")
		if err != nil {
			return RepoStatus{}, err
		}
		status.Ahead = ahead
		status.Behind = behind
	}
	defaultBranch, _ := c.output(ctx, repo, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	status.DefaultBranch = strings.TrimPrefix(strings.TrimSpace(defaultBranch), "origin/")
	if status.DefaultBranch != "" && status.Branch != "" && status.Branch != status.DefaultBranch {
		behindDefault, err := c.output(ctx, repo, "rev-list", "--count", "HEAD..origin/"+status.DefaultBranch)
		if err == nil {
			if parsed, parseErr := strconv.ParseUint(strings.TrimSpace(behindDefault), 10, 32); parseErr == nil {
				status.BehindDefault = uint32(parsed)
			}
		}
	}
	status.RemoteState = remoteState(status.TrackingBranch != "", status.Ahead, status.Behind)
	return status, nil
}

func (c *Client) WorktreeList(ctx context.Context, repo string) ([]Worktree, error) {
	out, err := c.output(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreePorcelain(out), nil
}

func (c *Client) WorktreeAdd(ctx context.Context, req WorktreeAddRequest) error {
	if req.Path == "" {
		return fmt.Errorf("worktree path required")
	}
	args := []string{"worktree", "add"}
	if req.Branch != "" && !req.ExistingRef {
		args = append(args, "-b", req.Branch)
	}
	args = append(args, req.Path)
	if req.ExistingRef && req.Branch != "" {
		args = append(args, req.Branch)
	} else if req.StartPoint != "" {
		args = append(args, req.StartPoint)
	}
	return c.run(ctx, req.RepoPath, args...)
}

func (c *Client) WorktreeRemove(ctx context.Context, req WorktreeRemoveRequest) error {
	args := []string{"worktree", "remove"}
	if req.Force {
		args = append(args, "--force")
	}
	args = append(args, req.Path)
	return c.run(ctx, req.RepoPath, args...)
}

func (c *Client) aheadBehind(ctx context.Context, repo string, rangeExpr string) (uint32, uint32, error) {
	out, err := c.output(ctx, repo, "rev-list", "--left-right", "--count", rangeExpr)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid ahead/behind output: %q", out)
	}
	ahead, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	behind, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return uint32(ahead), uint32(behind), nil
}

func (c *Client) output(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := c.runner.Run(ctx, Command{Path: c.binary.Path, Args: append([]string(nil), args...), Dir: dir})
	if err != nil {
		return "", err
	}
	if output.StatusCode != 0 {
		return "", &ExitError{StatusCode: output.StatusCode, Stderr: string(output.Stderr)}
	}
	return strings.TrimSpace(string(output.Stdout)), nil
}

func (c *Client) run(ctx context.Context, dir string, args ...string) error {
	_, err := c.output(ctx, dir, args...)
	return err
}

func remoteState(hasTracking bool, ahead uint32, behind uint32) RemoteState {
	switch {
	case !hasTracking:
		return RemoteUnknown
	case ahead == 0 && behind == 0:
		return RemoteUpToDate
	case ahead > 0 && behind == 0:
		return RemoteAhead
	case ahead == 0 && behind > 0:
		return RemoteBehind
	default:
		return RemoteDiverged
	}
}

func parseWorktreePorcelain(out string) []Worktree {
	var worktrees []Worktree
	var current Worktree
	flush := func() {
		if current.Path != "" {
			worktrees = append(worktrees, current)
			current = Worktree{}
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			current.Path = value
		case "HEAD":
			current.Head = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			current.Bare = true
		case "detached":
			current.Detached = true
		}
	}
	flush()
	return worktrees
}
