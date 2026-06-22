package agentbridge

import "strings"

type EventKind string

const (
	EventKindApproval     EventKind = "approval"
	EventKindQuestion     EventKind = "question"
	EventKindNotification EventKind = "notification"
	EventKindToolResult   EventKind = "tool_result"
	EventKindSession      EventKind = "session"
	EventKindPrompt       EventKind = "prompt"
	EventKindLifecycle    EventKind = "lifecycle"
)

type EventOption struct {
	Label string
	Value string
}

type NormalizedEvent struct {
	Kind              EventKind
	Title             string
	Message           string
	SessionID         string
	ProviderSessionID string
	PTYID             string
	CWD               string
	Agent             string
	Options           []EventOption
	Answerable        bool
}

func NormalizeEvent(event Event) NormalizedEvent {
	whisk := mapField(event.Raw, "whisk")
	normalized := NormalizedEvent{
		Kind:              eventKind(event),
		Message:           firstNonEmptyString(event.Message, questionMessage(event.Raw)),
		SessionID:         firstNonEmptyString(event.SessionID, stringField(whisk, "sessionId")),
		ProviderSessionID: stringField(event.Raw, "session_id"),
		PTYID:             firstNonEmptyString(event.PTYID, stringField(whisk, "ptyId")),
		CWD:               stringField(whisk, "cwd"),
		Agent:             firstNonEmptyString(stringField(whisk, "agent"), string(event.Provider)),
		Options:           firstOptions(eventOptions(event.Raw), questionOptions(event.Raw)),
	}
	normalized.Title = eventTitle(event.Provider, normalized.Kind)
	normalized.Answerable = normalized.Kind == EventKindQuestion &&
		(event.EventName == "Elicitation" || ((event.EventName == "PreToolUse" || event.EventName == "PermissionRequest") && event.ToolName == "AskUserQuestion"))
	return normalized
}

func eventKind(event Event) EventKind {
	if event.ToolName == "AskUserQuestion" {
		return EventKindQuestion
	}
	switch event.EventName {
	case "PreToolUse", "PermissionRequest":
		return EventKindApproval
	case "Elicitation":
		return EventKindQuestion
	case "Notification":
		return EventKindNotification
	case "PostToolUse":
		return EventKindToolResult
	case "SessionStart":
		return EventKindSession
	case "UserPromptSubmit":
		return EventKindPrompt
	default:
		return EventKindLifecycle
	}
}

func eventTitle(provider Provider, kind EventKind) string {
	name := strings.Title(string(provider))
	switch kind {
	case EventKindQuestion:
		return name + " question"
	case EventKindPrompt:
		return name + " prompt"
	case EventKindApproval:
		return name + " approval"
	case EventKindNotification:
		return name + " notification"
	case EventKindToolResult:
		return name + " tool result"
	case EventKindSession:
		return name + " session"
	default:
		return name + " event"
	}
}

func eventOptions(raw map[string]any) []EventOption {
	values, _ := raw["options"].([]any)
	options := make([]EventOption, 0, len(values))
	for _, value := range values {
		object, _ := value.(map[string]any)
		label := stringField(object, "label")
		optionValue := stringField(object, "value")
		if label == "" || optionValue == "" {
			continue
		}
		options = append(options, EventOption{Label: label, Value: optionValue})
	}
	return options
}

func questionMessage(raw map[string]any) string {
	question := firstQuestion(raw)
	return stringField(question, "question")
}

func questionOptions(raw map[string]any) []EventOption {
	question := firstQuestion(raw)
	values, _ := question["options"].([]any)
	options := make([]EventOption, 0, len(values))
	for _, value := range values {
		object, _ := value.(map[string]any)
		label := stringField(object, "label")
		if label == "" {
			continue
		}
		options = append(options, EventOption{Label: label, Value: firstNonEmptyString(stringField(object, "value"), label)})
	}
	return options
}

func firstQuestion(raw map[string]any) map[string]any {
	toolInput := mapField(raw, "tool_input")
	values, _ := toolInput["questions"].([]any)
	if len(values) == 0 {
		return nil
	}
	question, _ := values[0].(map[string]any)
	return question
}

func firstOptions(values ...[]EventOption) []EventOption {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func mapField(raw map[string]any, key string) map[string]any {
	value, _ := raw[key].(map[string]any)
	return value
}

func stringField(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
