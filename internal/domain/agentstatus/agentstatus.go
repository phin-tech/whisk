// Package agentstatus classifies untrusted terminal metadata into advisory
// agent status display hints.
//
// Title and output-tail classifications are fallback metadata only. Even a
// StateDone result must not complete work-item runs, resolve approvals or
// questions, mutate workflows, or drive mailbox lifecycle changes.
package agentstatus

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTitleChars      = 1024
	MaxPromptChars     = 200
	MaxOutputTailChars = 4096
)

type Agent string

const (
	AgentUnknown  Agent = "unknown"
	AgentClaude   Agent = "claude"
	AgentCodex    Agent = "codex"
	AgentGemini   Agent = "gemini"
	AgentOpenCode Agent = "opencode"
	AgentAider    Agent = "aider"
)

type State string

const (
	StateUnknown State = "unknown"
	StateWorking State = "working"
	StateWaiting State = "waiting"
	StateIdle    State = "idle"
	StateDone    State = "done"
)

type Source string

const (
	SourceUnknown    Source = "unknown"
	SourceBridge     Source = "bridge"
	SourceOSC9999    Source = "osc9999"
	SourceTitle      Source = "osc-title"
	SourceOutputTail Source = "output-tail"
	SourceProcess    Source = "process"
)

type Confidence string

const (
	ConfidenceUnknown  Confidence = "unknown"
	ConfidenceExplicit Confidence = "explicit"
	ConfidenceFallback Confidence = "fallback"
)

type Status struct {
	Agent      Agent
	Label      string
	State      State
	Source     Source
	Confidence Confidence
	Title      string
	Prompt     string
}

const (
	claudeIdlePrefix      = "\u2733"
	geminiWorking         = "\u2726"
	geminiSilentWorking   = "\u23f2"
	geminiIdle            = "\u25c7"
	geminiPermission      = "\u270b"
	truncationMarker      = "..."
	windowsExecutableSuff = ".exe"
)

var knownAgents = []struct {
	agent Agent
	token string
	label string
}{
	{agent: AgentCodex, token: "codex", label: "Codex"},
	{agent: AgentGemini, token: "gemini", label: "Gemini CLI"},
	{agent: AgentOpenCode, token: "opencode", label: "OpenCode"},
	{agent: AgentAider, token: "aider", label: "Aider"},
	{agent: AgentClaude, token: "claude", label: "Claude Code"},
}

func ClassifyTitle(title string) (Status, bool) {
	normalized := NormalizeDisplayField(title, MaxTitleChars)
	if normalized == "" || isClaudeManagementTitle(normalized) {
		return Status{}, false
	}

	agent, label, ok := detectTitleAgent(normalized)
	if !ok {
		return Status{}, false
	}

	state := detectState(normalized, agent)
	return NormalizeStatus(Status{
		Agent:      agent,
		Label:      label,
		State:      state,
		Source:     SourceTitle,
		Confidence: ConfidenceFallback,
		Title:      normalized,
	}), true
}

func ClassifyOutputTail(tail string) (Status, bool) {
	boundedTail := lastRunes(tail, MaxOutputTailChars)
	normalized := NormalizeDisplayField(boundedTail, MaxOutputTailChars)
	if normalized == "" {
		return Status{}, false
	}

	agent, label, ok := detectNamedAgent(normalized)
	if !ok {
		return Status{}, false
	}

	state := detectState(normalized, agent)
	if state == StateUnknown {
		return Status{}, false
	}

	return NormalizeStatus(Status{
		Agent:      agent,
		Label:      label,
		State:      state,
		Source:     SourceOutputTail,
		Confidence: ConfidenceFallback,
		Prompt:     lastNonEmptyLine(boundedTail),
	}), true
}

func SelectStatus(candidates ...Status) (Status, bool) {
	var best Status
	var found bool
	for _, candidate := range candidates {
		normalized := NormalizeStatus(candidate)
		if !isUsefulStatus(normalized) {
			continue
		}
		if !found || sourcePriority(normalized.Source) > sourcePriority(best.Source) {
			best = normalized
			found = true
		}
	}
	return best, found
}

func NormalizeStatus(status Status) Status {
	if status.Agent == "" {
		status.Agent = AgentUnknown
	}
	if status.State == "" {
		status.State = StateUnknown
	}
	if status.Source == "" {
		status.Source = SourceUnknown
	}
	if status.Confidence == "" {
		status.Confidence = defaultConfidence(status.Source)
	}
	if status.Label == "" {
		status.Label = LabelForAgent(status.Agent)
	}
	status.Title = NormalizeDisplayField(status.Title, MaxTitleChars)
	status.Prompt = NormalizeDisplayField(status.Prompt, MaxPromptChars)
	return status
}

func NormalizeDisplayField(value string, maxRunes int) string {
	normalized := stripControlSequences(value)
	normalized = collapseWhitespace(normalized)
	normalized = strings.TrimSpace(normalized)
	return truncateRunes(normalized, maxRunes)
}

