package browser

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	DefaultCaptureTextBytes       = 32 * 1024
	DefaultCaptureHTMLBytes       = 64 * 1024
	DefaultCaptureCSSBytes        = 32 * 1024
	DefaultCaptureScreenshotBytes = 1024 * 1024

	MaximumCaptureTextBytes       = 256 * 1024
	MaximumCaptureHTMLBytes       = 512 * 1024
	MaximumCaptureCSSBytes        = 256 * 1024
	MaximumCaptureScreenshotBytes = 4 * 1024 * 1024
)

const maxBrowserIdentifierBytes = 128

const (
	CaptureFieldText       = "text"
	CaptureFieldHTML       = "html"
	CaptureFieldCSS        = "css"
	CaptureFieldScreenshot = "screenshot"
)

type ResourceID string
type TargetID string
type CaptureID string

type TargetType string

const (
	TargetTypePage           TargetType = "page"
	TargetTypeBackgroundPage TargetType = "background_page"
	TargetTypeServiceWorker  TargetType = "service_worker"
	TargetTypeSharedWorker   TargetType = "shared_worker"
	TargetTypeWorker         TargetType = "worker"
	TargetTypeIframe         TargetType = "iframe"
	TargetTypeBrowser        TargetType = "browser"
	TargetTypeWebView        TargetType = "webview"
	TargetTypeOther          TargetType = "other"
)

type TargetStatus string

const (
	TargetStatusUnknown   TargetStatus = "unknown"
	TargetStatusAvailable TargetStatus = "available"
	TargetStatusAttached  TargetStatus = "attached"
	TargetStatusClosed    TargetStatus = "closed"
)

type Resource struct {
	ID        ResourceID `json:"id"`
	Name      string     `json:"name,omitempty"`
	CDPURL    string     `json:"cdpUrl"`
	Connected bool       `json:"connected"`
}

type Target struct {
	ID         TargetID     `json:"id"`
	ResourceID ResourceID   `json:"resourceId"`
	Type       TargetType   `json:"type"`
	Status     TargetStatus `json:"status"`
	URL        string       `json:"url,omitempty"`
	Title      string       `json:"title,omitempty"`
}

type CaptureOptions struct {
	Selector           string `json:"selector"`
	IncludeText        bool   `json:"includeText"`
	IncludeHTML        bool   `json:"includeHtml"`
	IncludeCSS         bool   `json:"includeCss"`
	IncludeScreenshot  bool   `json:"includeScreenshot"`
	MaxTextBytes       int    `json:"maxTextBytes,omitempty"`
	MaxHTMLBytes       int    `json:"maxHtmlBytes,omitempty"`
	MaxCSSBytes        int    `json:"maxCssBytes,omitempty"`
	MaxScreenshotBytes int    `json:"maxScreenshotBytes,omitempty"`
}

type Truncation struct {
	Field         string `json:"field"`
	OriginalBytes int    `json:"originalBytes"`
	KeptBytes     int    `json:"keptBytes"`
}

type CapturedPayload struct {
	Text                  string       `json:"text,omitempty"`
	HTML                  string       `json:"html,omitempty"`
	CSS                   string       `json:"css,omitempty"`
	ScreenshotBase64      string       `json:"screenshotBase64,omitempty"`
	ScreenshotContentType string       `json:"screenshotContentType,omitempty"`
	Truncated             []Truncation `json:"truncated,omitempty"`
}

type CapturePromptBlock struct {
	Title    string
	URL      string
	Selector string
	Payload  CapturedPayload
}

type State struct {
	resources map[ResourceID]Resource
	targets   map[ResourceID]map[TargetID]Target
}

func NewState() *State {
	return &State{
		resources: map[ResourceID]Resource{},
		targets:   map[ResourceID]map[TargetID]Target{},
	}
}

func (s *State) ConnectResource(resource Resource, targets []Target) (Resource, error) {
	s.ensure()

	resource.Connected = true
	normalized, err := NormalizeResource(resource)
	if err != nil {
		return Resource{}, err
	}
	if _, ok := s.resources[normalized.ID]; ok {
		return Resource{}, fmt.Errorf("browser resource %s already exists", normalized.ID)
	}
	for _, existing := range s.resources {
		if existing.CDPURL == normalized.CDPURL {
			return Resource{}, fmt.Errorf("browser resource already connected for cdp url %s", normalized.CDPURL)
		}
	}

	normalizedTargets, err := normalizeTargetsForResource(normalized.ID, targets)
	if err != nil {
		return Resource{}, err
	}
	s.resources[normalized.ID] = normalized
	s.targets[normalized.ID] = normalizedTargets
	return normalized, nil
}

