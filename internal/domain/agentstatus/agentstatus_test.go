package agentstatus

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestClassifyTitleKnownAgentStates(t *testing.T) {
	tests := []struct {
		name  string
		title string
		agent Agent
		label string
		state State
	}{
		{
			name:  "codex working",
			title: "Codex working",
			agent: AgentCodex,
			label: "Codex",
			state: StateWorking,
		},
		{
			name:  "codex ready",
			title: "Codex ready",
			agent: AgentCodex,
			label: "Codex",
			state: StateIdle,
		},
		{
			name:  "codex done",
			title: "Codex done",
			agent: AgentCodex,
			label: "Codex",
			state: StateDone,
		},
		{
			name:  "claude working prefix",
			title: ". edit the daemon watcher",
			agent: AgentClaude,
			label: "Claude Code",
			state: StateWorking,
		},
		{
			name:  "claude star idle prefix",
			title: "* edit the daemon watcher",
			agent: AgentClaude,
			label: "Claude Code",
			state: StateIdle,
		},
		{
			name:  "claude idle glyph prefix",
			title: "\u2733 edit the daemon watcher",
			agent: AgentClaude,
			label: "Claude Code",
			state: StateIdle,
		},
		{
			name:  "gemini working glyph",
			title: "\u2726 Gemini CLI",
			agent: AgentGemini,
			label: "Gemini CLI",
			state: StateWorking,
		},
		{
			name:  "gemini idle glyph",
			title: "\u25c7 Gemini CLI",
			agent: AgentGemini,
			label: "Gemini CLI",
			state: StateIdle,
		},
		{
			name:  "opencode working",
			title: "OpenCode running",
			agent: AgentOpenCode,
			label: "OpenCode",
			state: StateWorking,
		},
		{
			name:  "aider idle",
			title: "Aider idle",
			agent: AgentAider,
			label: "Aider",
			state: StateIdle,
		},
		{
			name:  "known agent unknown state",
			title: "Codex",
			agent: AgentCodex,
			label: "Codex",
			state: StateUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ClassifyTitle(test.title)
			if !ok {
				t.Fatalf("ClassifyTitle(%q) did not classify", test.title)
			}
			assertStatus(t, got, Status{
				Agent:      test.agent,
				Label:      test.label,
				State:      test.state,
				Source:     SourceTitle,
				Confidence: ConfidenceFallback,
				Title:      NormalizeDisplayField(test.title, MaxTitleChars),
			})
			if !got.Advisory() {
				t.Fatalf("title status should be advisory")
			}
		})
	}
}

func TestClassifyTitleWaitingSignals(t *testing.T) {
	tests := []struct {
		name  string
		title string
		agent Agent
	}{
		{name: "codex permission", title: "Codex permission required", agent: AgentCodex},
		{name: "claude action required", title: "Claude Code action required", agent: AgentClaude},
		{name: "gemini permission glyph", title: "\u270b Gemini CLI", agent: AgentGemini},
		{name: "opencode waiting", title: "OpenCode waiting for input", agent: AgentOpenCode},
		{name: "aider approval", title: "Aider awaiting approval", agent: AgentAider},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ClassifyTitle(test.title)
			if !ok {
				t.Fatalf("ClassifyTitle(%q) did not classify", test.title)
			}
			if got.Agent != test.agent {
				t.Fatalf("agent = %q, want %q", got.Agent, test.agent)
			}
			if got.State != StateWaiting {
				t.Fatalf("state = %q, want %q", got.State, StateWaiting)
			}
		})
	}
}

func TestClassifyOutputTail(t *testing.T) {
	tests := []struct {
		name  string
		tail  string
		agent Agent
		state State
	}{
		{
			name:  "codex waiting",
			tail:  "diff generated\nCodex waiting for permission",
			agent: AgentCodex,
			state: StateWaiting,
		},
		{
			name:  "claude working",
			tail:  "Claude Code is thinking about the next edit",
			agent: AgentClaude,
			state: StateWorking,
		},
		{
			name:  "gemini done",
			tail:  "Gemini CLI done",
			agent: AgentGemini,
			state: StateDone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ClassifyOutputTail(test.tail)
			if !ok {
				t.Fatalf("ClassifyOutputTail(%q) did not classify", test.tail)
			}
			if got.Agent != test.agent {
				t.Fatalf("agent = %q, want %q", got.Agent, test.agent)
			}
			if got.State != test.state {
				t.Fatalf("state = %q, want %q", got.State, test.state)
			}
			if got.Source != SourceOutputTail {
				t.Fatalf("source = %q, want %q", got.Source, SourceOutputTail)
			}
			if got.Confidence != ConfidenceFallback {
				t.Fatalf("confidence = %q, want %q", got.Confidence, ConfidenceFallback)
			}
			if got.Prompt == "" {
				t.Fatalf("prompt preview should be populated from the output tail")
			}
			if !got.Advisory() {
				t.Fatalf("output-tail status should be advisory")
			}
		})
	}
}

func TestClassifyOutputTailRequiresAgentAndStateOnSameLine(t *testing.T) {
	tail := strings.Join([]string{
		"Claude Code transcript header",
		"tool output complete",
		"waiting for another process",
	}, "\n")

	if got, ok := ClassifyOutputTail(tail); ok {
		t.Fatalf("ClassifyOutputTail classified unrelated cross-line tokens as %+v, want no classification", got)
	}
}

