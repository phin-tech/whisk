package hooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, command RunCommand) error
}

type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, command RunCommand) error {
	cmd := exec.CommandContext(ctx, "sh", "-lc", command.Command)
	cmd.Dir = command.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook command failed: %w: %s", err, out)
	}
	return nil
}

type Service struct {
	root   string
	runner Runner
}

type Command struct {
	Event      string
	Source     string
	ConfigPath string
	Name       string
	Command    string
}

type RunRequest struct {
	Event        string
	RepoPath     string
	WorktreePath string
}

type RunCommand struct {
	Command string
	Dir     string
}

type RunSummary struct {
	Event string
	Ran   int
}

func NewService(root string, runner Runner) *Service {
	if runner == nil {
		runner = ShellRunner{}
	}
	return &Service{root: root, runner: runner}
}

func (s *Service) Run(ctx context.Context, req RunRequest) (RunSummary, error) {
	if err := validateEvent(req.Event); err != nil {
		return RunSummary{}, err
	}
	commands, err := s.commandsForRun(req)
	if err != nil {
		return RunSummary{}, err
	}
	ran := 0
	for _, command := range commands {
		if command.Event != req.Event {
			continue
		}
		if err := s.runner.Run(ctx, RunCommand{Command: command.Command, Dir: hookCwd(req)}); err != nil {
			return RunSummary{}, err
		}
		ran++
	}
	return RunSummary{Event: req.Event, Ran: ran}, nil
}

func (s *Service) Approve(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("approval id required")
	}
	approvals, err := s.approvals()
	if err != nil {
		return err
	}
	for _, approval := range approvals {
		if approval == id {
			return nil
		}
	}
	approvals = append(approvals, id)
	raw, err := json.MarshalIndent(approvals, "", "  ")
	if err != nil {
		return err
	}
	if err := ensureParent(s.approvalsPath()); err != nil {
		return err
	}
	return writeText(s.approvalsPath(), string(append(raw, '\n')))
}

func (s *Service) commandsForRun(req RunRequest) ([]Command, error) {
	commands, err := s.loadCommands(s.userConfigPath(), "user")
	if err != nil {
		return nil, err
	}
	if req.RepoPath == "" {
		return commands, nil
	}
	projectCommands, err := s.loadCommands(filepath.Join(req.RepoPath, ".config", "whisk", "hooks.toml"), "project")
	if err != nil {
		return nil, err
	}
	approved, err := s.approvedSet()
	if err != nil {
		return nil, err
	}
	for _, command := range projectCommands {
		if approved[ApprovalID(command)] {
			commands = append(commands, command)
		}
	}
	return commands, nil
}

func (s *Service) loadCommands(path string, source string) ([]Command, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Command{}, nil
		}
		return nil, err
	}
	return ParseCommands(string(raw), path, source)
}

func ParseCommands(content string, path string, source string) ([]Command, error) {
	var commands []Command
	currentEvent := ""
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(stripComment(rawLine))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentEvent = strings.TrimSpace(strings.Trim(line, "[]"))
			if err := validateEvent(currentEvent); err != nil {
				currentEvent = ""
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		command, ok := parseString(strings.TrimSpace(value))
		if !ok {
			continue
		}
		event := currentEvent
		name := key
		if event == "" {
			if err := validateEvent(key); err != nil {
				continue
			}
			event = key
			name = "default"
		}
		commands = append(commands, Command{
			Event:      event,
			Source:     source,
			ConfigPath: path,
			Name:       name,
			Command:    command,
		})
	}
	return commands, nil
}

func ApprovalID(command Command) string {
	sum := sha256.Sum256([]byte(command.ConfigPath + "\x00" + command.Event + "\x00" + command.Name + "\x00" + command.Command))
	return hex.EncodeToString(sum[:])
}

func (s *Service) approvedSet() (map[string]bool, error) {
	approvals, err := s.approvals()
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, approval := range approvals {
		out[approval] = true
	}
	return out, nil
}

func (s *Service) approvals() ([]string, error) {
	raw, err := os.ReadFile(s.approvalsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var approvals []string
	if err := json.Unmarshal(raw, &approvals); err != nil {
		return nil, err
	}
	return approvals, nil
}

func (s *Service) userConfigPath() string {
	return filepath.Join(s.root, "hooks.toml")
}

func (s *Service) approvalsPath() string {
	return filepath.Join(s.root, "hook-approvals.json")
}

func hookCwd(req RunRequest) string {
	if req.WorktreePath != "" {
		if info, err := os.Stat(req.WorktreePath); err == nil && info.IsDir() {
			return req.WorktreePath
		}
	}
	return req.RepoPath
}

func validateEvent(event string) error {
	switch event {
	case "pre_agent", "post_agent", "pre_command", "post_command":
		return nil
	default:
		return fmt.Errorf("unknown hook event %q", event)
	}
}

func stripComment(line string) string {
	if before, _, ok := strings.Cut(line, "#"); ok {
		return before
	}
	return line
}

func parseString(value string) (string, bool) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}
	return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`), true
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writeText(path string, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}
