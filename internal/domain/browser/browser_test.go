package browser

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCDPEndpointAcceptsLoopbackHTTP(t *testing.T) {
	tests := map[string]string{
		"http://127.0.0.1:9222":    "http://127.0.0.1:9222",
		"http://localhost:9222":    "http://localhost:9222",
		"http://[::1]:9222":        "http://[::1]:9222",
		" http://127.0.0.1:9222/ ": "http://127.0.0.1:9222",
	}

	for raw, want := range tests {
		got, err := NormalizeCDPEndpoint(raw)
		if err != nil {
			t.Fatalf("NormalizeCDPEndpoint(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("NormalizeCDPEndpoint(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeCDPEndpointRejectsUnsafeEndpoints(t *testing.T) {
	tests := []string{
		"",
		"https://127.0.0.1:9222",
		"http://example.com:9222",
		"http://127.0.0.1",
		"http://127.0.0.1:9222/json/version",
		"http://user:pass@127.0.0.1:9222",
	}

	for _, raw := range tests {
		if _, err := NormalizeCDPEndpoint(raw); err == nil {
			t.Fatalf("NormalizeCDPEndpoint(%q) succeeded, want error", raw)
		}
	}
}

func TestBuildChromeLaunchSpecConstructsLoopbackCDPCommand(t *testing.T) {
	spec, err := BuildChromeLaunchSpec(LaunchRequest{
		ChromePath:    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		UserDataDir:   "/tmp/whisk-browser",
		DebuggingPort: 9333,
	})
	if err != nil {
		t.Fatalf("BuildChromeLaunchSpec: %v", err)
	}

	wantArgs := []string{
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=9333",
		"--user-data-dir=/tmp/whisk-browser",
		"--no-first-run",
		"--no-default-browser-check",
		"about:blank",
	}
	if spec.Command != "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
		t.Fatalf("command = %q", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", spec.Args, wantArgs)
	}
	if got, want := spec.Endpoint, "http://127.0.0.1:9333"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestBuildChromeLaunchSpecValidatesInputs(t *testing.T) {
	tests := []LaunchRequest{
		{UserDataDir: "/tmp/whisk-browser", DebuggingPort: 9222},
		{ChromePath: "/chrome", DebuggingPort: 9222},
		{ChromePath: "/chrome", UserDataDir: "/tmp/whisk-browser", DebuggingPort: 0},
		{ChromePath: "/chrome", UserDataDir: "/tmp/whisk-browser", DebuggingPort: 70000},
	}

	for _, req := range tests {
		_, err := BuildChromeLaunchSpec(req)
		if err == nil {
			t.Fatalf("BuildChromeLaunchSpec(%#v) succeeded, want error", req)
		}
		if strings.TrimSpace(err.Error()) == "" {
			t.Fatalf("BuildChromeLaunchSpec(%#v) returned empty error", req)
		}
	}
}