func TestClassifyTitleRejectsTokenBoundaryFalsePositives(t *testing.T) {
	titles := []string{
		"~/codex/working",
		`C:\codex\ready`,
		"codex-ready-cap",
		"opencode-blinker working",
		"already",
		"reworking",
		"is-thinking-cap",
		"my-aider-project done",
		"gemini.generated ready",
	}

	for _, title := range titles {
		if got, ok := ClassifyTitle(title); ok {
			t.Fatalf("ClassifyTitle(%q) = %+v, want no classification", title, got)
		}
	}
}

func TestNormalizeDisplayFieldStripsControlsAndBoundsRunes(t *testing.T) {
	raw := "Codex\x00\x1b[31m working\twith\ncontrols"
	got := NormalizeDisplayField(raw, 24)
	if strings.ContainsAny(got, "\x00\x1b\n\t") {
		t.Fatalf("NormalizeDisplayField kept control characters: %q", got)
	}
	if got != "Codex working with co..." {
		t.Fatalf("NormalizeDisplayField() = %q, want %q", got, "Codex working with co...")
	}
	if utf8.RuneCountInString(got) > 24 {
		t.Fatalf("NormalizeDisplayField length = %d, want <= 24", utf8.RuneCountInString(got))
	}
}

func TestClassifyTitleBoundsLongTitle(t *testing.T) {
	title := "Codex working " + strings.Repeat("x", MaxTitleChars*2)
	got, ok := ClassifyTitle(title)
	if !ok {
		t.Fatalf("ClassifyTitle(long title) did not classify")
	}
	if utf8.RuneCountInString(got.Title) > MaxTitleChars {
		t.Fatalf("title length = %d, want <= %d", utf8.RuneCountInString(got.Title), MaxTitleChars)
	}
	if !strings.HasSuffix(got.Title, "...") {
		t.Fatalf("bounded title should show truncation marker, got %q", got.Title[len(got.Title)-10:])
	}
}

func TestSelectStatusUsesSourcePrecedence(t *testing.T) {
	bridge := Status{
		Agent:      AgentCodex,
		Label:      "Codex",
		State:      StateWorking,
		Source:     SourceBridge,
		Confidence: ConfidenceExplicit,
	}
	title := Status{
		Agent:      AgentCodex,
		Label:      "Codex",
		State:      StateDone,
		Source:     SourceTitle,
		Confidence: ConfidenceFallback,
		Title:      "Codex done",
	}
	tail := Status{
		Agent:      AgentCodex,
		Label:      "Codex",
		State:      StateWaiting,
		Source:     SourceOutputTail,
		Confidence: ConfidenceFallback,
		Prompt:     "Codex waiting for permission",
	}
	osc9999 := Status{
		Agent:      AgentCodex,
		Label:      "Codex",
		State:      StateWaiting,
		Source:     SourceOSC9999,
		Confidence: ConfidenceFallback,
		Title:      "Codex waiting",
	}
	process := Status{
		Agent:      AgentUnknown,
		Label:      "Unknown",
		State:      StateDone,
		Source:     SourceProcess,
		Confidence: ConfidenceFallback,
	}

	got, ok := SelectStatus(tail, title)
	if !ok {
		t.Fatalf("SelectStatus returned no status")
	}
	if got.Source != SourceTitle || got.State != StateDone {
		t.Fatalf("title should win over output tail, got %+v", got)
	}

	got, ok = SelectStatus(process, tail)
	if !ok {
		t.Fatalf("SelectStatus returned no status")
	}
	if got.Source != SourceOutputTail || got.State != StateWaiting {
		t.Fatalf("output tail should win over process, got %+v", got)
	}

	got, ok = SelectStatus(title, osc9999, tail)
	if !ok {
		t.Fatalf("SelectStatus returned no status")
	}
	if got.Source != SourceOSC9999 || got.State != StateWaiting {
		t.Fatalf("osc9999 should win over title and output tail, got %+v", got)
	}

	got, ok = SelectStatus(title, bridge, tail)
	if !ok {
		t.Fatalf("SelectStatus returned no status")
	}
	if got.Source != SourceBridge || got.State != StateWorking {
		t.Fatalf("bridge should win over fallback sources, got %+v", got)
	}
	if got.Advisory() {
		t.Fatalf("bridge status should not be marked advisory by source")
	}
}

func TestSelectStatusIgnoresEmptyHigherPriorityStatus(t *testing.T) {
	tail := Status{
		Agent:      AgentCodex,
		State:      StateWaiting,
		Source:     SourceOutputTail,
		Confidence: ConfidenceFallback,
		Prompt:     "Codex waiting for permission",
	}
	emptyBridge := Status{
		Source: SourceBridge,
	}

	got, ok := SelectStatus(tail, emptyBridge)
	if !ok {
		t.Fatalf("SelectStatus returned no status")
	}
	if got.Source != SourceOutputTail || got.Agent != AgentCodex || got.State != StateWaiting {
		t.Fatalf("empty higher-priority status should not suppress useful fallback, got %+v", got)
	}
}

func assertStatus(t *testing.T, got Status, want Status) {
	t.Helper()
	if got.Agent != want.Agent {
		t.Fatalf("agent = %q, want %q", got.Agent, want.Agent)
	}
	if got.Label != want.Label {
		t.Fatalf("label = %q, want %q", got.Label, want.Label)
	}
	if got.State != want.State {
		t.Fatalf("state = %q, want %q", got.State, want.State)
	}
	if got.Source != want.Source {
		t.Fatalf("source = %q, want %q", got.Source, want.Source)
	}
	if got.Confidence != want.Confidence {
		t.Fatalf("confidence = %q, want %q", got.Confidence, want.Confidence)
	}
	if got.Title != want.Title {
		t.Fatalf("title = %q, want %q", got.Title, want.Title)
	}
}