func (s *State) ListResources() []Resource {
	s.ensure()

	out := make([]Resource, 0, len(s.resources))
	for _, resource := range s.resources {
		out = append(out, resource)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListTargets(resourceID ResourceID) ([]Target, error) {
	s.ensure()

	id, err := NormalizeResourceID(string(resourceID))
	if err != nil {
		return nil, err
	}
	if _, ok := s.resources[id]; !ok {
		return nil, fmt.Errorf("browser resource %s not found", id)
	}
	targetsByID := s.targets[id]
	out := make([]Target, 0, len(targetsByID))
	for _, target := range targetsByID {
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *State) ReplaceTargets(resourceID ResourceID, targets []Target) ([]Target, error) {
	s.ensure()

	id, err := NormalizeResourceID(string(resourceID))
	if err != nil {
		return nil, err
	}
	if _, ok := s.resources[id]; !ok {
		return nil, fmt.Errorf("browser resource %s not found", id)
	}
	normalizedTargets, err := normalizeTargetsForResource(id, targets)
	if err != nil {
		return nil, err
	}
	s.targets[id] = normalizedTargets
	return s.ListTargets(id)
}

func (s *State) DisconnectResource(resourceID ResourceID) (Resource, error) {
	s.ensure()

	id, err := NormalizeResourceID(string(resourceID))
	if err != nil {
		return Resource{}, err
	}
	resource, ok := s.resources[id]
	if !ok {
		return Resource{}, fmt.Errorf("browser resource %s not found", id)
	}
	delete(s.resources, id)
	delete(s.targets, id)
	return resource, nil
}

func (s *State) ensure() {
	if s.resources == nil {
		s.resources = map[ResourceID]Resource{}
	}
	if s.targets == nil {
		s.targets = map[ResourceID]map[TargetID]Target{}
	}
}

func NormalizeResourceID(raw string) (ResourceID, error) {
	normalized, err := normalizeBrowserIdentifier(raw, "browser resource id")
	return ResourceID(normalized), err
}

func NormalizeTargetID(raw string) (TargetID, error) {
	normalized, err := normalizeBrowserIdentifier(raw, "browser target id")
	return TargetID(normalized), err
}

func NormalizeCaptureID(raw string) (CaptureID, error) {
	normalized, err := normalizeBrowserIdentifier(raw, "browser capture id")
	return CaptureID(normalized), err
}

func NormalizeResource(resource Resource) (Resource, error) {
	id, err := NormalizeResourceID(string(resource.ID))
	if err != nil {
		return Resource{}, err
	}
	endpoint, err := NormalizeCDPEndpoint(resource.CDPURL)
	if err != nil {
		return Resource{}, err
	}
	resource.ID = id
	resource.Name = strings.TrimSpace(resource.Name)
	resource.CDPURL = endpoint
	return resource, nil
}

func NormalizeTarget(target Target) (Target, error) {
	id, err := NormalizeTargetID(string(target.ID))
	if err != nil {
		return Target{}, err
	}
	resourceID, err := NormalizeResourceID(string(target.ResourceID))
	if err != nil {
		return Target{}, err
	}
	target.ID = id
	target.ResourceID = resourceID
	target.Type = NormalizeTargetType(string(target.Type))
	target.Status = NormalizeTargetStatus(string(target.Status))
	target.URL = strings.TrimSpace(target.URL)
	target.Title = strings.TrimSpace(target.Title)
	return target, nil
}

func NormalizeTargetType(raw string) TargetType {
	switch normalizeTargetToken(raw) {
	case string(TargetTypePage):
		return TargetTypePage
	case string(TargetTypeBackgroundPage):
		return TargetTypeBackgroundPage
	case string(TargetTypeServiceWorker):
		return TargetTypeServiceWorker
	case string(TargetTypeSharedWorker):
		return TargetTypeSharedWorker
	case string(TargetTypeWorker):
		return TargetTypeWorker
	case string(TargetTypeIframe):
		return TargetTypeIframe
	case string(TargetTypeBrowser):
		return TargetTypeBrowser
	case string(TargetTypeWebView):
		return TargetTypeWebView
	default:
		return TargetTypeOther
	}
}

func NormalizeTargetStatus(raw string) TargetStatus {
	switch normalizeTargetToken(raw) {
	case string(TargetStatusAvailable), "active", "detached", "open", "ready":
		return TargetStatusAvailable
	case string(TargetStatusAttached), "selected", "current":
		return TargetStatusAttached
	case string(TargetStatusClosed), "crashed", "unavailable":
		return TargetStatusClosed
	default:
		return TargetStatusUnknown
	}
}

func NormalizeCaptureOptions(options CaptureOptions) (CaptureOptions, error) {
	options.Selector = strings.TrimSpace(options.Selector)
	if options.Selector == "" {
		return CaptureOptions{}, fmt.Errorf("browser capture selector required")
	}
	if strings.ContainsRune(options.Selector, 0) {
		return CaptureOptions{}, fmt.Errorf("browser capture selector must not contain NUL")
	}
	if !options.IncludeText && !options.IncludeHTML && !options.IncludeCSS && !options.IncludeScreenshot {
		options.IncludeText = true
		options.IncludeHTML = true
	}
	options.MaxTextBytes = normalizeCaptureByteCap(options.IncludeText, options.MaxTextBytes, DefaultCaptureTextBytes, MaximumCaptureTextBytes)
	options.MaxHTMLBytes = normalizeCaptureByteCap(options.IncludeHTML, options.MaxHTMLBytes, DefaultCaptureHTMLBytes, MaximumCaptureHTMLBytes)
	options.MaxCSSBytes = normalizeCaptureByteCap(options.IncludeCSS, options.MaxCSSBytes, DefaultCaptureCSSBytes, MaximumCaptureCSSBytes)
	options.MaxScreenshotBytes = normalizeCaptureByteCap(options.IncludeScreenshot, options.MaxScreenshotBytes, DefaultCaptureScreenshotBytes, MaximumCaptureScreenshotBytes)
	return options, nil
}

func ApplyCaptureCaps(payload CapturedPayload, options CaptureOptions) (CapturedPayload, error) {
	options, err := NormalizeCaptureOptions(options)
	if err != nil {
		return CapturedPayload{}, err
	}
	out := CapturedPayload{
		ScreenshotContentType: strings.TrimSpace(payload.ScreenshotContentType),
	}
	if options.IncludeText {
		out.Text, out.Truncated = truncateCaptureField(out.Truncated, CaptureFieldText, payload.Text, options.MaxTextBytes)
	}
	if options.IncludeHTML {
		out.HTML, out.Truncated = truncateCaptureField(out.Truncated, CaptureFieldHTML, payload.HTML, options.MaxHTMLBytes)
	}
	if options.IncludeCSS {
		out.CSS, out.Truncated = truncateCaptureField(out.Truncated, CaptureFieldCSS, payload.CSS, options.MaxCSSBytes)
	}
	if options.IncludeScreenshot {
		out.ScreenshotBase64, out.Truncated = capScreenshotBase64(out.Truncated, payload.ScreenshotBase64, options.MaxScreenshotBytes)
	} else {
		out.ScreenshotContentType = ""
	}
	if out.ScreenshotBase64 == "" {
		out.ScreenshotContentType = ""
	}
	return out, nil
}

func RenderCapturePromptBlock(block CapturePromptBlock) string {
	title := cleanPromptLine(block.Title)
	if title == "" {
		title = "Untitled page"
	}

	var b strings.Builder
	b.WriteString("Browser capture: ")
	b.WriteString(title)
	b.WriteByte('\n')

	if url := cleanPromptLine(block.URL); url != "" {
		b.WriteString("URL: ")
		b.WriteString(url)
		b.WriteByte('\n')
	}
	if selector := cleanPromptLine(block.Selector); selector != "" {
		b.WriteString("Selector: ")
		b.WriteString(selector)
		b.WriteByte('\n')
	}

	writePromptDataSection(&b, "Text", "text", block.Payload.Text, MaximumCaptureTextBytes)
	writePromptDataSection(&b, "HTML excerpt", "html", block.Payload.HTML, MaximumCaptureHTMLBytes)
	writePromptDataSection(&b, "CSS excerpt", "css", block.Payload.CSS, MaximumCaptureCSSBytes)

	if block.Payload.ScreenshotBase64 != "" {
		contentType := cleanPromptLine(block.Payload.ScreenshotContentType)
		if contentType == "" {
			contentType = "unknown content type"
		}
		ensurePromptBlankLine(&b)
		b.WriteString("Screenshot: ")
		b.WriteString(contentType)
		b.WriteString(" capture available (")
		b.WriteString(fmt.Sprintf("%d bytes base64, omitted from prompt block", len(block.Payload.ScreenshotBase64)))
		b.WriteString(")\n")
	}

	if len(block.Payload.Truncated) > 0 {
		ensurePromptBlankLine(&b)
		b.WriteString("Truncated:\n")
		for _, truncation := range block.Payload.Truncated {
			field := cleanPromptLine(truncation.Field)
			if field == "" {
				field = "unknown"
			}
			b.WriteString("- ")
			b.WriteString(field)
			b.WriteString(": kept ")
			b.WriteString(fmt.Sprintf("%d of %d bytes\n", truncation.KeptBytes, truncation.OriginalBytes))
		}
	}

	return b.String()
}

func normalizeTargetsForResource(resourceID ResourceID, targets []Target) (map[TargetID]Target, error) {
	out := map[TargetID]Target{}
	for _, target := range targets {
		normalized, err := NormalizeTarget(target)
		if err != nil {
			return nil, err
		}
		if normalized.ResourceID != resourceID {
			return nil, fmt.Errorf("browser target %s belongs to resource %s, want %s", normalized.ID, normalized.ResourceID, resourceID)
		}
		if _, ok := out[normalized.ID]; ok {
			return nil, fmt.Errorf("browser target %s already exists for resource %s", normalized.ID, resourceID)
		}
		out[normalized.ID] = normalized
	}
	return out, nil
}

func normalizeBrowserIdentifier(raw string, label string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s required", label)
	}
	if len(value) > maxBrowserIdentifierBytes {
		return "", fmt.Errorf("%s must be at most %d bytes", label, maxBrowserIdentifierBytes)
	}
	for i, r := range value {
		if r > 127 || !isBrowserIdentifierRune(r) {
			return "", fmt.Errorf("%s contains invalid character %q", label, r)
		}
		if i == 0 && !isASCIIAlnum(r) {
			return "", fmt.Errorf("%s must start with a letter or digit", label)
		}
	}
	return value, nil
}

func normalizeTargetToken(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return value
}

func normalizeCaptureByteCap(enabled bool, value int, defaultValue int, maxValue int) int {
	if !enabled {
		return 0
	}
	if value <= 0 {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func truncateCaptureField(truncations []Truncation, field string, value string, maxBytes int) (string, []Truncation) {
	truncated, didTruncate := truncateStringBytes(value, maxBytes)
	if !didTruncate {
		return truncated, truncations
	}
	return truncated, append(truncations, Truncation{
		Field:         field,
		OriginalBytes: len(value),
		KeptBytes:     len(truncated),
	})
}

func capScreenshotBase64(truncations []Truncation, value string, maxBytes int) (string, []Truncation) {
	if len(value) <= maxBytes {
		return value, truncations
	}
	return "", append(truncations, Truncation{
		Field:         CaptureFieldScreenshot,
		OriginalBytes: len(value),
		KeptBytes:     0,
	})
}

func truncateStringBytes(value string, maxBytes int) (string, bool) {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if len(value) <= maxBytes {
		return value, false
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(value[:cut]) {
		cut--
	}
	return value[:cut], true
}

func writePromptDataSection(b *strings.Builder, title string, language string, body string, maxBytes int) {
	body = cleanPromptBody(body)
	if body == "" {
		return
	}
	originalBytes := len(body)
	var didTruncate bool
	body, didTruncate = truncateStringBytes(body, maxBytes)
	fence := promptDataFence(body)

	ensurePromptBlankLine(b)
	b.WriteString(title)
	b.WriteString(" (data-only, untrusted captured data, ")
	if didTruncate {
		b.WriteString(fmt.Sprintf("%d of %d bytes", len(body), originalBytes))
	} else {
		b.WriteString(fmt.Sprintf("%d bytes", originalBytes))
	}
	b.WriteString("):\n")
	b.WriteString(fence)
	if language != "" {
		b.WriteString(language)
	}
	b.WriteByte('\n')
	b.WriteString(body)
	b.WriteByte('\n')
	b.WriteString(fence)
	b.WriteByte('\n')
	if didTruncate {
		b.WriteString(fmt.Sprintf("[capture data capped in prompt renderer: omitted %d bytes]\n", originalBytes-len(body)))
	}
}

func promptDataFence(body string) string {
	longestRun := 2
	currentRun := 0
	for _, r := range body {
		if r == '`' {
			currentRun++
			if currentRun > longestRun {
				longestRun = currentRun
			}
			continue
		}
		currentRun = 0
	}
	return strings.Repeat("`", longestRun+1)
}

func ensurePromptBlankLine(b *strings.Builder) {
	if b.Len() == 0 {
		return
	}
	current := b.String()
	if strings.HasSuffix(current, "\n\n") {
		return
	}
	if strings.HasSuffix(current, "\n") {
		b.WriteByte('\n')
		return
	}
	b.WriteString("\n\n")
}

func cleanPromptLine(value string) string {
	return strings.Join(strings.Fields(cleanPromptBody(value)), " ")
}

func cleanPromptBody(value string) string {
	value = strings.ToValidUTF8(value, "")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	var b strings.Builder
	for _, r := range value {
		if r == '\n' || r == '\t' || r >= 32 {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func isBrowserIdentifierRune(r rune) bool {
	return isASCIIAlnum(r) || r == '_' || r == '-' || r == '.'
}

func isASCIIAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
