package browser

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const DefaultDebuggingPort = 9222

type CDPTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	Title                string `json:"title"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl,omitempty"`
}

type CDPProbeResult struct {
	Endpoint             string      `json:"endpoint"`
	Browser              string      `json:"browser,omitempty"`
	ProtocolVersion      string      `json:"protocolVersion,omitempty"`
	WebSocketDebuggerURL string      `json:"webSocketDebuggerUrl,omitempty"`
	Targets              []CDPTarget `json:"targets,omitempty"`
}

type LaunchRequest struct {
	ChromePath    string
	UserDataDir   string
	DebuggingPort int
}

type LaunchCommand struct {
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Endpoint string   `json:"endpoint"`
}

func NormalizeCDPEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("cdp url required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse cdp url: %w", err)
	}
	if parsed.Scheme != "http" {
		return "", fmt.Errorf("cdp url must use http")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("cdp url must not include user info")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("cdp url requires host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("cdp url must be an endpoint root, not a path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("cdp url must not include query or fragment")
	}

	host := strings.ToLower(parsed.Hostname())
	if !isLoopbackHost(host) {
		return "", fmt.Errorf("cdp url host must be loopback")
	}
	port := parsed.Port()
	if port == "" {
		return "", fmt.Errorf("cdp url requires explicit port")
	}
	if err := validatePort(port); err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}).String(), nil
}

func BuildChromeLaunchSpec(req LaunchRequest) (LaunchCommand, error) {
	command := strings.TrimSpace(req.ChromePath)
	if command == "" {
		return LaunchCommand{}, fmt.Errorf("chrome path required")
	}
	userDataDir := strings.TrimSpace(req.UserDataDir)
	if userDataDir == "" {
		return LaunchCommand{}, fmt.Errorf("user data dir required")
	}
	if req.DebuggingPort <= 0 || req.DebuggingPort > 65535 {
		return LaunchCommand{}, fmt.Errorf("debugging port must be between 1 and 65535")
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", req.DebuggingPort)
	return LaunchCommand{
		Command: command,
		Args: []string{
			"--remote-debugging-address=127.0.0.1",
			fmt.Sprintf("--remote-debugging-port=%d", req.DebuggingPort),
			"--user-data-dir=" + userDataDir,
			"--no-first-run",
			"--no-default-browser-check",
			"about:blank",
		},
		Endpoint: endpoint,
	}, nil
}

func ProductSecurityGates() []string {
	return []string{
		"Choose attach-only, launch-with-dedicated-profile, or both.",
		"Define the explicit user authorization step before connecting to Chrome CDP.",
		"Decide whether authenticated default profiles are allowed.",
		"Set capture caps for text, HTML, CSS, and screenshots with truncation metadata.",
		"Keep screenshots explicit and default-off.",
		"Exclude cookies, storage, network bodies, and browser logs from capture payloads.",
		"Decide whether captures are durable attachments and how users delete them.",
		"Define daemon restart behavior for browser resources.",
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validatePort(raw string) error {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("cdp url port must be numeric")
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("cdp url port must be between 1 and 65535")
	}
	return nil
}
