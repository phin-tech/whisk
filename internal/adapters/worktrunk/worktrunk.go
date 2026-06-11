package worktrunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const MinVersion = "0.44.0"

type Binary struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type DetectOptions struct {
	OverridePath string
}

type Command struct {
	Path string
	Args []string
	Dir  string
	Env  map[string]string
}

type Output struct {
	StatusCode int
	Stdout     []byte
	Stderr     []byte
}

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, command Command) (Output, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Run(ctx context.Context, command Command) (Output, error) {
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	if len(command.Env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range command.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	stdout, err := cmd.Output()
	if err == nil {
		return Output{StatusCode: 0, Stdout: stdout}, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return Output{
			StatusCode: exitError.ExitCode(),
			Stdout:     stdout,
			Stderr:     exitError.Stderr,
		}, nil
	}
	return Output{}, err
}

type Client struct {
	binary Binary
	runner Runner
}

type CreateRequest struct {
	RepoPath string
	Branch   string
	Base     string
	Env      map[string]string
}

type RemoveRequest struct {
	RepoPath     string
	WorktreePath string
	AlsoBranch   bool
	Force        bool
	Env          map[string]string
}

type Item struct {
	Branch         string       `json:"branch"`
	Path           string       `json:"path"`
	Kind           string       `json:"kind"`
	Commit         Commit       `json:"commit"`
	WorkingTree    WorkingTree  `json:"working_tree"`
	MainState      string       `json:"main_state"`
	OperationState string       `json:"operation_state"`
	Main           Main         `json:"main"`
	Remote         Remote       `json:"remote"`
	Worktree       WorktreeInfo `json:"worktree"`
	IsMain         bool         `json:"is_main"`
	IsCurrent      bool         `json:"is_current"`
	IsPrevious     bool         `json:"is_previous"`
	URL            string       `json:"url"`
	URLActive      bool         `json:"url_active"`
	StatusLine     string       `json:"statusline"`
	Symbols        []string     `json:"symbols"`
	Vars           []string     `json:"vars"`
	CI             CI           `json:"ci"`
}

type Commit struct {
	Hash    string `json:"hash"`
	Short   string `json:"short"`
	Summary string `json:"summary"`
}

type WorkingTree struct {
	Clean     bool `json:"clean"`
	Dirty     bool `json:"dirty"`
	Untracked int  `json:"untracked"`
	Modified  int  `json:"modified"`
	Deleted   int  `json:"deleted"`
}

func (w *WorkingTree) UnmarshalJSON(data []byte) error {
	var raw struct {
		Clean     bool            `json:"clean"`
		Dirty     bool            `json:"dirty"`
		Untracked json.RawMessage `json:"untracked"`
		Modified  json.RawMessage `json:"modified"`
		Deleted   json.RawMessage `json:"deleted"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	untracked, err := decodeWorktreeCount(raw.Untracked)
	if err != nil {
		return fmt.Errorf("untracked: %w", err)
	}
	modified, err := decodeWorktreeCount(raw.Modified)
	if err != nil {
		return fmt.Errorf("modified: %w", err)
	}
	deleted, err := decodeWorktreeCount(raw.Deleted)
	if err != nil {
		return fmt.Errorf("deleted: %w", err)
	}
	w.Clean = raw.Clean
	w.Dirty = raw.Dirty
	w.Untracked = untracked
	w.Modified = modified
	w.Deleted = deleted
	return nil
}

func decodeWorktreeCount(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil {
		return count, nil
	}
	var changed bool
	if err := json.Unmarshal(raw, &changed); err == nil {
		if changed {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("expected integer or boolean, got %s", string(raw))
}

type Main struct {
	Branch string `json:"branch"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

type Remote struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

type WorktreeInfo struct {
	Locked       bool   `json:"locked"`
	LockReason   string `json:"lock_reason"`
	Prunable     bool   `json:"prunable"`
	PruneReason  string `json:"prune_reason"`
	Bare         bool   `json:"bare"`
	Detached     bool   `json:"detached"`
	GitDir       string `json:"git_dir"`
	WorktreePath string `json:"worktree_path"`
}

type CI struct {
	State string `json:"state"`
	URL   string `json:"url"`
}

type ExitError struct {
	StatusCode int
	Stderr     string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("wt exited with status %d: %s", e.StatusCode, e.Stderr)
}

type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	return "failed to parse wt JSON output: " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

type NotFoundError struct {
	Path string
}

func (e *NotFoundError) Error() string {
	return "no worktree registered at " + e.Path
}

type LockedError struct {
	Reason string
}

func (e *LockedError) Error() string {
	return "worktree is locked: " + e.Reason
}

type DirtyError struct {
	Reason string
}

func (e *DirtyError) Error() string {
	return "worktree has uncommitted changes: " + e.Reason
}

type ProtectedWorktreeError struct {
	Path   string
	Reason string
}

func (e *ProtectedWorktreeError) Error() string {
	return "cannot remove protected worktree " + strconv.Quote(e.Path) + ": " + e.Reason
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func NewClient(binary Binary, runner Runner) *Client {
	if runner == nil {
		runner = OSRunner{}
	}
	return &Client{binary: binary, runner: runner}
}

func Detect(ctx context.Context, runner Runner, opts DetectOptions) (Binary, bool, error) {
	if runner == nil {
		runner = OSRunner{}
	}
	path := strings.TrimSpace(opts.OverridePath)
	if path == "" {
		found, err := runner.LookPath("wt")
		if err != nil || found == "" {
			return Binary{}, false, nil
		}
		path = found
	} else if !pathExists(path) {
		return Binary{}, false, nil
	}
	if isWindowsTerminalAlias(path) {
		return Binary{}, false, nil
	}
	output, err := runner.Run(ctx, Command{Path: path, Args: []string{"--version"}})
	if err != nil || output.StatusCode != 0 {
		return Binary{}, false, err
	}
	version, ok := ParseVersionLine(string(output.Stdout))
	if !ok || compareVersion(version, minVersion()) < 0 {
		return Binary{}, false, nil
	}
	return Binary{Path: path, Version: version.String()}, true, nil
}

func DetectWTConfig(repoPath string) bool {
	info, err := os.Stat(filepath.Join(repoPath, ".config", "wt.toml"))
	return err == nil && !info.IsDir()
}

func ParseVersionLine(line string) (Version, bool) {
	for _, token := range strings.Fields(line) {
		candidate := strings.TrimPrefix(token, "v")
		if version, ok := parseVersion(candidate); ok {
			return version, true
		}
	}
	return Version{}, false
}

func (c *Client) List(ctx context.Context, repoPath string) ([]Item, error) {
	return c.list(ctx, repoPath, true, nil)
}

func (c *Client) Create(ctx context.Context, req CreateRequest) (string, error) {
	if req.Branch == "" {
		return "", fmt.Errorf("branch is required")
	}
	if items, err := c.list(ctx, req.RepoPath, false, req.Env); err == nil {
		if path := findBranchPath(items, req.Branch); path != "" {
			return path, nil
		}
	}
	args := []string{"switch", "--create", "--no-cd"}
	if req.Base != "" {
		args = append(args, "--base", req.Base)
	}
	args = append(args, req.Branch)
	if err := c.runWT(ctx, req.RepoPath, args, req.Env); err != nil {
		if !isBranchAlreadyExistsError(err, req.Branch) {
			return "", err
		}
		if err := c.runWT(ctx, req.RepoPath, []string{"switch", "--no-cd", req.Branch}, req.Env); err != nil {
			return "", err
		}
	}
	items, err := c.list(ctx, req.RepoPath, false, req.Env)
	if err != nil {
		return "", err
	}
	if path := findBranchPath(items, req.Branch); path != "" {
		return path, nil
	}
	return "", &NotFoundError{Path: "wt reported success but no worktree named " + strconv.Quote(req.Branch) + " is listed"}
}

func isBranchAlreadyExistsError(err error, branch string) bool {
	var exitError *ExitError
	if !errors.As(err, &exitError) {
		return false
	}
	lower := strings.ToLower(exitError.Stderr)
	return strings.Contains(lower, "branch") &&
		strings.Contains(lower, strings.ToLower(branch)) &&
		strings.Contains(lower, "already exists")
}

func (c *Client) Remove(ctx context.Context, req RemoveRequest) error {
	items, err := c.list(ctx, req.RepoPath, false, req.Env)
	if err != nil {
		return err
	}
	item, ok := findWorktreePath(items, req.WorktreePath)
	if !ok {
		return &NotFoundError{Path: req.WorktreePath}
	}
	if item.Branch == "" {
		return &NotFoundError{Path: req.WorktreePath + " (detached HEAD, cannot remove via wt)"}
	}
	if item.IsMain {
		return &ProtectedWorktreeError{Path: req.WorktreePath, Reason: "main worktree"}
	}
	if item.IsCurrent {
		return &ProtectedWorktreeError{Path: req.WorktreePath, Reason: "current worktree"}
	}
	args := []string{"remove"}
	if !req.AlsoBranch {
		args = append(args, "--no-delete-branch")
	}
	if req.Force {
		args = append(args, "--force")
	}
	args = append(args, item.Branch)
	err = c.runWT(ctx, req.RepoPath, args, req.Env)
	if err == nil || req.Force {
		return err
	}
	var exitError *ExitError
	if !errors.As(err, &exitError) {
		return err
	}
	lower := strings.ToLower(exitError.Stderr)
	if strings.Contains(lower, "locked") {
		return &LockedError{Reason: exitError.Stderr}
	}
	if strings.Contains(lower, "uncommitted changes") ||
		strings.Contains(lower, "uncommitted change") ||
		strings.Contains(lower, "local changes") ||
		strings.Contains(lower, "local change") ||
		strings.Contains(lower, "dirty") {
		return &DirtyError{Reason: exitError.Stderr}
	}
	return err
}

func (c *Client) list(ctx context.Context, repoPath string, full bool, env map[string]string) ([]Item, error) {
	args := []string{"list"}
	if full {
		args = append(args, "--full")
	}
	args = append(args, "--format=json")
	output, err := c.runner.Run(ctx, Command{
		Path: c.binary.Path,
		Args: args,
		Dir:  repoPath,
		Env:  cloneEnv(env),
	})
	if err != nil {
		return nil, err
	}
	if output.StatusCode != 0 {
		return nil, &ExitError{StatusCode: output.StatusCode, Stderr: string(output.Stderr)}
	}
	var items []Item
	if err := json.Unmarshal(output.Stdout, &items); err != nil {
		return nil, &ParseError{Err: err}
	}
	return items, nil
}

func (c *Client) runWT(ctx context.Context, repoPath string, args []string, env map[string]string) error {
	output, err := c.runner.Run(ctx, Command{
		Path: c.binary.Path,
		Args: append([]string(nil), args...),
		Dir:  repoPath,
		Env:  cloneEnv(env),
	})
	if err != nil {
		return err
	}
	if output.StatusCode != 0 {
		return &ExitError{StatusCode: output.StatusCode, Stderr: string(output.Stderr)}
	}
	return nil
}

func findBranchPath(items []Item, branch string) string {
	for _, item := range items {
		if item.Branch == branch && item.Path != "" {
			return item.Path
		}
	}
	return ""
}

func findWorktreePath(items []Item, worktreePath string) (Item, bool) {
	for _, item := range items {
		if pathsEqual(item.Path, worktreePath) {
			return item, true
		}
	}
	return Item{}, false
}

func pathsEqual(a string, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aEval, aErr := filepath.EvalSymlinks(a)
	if aErr != nil {
		aEval = filepath.Clean(a)
	}
	bEval, bErr := filepath.EvalSymlinks(b)
	if bErr != nil {
		bEval = filepath.Clean(b)
	}
	return aEval == bEval
}

func parseVersion(value string) (Version, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return Version{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, false
	}
	return Version{Major: major, Minor: minor, Patch: patch}, true
}

func compareVersion(a Version, b Version) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	return a.Patch - b.Patch
}

func minVersion() Version {
	version, _ := parseVersion(MinVersion)
	return version
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isWindowsTerminalAlias(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	clean := filepath.Clean(path)
	return strings.EqualFold(filepath.Base(clean), "wt.exe") &&
		strings.EqualFold(filepath.Base(filepath.Dir(clean)), "WindowsApps") &&
		strings.EqualFold(filepath.Base(filepath.Dir(filepath.Dir(clean))), "Microsoft")
}

func cloneEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(env))
	for key, value := range env {
		cloned[key] = value
	}
	return cloned
}
