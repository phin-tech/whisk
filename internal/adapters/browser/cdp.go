package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
)

type CDPProbe struct {
	client *http.Client
}

func NewCDPProbe(client *http.Client) *CDPProbe {
	if client == nil {
		client = http.DefaultClient
	}
	safeClient := *client
	safeClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return fmt.Errorf("cdp probe refused redirect to %s", req.URL.Redacted())
		}
		return fmt.Errorf("cdp probe refused redirect from %s to %s", via[len(via)-1].URL.Redacted(), req.URL.Redacted())
	}
	return &CDPProbe{client: &safeClient}
}

func (p *CDPProbe) ProbeCDP(ctx context.Context, endpoint string) (domainbrowser.CDPProbeResult, error) {
	endpoint, err := domainbrowser.NormalizeCDPEndpoint(endpoint)
	if err != nil {
		return domainbrowser.CDPProbeResult{}, err
	}

	var version cdpVersionResponse
	if err := p.getJSON(ctx, endpoint+"/json/version", &version); err != nil {
		return domainbrowser.CDPProbeResult{}, fmt.Errorf("read cdp version: %w", err)
	}
	var targets []cdpTargetResponse
	if err := p.getJSON(ctx, endpoint+"/json/list", &targets); err != nil {
		return domainbrowser.CDPProbeResult{}, fmt.Errorf("read cdp targets: %w", err)
	}

	result := domainbrowser.CDPProbeResult{
		Endpoint:             endpoint,
		Browser:              version.Browser,
		ProtocolVersion:      version.ProtocolVersion,
		WebSocketDebuggerURL: version.WebSocketDebuggerURL,
		Targets:              make([]domainbrowser.CDPTarget, 0, len(targets)),
	}
	for _, target := range targets {
		result.Targets = append(result.Targets, domainbrowser.CDPTarget{
			ID:                   target.ID,
			Type:                 target.Type,
			URL:                  target.URL,
			Title:                target.Title,
			WebSocketDebuggerURL: target.WebSocketDebuggerURL,
		})
	}
	return result, nil
}

func (p *CDPProbe) getJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, rawURL)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

type cdpVersionResponse struct {
	Browser              string `json:"Browser"`
	ProtocolVersion      string `json:"Protocol-Version"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type cdpTargetResponse struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	Title                string `json:"title"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}
