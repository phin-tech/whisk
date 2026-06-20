package agenthooklog

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const (
	DefaultMaxFiles = 5
	DefaultMaxBytes = 5 * 1024 * 1024
)

type Paths struct {
	ConfigRoot string
	LogPath    string
}

type Logger struct {
	paths    Paths
	maxFiles int
	maxBytes int64
	opener   func(string) error
}

type Entry struct {
	Timestamp         time.Time      `json:"timestamp"`
	Provider          string         `json:"provider"`
	EventName         string         `json:"eventName"`
	Kind              string         `json:"kind,omitempty"`
	Title             string         `json:"title,omitempty"`
	BridgeID          string         `json:"bridgeId,omitempty"`
	SessionID         string         `json:"sessionId,omitempty"`
	ProviderSessionID string         `json:"providerSessionId,omitempty"`
	PTYID             string         `json:"ptyId,omitempty"`
	CWD               string         `json:"cwd,omitempty"`
	Agent             string         `json:"agent,omitempty"`
	ToolName          string         `json:"toolName,omitempty"`
	Message           string         `json:"message,omitempty"`
	NotificationType  string         `json:"notificationType,omitempty"`
	ElicitationID     string         `json:"elicitationId,omitempty"`
	Action            string         `json:"action,omitempty"`
	Result            string         `json:"result"`
	Options           []EntryOption  `json:"options,omitempty"`
	Answerable        bool           `json:"answerable,omitempty"`
	Raw               map[string]any `json:"raw,omitempty"`
}

type EntryOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Status struct {
	Enabled           bool   `json:"enabled"`
	ClearAfterSession bool   `json:"clearAfterSession"`
	Path              string `json:"path"`
	SizeBytes         int64  `json:"sizeBytes"`
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	root := filepath.Join(home, ".config", "whisk")
	return Paths{
		ConfigRoot: root,
		LogPath:    filepath.Join(root, "agent-hooks", "hooks.jsonl"),
	}, nil
}

func New(paths Paths) *Logger {
	return NewWithOptions(paths, DefaultMaxFiles, DefaultMaxBytes, openPath)
}

func NewWithOptions(paths Paths, maxFiles int, maxBytes int64, opener func(string) error) *Logger {
	if paths.LogPath == "" {
		if paths.ConfigRoot != "" {
			paths.LogPath = filepath.Join(paths.ConfigRoot, "agent-hooks", "hooks.jsonl")
		} else if defaults, err := DefaultPaths(); err == nil {
			paths = defaults
		}
	}
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFiles
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if opener == nil {
		opener = openPath
	}
	return &Logger{paths: paths, maxFiles: maxFiles, maxBytes: maxBytes, opener: opener}
}

func (l *Logger) Path() string {
	return l.paths.LogPath
}

func (l *Logger) Size() (int64, error) {
	info, err := os.Stat(l.paths.LogPath)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (l *Logger) Append(entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	entry.Raw = Redact(entry.Raw)
	if err := os.MkdirAll(filepath.Dir(l.paths.LogPath), 0o755); err != nil {
		return err
	}
	if err := l.rotateIfNeeded(); err != nil {
		return err
	}
	file, err := os.OpenFile(l.paths.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return nil
}

func (l *Logger) Clear() error {
	for _, path := range l.logFiles() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (l *Logger) Open() error {
	if err := os.MkdirAll(filepath.Dir(l.paths.LogPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(l.paths.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return l.opener(l.paths.LogPath)
}

func (l *Logger) rotateIfNeeded() error {
	info, err := os.Stat(l.paths.LogPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() < l.maxBytes {
		return nil
	}
	for idx := l.maxFiles - 1; idx >= 1; idx-- {
		src := l.rotatedPath(idx)
		dst := l.rotatedPath(idx + 1)
		if idx == l.maxFiles-1 {
			_ = os.Remove(dst)
		}
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, dst); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(l.paths.LogPath, l.rotatedPath(1))
}

func (l *Logger) logFiles() []string {
	paths := []string{l.paths.LogPath}
	for idx := 1; idx <= l.maxFiles; idx++ {
		paths = append(paths, l.rotatedPath(idx))
	}
	return paths
}

func (l *Logger) rotatedPath(index int) string {
	ext := filepath.Ext(l.paths.LogPath)
	base := strings.TrimSuffix(l.paths.LogPath, ext)
	return fmt.Sprintf("%s.%d%s", base, index, ext)
}

func Redact(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	redacted := make(map[string]any, len(values))
	for key, value := range values {
		if isSensitiveKey(key) {
			redacted[key] = "[redacted]"
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			redacted[key] = Redact(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if itemMap, ok := item.(map[string]any); ok {
					items = append(items, Redact(itemMap))
				} else {
					items = append(items, item)
				}
			}
			redacted[key] = items
		default:
			redacted[key] = value
		}
	}
	return redacted
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	sensitive := []string{"token", "api_key", "password", "secret", "authorization", "cookie", "credential"}
	return slices.ContainsFunc(sensitive, func(pattern string) bool {
		return strings.Contains(normalized, pattern)
	})
}

func openPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
