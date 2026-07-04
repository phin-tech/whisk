package browser

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeResourceIDs(t *testing.T) {
	tests := map[string]ResourceID{
		" browser_01 ": "browser_01",
		"browser-01":   "browser-01",
		"browser.01":   "browser.01",
		"Browser01":    "Browser01",
	}

	for raw, want := range tests {
		got, err := NormalizeResourceID(raw)
		if err != nil {
			t.Fatalf("NormalizeResourceID(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("NormalizeResourceID(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeResourceIDsRejectUnsafeValues(t *testing.T) {
	tests := []string{
		"",
		" browser 01 ",
		"browser/01",
		"browser:01",
		"-browser",
		".browser",
		"browseré",
		strings.Repeat("a", 129),
	}

	for _, raw := range tests {
		if _, err := NormalizeResourceID(raw); err == nil {
			t.Fatalf("NormalizeResourceID(%q) succeeded, want error", raw)
		}
	}
}

func TestNormalizeResourceValidatesEndpoint(t *testing.T) {
	got, err := NormalizeResource(Resource{
		ID:        " browser_01 ",
		Name:      " Main Browser ",
		CDPURL:    " http://LOCALHOST:9222/ ",
		Connected: true,
	})
	if err != nil {
		t.Fatalf("NormalizeResource: %v", err)
	}

	want := Resource{
		ID:        "browser_01",
		Name:      "Main Browser",
		CDPURL:    "http://localhost:9222",
		Connected: true,
	}
	if got != want {
		t.Fatalf("NormalizeResource = %#v, want %#v", got, want)
	}
}

func TestNormalizeTargetNormalizesTypeAndStatus(t *testing.T) {
	got, err := NormalizeTarget(Target{
		ID:         " page-1 ",
		ResourceID: " browser_01 ",
		Type:       "Service-Worker",
		Status:     "active",
		URL:        " https://example.test/app ",
		Title:      " Example App ",
	})
	if err != nil {
		t.Fatalf("NormalizeTarget: %v", err)
	}

	want := Target{
		ID:         "page-1",
		ResourceID: "browser_01",
		Type:       TargetTypeServiceWorker,
		Status:     TargetStatusAvailable,
		URL:        "https://example.test/app",
		Title:      "Example App",
	}
	if got != want {
		t.Fatalf("NormalizeTarget = %#v, want %#v", got, want)
	}
}

func TestNormalizeTargetType(t *testing.T) {
	tests := map[string]TargetType{
		"page":            TargetTypePage,
		"background-page": TargetTypeBackgroundPage,
		"service worker":  TargetTypeServiceWorker,
		"shared_worker":   TargetTypeSharedWorker,
		"WORKER":          TargetTypeWorker,
		"iframe":          TargetTypeIframe,
		"browser":         TargetTypeBrowser,
		"webview":         TargetTypeWebView,
		"extension-popup": TargetTypeOther,
		"":                TargetTypeOther,
	}

	for raw, want := range tests {
		if got := NormalizeTargetType(raw); got != want {
			t.Fatalf("NormalizeTargetType(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeTargetStatus(t *testing.T) {
	tests := map[string]TargetStatus{
		"available":  TargetStatusAvailable,
		"active":     TargetStatusAvailable,
		"ATTACHED":   TargetStatusAttached,
		"selected":   TargetStatusAttached,
		"closed":     TargetStatusClosed,
		"detached":   TargetStatusClosed,
		"crashed":    TargetStatusClosed,
		"unexpected": TargetStatusUnknown,
		"":           TargetStatusUnknown,
	}

	for raw, want := range tests {
		if got := NormalizeTargetStatus(raw); got != want {
			t.Fatalf("NormalizeTargetStatus(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeCaptureOptionsDefaultsAndCaps(t *testing.T) {
	got, err := NormalizeCaptureOptions(CaptureOptions{
		Selector:           " #main ",
		IncludeScreenshot:  true,
		MaxTextBytes:       -1,
		MaxHTMLBytes:       MaximumCaptureHTMLBytes + 1,
		MaxCSSBytes:        100,
		MaxScreenshotBytes: MaximumCaptureScreenshotBytes + 1,
	})
	if err != nil {
		t.Fatalf("NormalizeCaptureOptions: %v", err)
	}

	want := CaptureOptions{
		Selector:           "#main",
		IncludeScreenshot:  true,
		MaxTextBytes:       0,
		MaxHTMLBytes:       0,
		MaxCSSBytes:        0,
		MaxScreenshotBytes: MaximumCaptureScreenshotBytes,
	}
	if got != want {
		t.Fatalf("NormalizeCaptureOptions = %#v, want %#v", got, want)
	}

	defaulted, err := NormalizeCaptureOptions(CaptureOptions{Selector: "#main"})
	if err != nil {
		t.Fatalf("NormalizeCaptureOptions default: %v", err)
	}
	if !defaulted.IncludeText || !defaulted.IncludeHTML || defaulted.IncludeCSS || defaulted.IncludeScreenshot {
		t.Fatalf("default includes = %#v, want text/html only", defaulted)
	}
	if defaulted.MaxTextBytes != DefaultCaptureTextBytes || defaulted.MaxHTMLBytes != DefaultCaptureHTMLBytes {
		t.Fatalf("default caps = %#v", defaulted)
	}
}

func TestNormalizeCaptureOptionsRejectsInvalidSelector(t *testing.T) {
	tests := []CaptureOptions{
		{},
		{Selector: " \t "},
		{Selector: "#main\x00"},
	}

	for _, options := range tests {
		if _, err := NormalizeCaptureOptions(options); err == nil {
			t.Fatalf("NormalizeCaptureOptions(%#v) succeeded, want error", options)
		}
	}
}

func TestApplyCaptureCapsTruncatesUTF8AndRecordsMetadata(t *testing.T) {
	got, err := ApplyCaptureCaps(CapturedPayload{
		Text:                  "hello🙂world",
		HTML:                  "<main>content</main>",
		CSS:                   "#main { color: red; }",
		ScreenshotBase64:      "abcdef",
		ScreenshotContentType: " image/png ",
	}, CaptureOptions{
		Selector:           "#main",
		IncludeText:        true,
		IncludeHTML:        true,
		IncludeScreenshot:  true,
		MaxTextBytes:       8,
		MaxHTMLBytes:       6,
		MaxScreenshotBytes: 4,
	})
	if err != nil {
		t.Fatalf("ApplyCaptureCaps: %v", err)
	}

	want := CapturedPayload{
		Text:                  "hello",
		HTML:                  "<main>",
		ScreenshotBase64:      "abcd",
		ScreenshotContentType: "image/png",
		Truncated: []Truncation{
			{Field: CaptureFieldText, OriginalBytes: len("hello🙂world"), KeptBytes: len("hello")},
			{Field: CaptureFieldHTML, OriginalBytes: len("<main>content</main>"), KeptBytes: len("<main>")},
			{Field: CaptureFieldScreenshot, OriginalBytes: len("abcdef"), KeptBytes: len("abcd")},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ApplyCaptureCaps = %#v, want %#v", got, want)
	}
}

func TestApplyCaptureCapsDropsDisabledPayloadFields(t *testing.T) {
	got, err := ApplyCaptureCaps(CapturedPayload{
		Text:                  "visible",
		HTML:                  "<main>visible</main>",
		CSS:                   "#main {}",
		ScreenshotBase64:      "abcdef",
		ScreenshotContentType: "image/png",
	}, CaptureOptions{
		Selector:    "#main",
		IncludeText: true,
	})
	if err != nil {
		t.Fatalf("ApplyCaptureCaps: %v", err)
	}

	want := CapturedPayload{Text: "visible"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ApplyCaptureCaps = %#v, want %#v", got, want)
	}
}

func TestRenderCapturePromptBlockSanitizesAndOmitsScreenshotData(t *testing.T) {
	got := RenderCapturePromptBlock(CapturePromptBlock{
		Title:    " Demo\r\nPage\x00 ",
		URL:      "https://example.test/app\r\nnext",
		Selector: " #main\x00 ",
		Payload: CapturedPayload{
			Text:                  "hello\r\nworld\x00",
			HTML:                  "<main>ok</main>",
			ScreenshotBase64:      "abcdef",
			ScreenshotContentType: "image/png",
			Truncated: []Truncation{
				{Field: CaptureFieldHTML, OriginalBytes: 20, KeptBytes: 10},
			},
		},
	})

	want := "Browser capture: Demo Page\n" +
		"URL: https://example.test/app next\n" +
		"Selector: #main\n" +
		"\n" +
		"Text:\n" +
		"hello\n" +
		"world\n" +
		"\n" +
		"HTML excerpt:\n" +
		"<main>ok</main>\n" +
		"\n" +
		"Screenshot: image/png capture available (6 bytes base64, omitted from prompt block)\n" +
		"\n" +
		"Truncated:\n" +
		"- html: kept 10 of 20 bytes\n"
	if got != want {
		t.Fatalf("RenderCapturePromptBlock =\n%q\nwant\n%q", got, want)
	}
}
