package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputSnapshotJSONIncludesTerminalSnapshot(t *testing.T) {
	snapshot := OutputSnapshot{
		PtyID:        "pty_01",
		Offset:       12,
		Output:       "tail",
		OutputBase64: "dGFpbA==",
		TerminalSnapshot: &TerminalSnapshot{
			Offset:             8,
			Cols:               80,
			Rows:               24,
			ViewportAnsi:       "\x1b[H\x1b[2Jhello",
			ScrollbackAnsi:     "old",
			RehydrateSequences: "\x1b[1;6H",
			Modes: TerminalModes{
				CursorVisible:  true,
				BracketedPaste: true,
			},
		},
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal output snapshot: %v", err)
	}
	jsonText := string(data)
	for _, want := range []string{
		`"terminalSnapshot"`,
		`"offset":8`,
		`"viewportAnsi"`,
		`"bracketedPaste":true`,
	} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("json %s missing %s", jsonText, want)
		}
	}
}

func TestPTYStreamSnapshotFrameJSONShape(t *testing.T) {
	frame := PTYStreamFrame{
		Type:  "snapshot",
		PtyID: "pty_01",
		TerminalSnapshot: &TerminalSnapshot{
			Offset:                  9,
			Cols:                    100,
			Rows:                    30,
			ViewportAnsi:            "\x1b[H\x1b[2Jready",
			RehydrateSequences:      "\x1b[1;6H",
			MouseTrackingModes:      []TerminalMouseTrackingMode{TerminalMouseTrackingNormal},
			MouseEncodingModes:      []TerminalMouseEncodingMode{TerminalMouseEncodingSGR},
			Modes:                   TerminalModes{CursorVisible: true, MouseTracking: TerminalMouseTrackingNormal, MouseEncoding: TerminalMouseEncodingSGR},
			RehydrateBeforeViewport: "\x1b[?1049h",
		},
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal pty stream frame: %v", err)
	}
	var decoded PTYStreamFrame
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal pty stream frame: %v", err)
	}
	if decoded.Type != "snapshot" ||
		decoded.TerminalSnapshot == nil ||
		decoded.TerminalSnapshot.Offset != 9 ||
		decoded.TerminalSnapshot.Modes.MouseTracking != TerminalMouseTrackingNormal {
		t.Fatalf("decoded frame = %#v", decoded)
	}
}

func TestPTYInfoJSONIncludesTerminalMetadataAndAgentStatus(t *testing.T) {
	info := PTYInfo{
		ID:                       "pty_01",
		WorkingDir:               "/repo",
		Cols:                     80,
		Rows:                     24,
		Running:                  true,
		Status:                   "running",
		Title:                    "Codex waiting for approval",
		TerminalWorkingDirectory: "file://localhost/repo",
		AgentStatus: &AgentStatus{
			Agent:      "codex",
			Label:      "Codex",
			State:      "waiting",
			Source:     "osc-title",
			Confidence: "fallback",
			Title:      "Codex waiting for approval",
			Advisory:   true,
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal pty info: %v", err)
	}
	jsonText := string(data)
	for _, want := range []string{
		`"title":"Codex waiting for approval"`,
		`"terminalWorkingDirectory":"file://localhost/repo"`,
		`"agentStatus"`,
		`"agent":"codex"`,
		`"advisory":true`,
	} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("json %s missing %s", jsonText, want)
		}
	}

	var decoded PTYInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal pty info: %v", err)
	}
	if decoded.AgentStatus == nil ||
		decoded.AgentStatus.State != "waiting" ||
		!decoded.AgentStatus.Advisory ||
		decoded.TerminalWorkingDirectory != "file://localhost/repo" {
		t.Fatalf("decoded pty info = %#v", decoded)
	}
}