func LabelForAgent(agent Agent) string {
	switch agent {
	case AgentClaude:
		return "Claude Code"
	case AgentCodex:
		return "Codex"
	case AgentGemini:
		return "Gemini CLI"
	case AgentOpenCode:
		return "OpenCode"
	case AgentAider:
		return "Aider"
	default:
		return "Unknown"
	}
}

func (status Status) Advisory() bool {
	return status.Source != SourceBridge
}

func detectTitleAgent(title string) (Agent, string, bool) {
	if isClaudePrefixTitle(title) {
		return AgentClaude, LabelForAgent(AgentClaude), true
	}
	if isGeminiSymbolTitle(title) {
		return AgentGemini, LabelForAgent(AgentGemini), true
	}
	return detectNamedAgent(title)
}

func detectNamedAgent(title string) (Agent, string, bool) {
	for _, known := range knownAgents {
		if hasAgentToken(title, known.token) {
			return known.agent, known.label, true
		}
	}
	return AgentUnknown, "", false
}

func detectState(title string, agent Agent) State {
	if agent == AgentGemini {
		switch {
		case strings.Contains(title, geminiPermission):
			return StateWaiting
		case strings.Contains(title, geminiWorking), strings.Contains(title, geminiSilentWorking):
			return StateWorking
		case strings.Contains(title, geminiIdle):
			return StateIdle
		}
	}

	if agent == AgentClaude {
		switch {
		case strings.HasPrefix(title, ". "):
			return StateWorking
		case strings.HasPrefix(title, "* "):
			return StateIdle
		case title == claudeIdlePrefix || strings.HasPrefix(title, claudeIdlePrefix+" "):
			return StateIdle
		}
	}

	switch {
	case hasWaitingSignal(title):
		return StateWaiting
	case hasDoneSignal(title):
		return StateDone
	case hasIdleSignal(title):
		return StateIdle
	case hasWorkingSignal(title), containsBrailleSpinner(title):
		return StateWorking
	default:
		return StateUnknown
	}
}

func hasWaitingSignal(title string) bool {
	lower := strings.ToLower(title)
	if strings.Contains(lower, "action required") ||
		strings.Contains(lower, "requires approval") ||
		strings.Contains(lower, "awaiting approval") ||
		strings.Contains(lower, "waiting for input") ||
		strings.Contains(lower, "waiting for permission") {
		return true
	}
	return hasKeywordToken(lower, "permission") ||
		hasKeywordToken(lower, "waiting") ||
		hasKeywordToken(lower, "approval")
}

func hasDoneSignal(title string) bool {
	lower := strings.ToLower(title)
	return hasKeywordToken(lower, "done") ||
		hasKeywordToken(lower, "finished") ||
		hasKeywordToken(lower, "complete") ||
		hasKeywordToken(lower, "completed")
}

func hasIdleSignal(title string) bool {
	lower := strings.ToLower(title)
	return hasKeywordToken(lower, "ready") ||
		hasKeywordToken(lower, "idle")
}

func hasWorkingSignal(title string) bool {
	lower := strings.ToLower(title)
	return hasKeywordToken(lower, "working") ||
		hasKeywordToken(lower, "thinking") ||
		hasKeywordToken(lower, "running")
}

func isClaudePrefixTitle(title string) bool {
	return strings.HasPrefix(title, ". ") ||
		strings.HasPrefix(title, "* ") ||
		title == claudeIdlePrefix ||
		strings.HasPrefix(title, claudeIdlePrefix+" ")
}

func isGeminiSymbolTitle(title string) bool {
	return strings.Contains(title, geminiPermission) ||
		strings.Contains(title, geminiWorking) ||
		strings.Contains(title, geminiSilentWorking) ||
		strings.Contains(title, geminiIdle)
}

func isClaudeManagementTitle(title string) bool {
	fields := strings.Fields(strings.ToLower(title))
	if len(fields) != 2 || fields[1] != "agents" {
		return false
	}
	command := strings.Trim(fields[0], `"'`)
	command = strings.ReplaceAll(command, "\\", "/")
	if idx := strings.LastIndex(command, "/"); idx >= 0 {
		command = command[idx+1:]
	}
	for _, suffix := range []string{"", ".exe", ".cmd", ".bat", ".ps1"} {
		if command == "claude"+suffix {
			return true
		}
	}
	return false
}

func hasAgentToken(value, token string) bool {
	lower := strings.ToLower(value)
	for start := 0; start < len(lower); {
		idx := strings.Index(lower[start:], token)
		if idx < 0 {
			return false
		}
		idx += start
		afterToken := idx + len(token)
		if agentLeftBoundary(lower, idx) {
			if agentRightBoundary(lower, afterToken) {
				return true
			}
			if suffixEnd, ok := windowsExecutableSuffixEnd(lower, afterToken); ok && agentRightBoundary(lower, suffixEnd) {
				return true
			}
		}
		start = idx + 1
	}
	return false
}

func hasKeywordToken(value, keyword string) bool {
	lower := strings.ToLower(value)
	for start := 0; start < len(lower); {
		idx := strings.Index(lower[start:], keyword)
		if idx < 0 {
			return false
		}
		idx += start
		afterKeyword := idx + len(keyword)
		if keywordLeftBoundary(lower, idx) && keywordRightBoundary(lower, afterKeyword) {
			return true
		}
		start = idx + 1
	}
	return false
}

func agentLeftBoundary(value string, idx int) bool {
	return idx == 0 || !isAgentBoundaryBlocked(value[idx-1])
}

func agentRightBoundary(value string, idx int) bool {
	return idx >= len(value) || !isAgentBoundaryBlocked(value[idx])
}

func keywordLeftBoundary(value string, idx int) bool {
	return idx == 0 || !isKeywordLeftBoundaryBlocked(value[idx-1])
}

func keywordRightBoundary(value string, idx int) bool {
	return idx >= len(value) || !isKeywordRightBoundaryBlocked(value[idx])
}

func isAgentBoundaryBlocked(ch byte) bool {
	return isASCIIWord(ch) || ch == '.' || ch == '/' || ch == '\\' || ch == '-'
}

func isKeywordLeftBoundaryBlocked(ch byte) bool {
	return isASCIIWord(ch) || ch == '.' || ch == '/' || ch == '\\' || ch == '-'
}

func isKeywordRightBoundaryBlocked(ch byte) bool {
	return isASCIIWord(ch) || ch == '-'
}

func isASCIIWord(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

func windowsExecutableSuffixEnd(value string, idx int) (int, bool) {
	for _, suffix := range []string{windowsExecutableSuff, ".cmd", ".bat", ".ps1"} {
		if strings.HasPrefix(value[idx:], suffix) {
			return idx + len(suffix), true
		}
	}
	return idx, false
}

func containsBrailleSpinner(value string) bool {
	for _, r := range value {
		if r >= '\u2800' && r <= '\u28ff' {
			return true
		}
	}
	return false
}

func sourcePriority(source Source) int {
	switch source {
	case SourceBridge:
		return 500
	case SourceOSC9999:
		return 400
	case SourceTitle:
		return 300
	case SourceOutputTail:
		return 200
	case SourceProcess:
		return 100
	default:
		return 0
	}
}

func defaultConfidence(source Source) Confidence {
	if source == SourceBridge {
		return ConfidenceExplicit
	}
	if source == SourceUnknown {
		return ConfidenceUnknown
	}
	return ConfidenceFallback
}

func isUsefulStatus(status Status) bool {
	if status.Source != SourceUnknown {
		return true
	}
	if status.Agent != AgentUnknown || status.State != StateUnknown {
		return true
	}
	return status.Title != "" || status.Prompt != ""
}

func stripControlSequences(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == '\x1b' {
			i = skipEscapeSequence(value, i+size)
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			i += size
			continue
		}
		if unicode.IsControl(r) {
			i += size
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func skipEscapeSequence(value string, idx int) int {
	if idx >= len(value) {
		return idx
	}
	switch value[idx] {
	case '[':
		for idx++; idx < len(value); idx++ {
			if value[idx] >= 0x40 && value[idx] <= 0x7e {
				return idx + 1
			}
		}
		return idx
	case ']':
		for idx++; idx < len(value); idx++ {
			if value[idx] == '\a' {
				return idx + 1
			}
			if value[idx] == '\x1b' && idx+1 < len(value) && value[idx+1] == '\\' {
				return idx + 2
			}
		}
		return idx
	default:
		_, size := utf8.DecodeRuneInString(value[idx:])
		return idx + size
	}
}

func collapseWhitespace(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	previousSpace := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			if !previousSpace {
				b.WriteByte(' ')
				previousSpace = true
			}
			continue
		}
		b.WriteRune(r)
		previousSpace = false
	}
	return b.String()
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	if maxRunes <= len(truncationMarker) {
		return firstRunes(value, maxRunes)
	}
	return firstRunes(value, maxRunes-len(truncationMarker)) + truncationMarker
}

func firstRunes(value string, count int) string {
	if count <= 0 {
		return ""
	}
	seen := 0
	for idx := range value {
		if seen == count {
			return value[:idx]
		}
		seen++
	}
	return value
}

func lastRunes(value string, count int) string {
	if count <= 0 {
		return ""
	}
	seen := 0
	for idx := len(value); idx > 0; {
		_, size := utf8.DecodeLastRuneInString(value[:idx])
		idx -= size
		seen++
		if seen == count {
			return value[idx:]
		}
	}
	return value
}

func lastNonEmptyLine(value string) string {
	lines := strings.Split(value, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := NormalizeDisplayField(lines[i], MaxPromptChars)
		if line != "" {
			return line
		}
	}
	return NormalizeDisplayField(value, MaxPromptChars)
}
