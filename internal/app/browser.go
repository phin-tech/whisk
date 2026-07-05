package app

import (
	"context"
	"fmt"
	"time"

	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
)

const (
	BrowserDiagnosticDisabled = "disabled"
	BrowserDiagnosticOK       = "ok"
	BrowserDiagnosticError    = "error"
)

const defaultBrowserConnectTimeout = 5 * time.Second

type BrowserProbeBackend interface {
	ProbeCDP(ctx context.Context, endpoint string) (domainbrowser.CDPProbeResult, error)
}

type BrowserTargetBackend interface {
	ListTargets(ctx context.Context, endpoint string, resourceID domainbrowser.ResourceID) ([]domainbrowser.Target, error)
}

type BrowserResource = domainbrowser.Resource
type BrowserTarget = domainbrowser.Target

type ConnectBrowserResourceRequest struct {
	Name                          string
	CDPURL                        string
	AcknowledgeBrowserControlRisk bool
	Timeout                       time.Duration
}

type BrowserDiagnosticRequest struct {
	CDPURL        string
	Timeout       time.Duration
	ChromePath    string
	UserDataDir   string
	DebuggingPort int
}

type BrowserDiagnostic struct {
	Enabled              bool                         `json:"enabled"`
	Status               string                       `json:"status"`
	CDPURL               string                       `json:"cdpUrl,omitempty"`
	Browser              string                       `json:"browser,omitempty"`
	ProtocolVersion      string                       `json:"protocolVersion,omitempty"`
	TargetCount          int                          `json:"targetCount"`
	Targets              []domainbrowser.CDPTarget    `json:"targets,omitempty"`
	LaunchCommand        *domainbrowser.LaunchCommand `json:"launchCommand,omitempty"`
	ProductSecurityGates []string                     `json:"productSecurityGates,omitempty"`
	Error                string                       `json:"error,omitempty"`
}

type BrowserDiagnosticService struct {
	probe BrowserProbeBackend
}

func NewBrowserDiagnosticService(probe BrowserProbeBackend) *BrowserDiagnosticService {
	return &BrowserDiagnosticService{probe: probe}
}

func (r *Runtime) DiagnoseBrowser(ctx context.Context, req BrowserDiagnosticRequest) (BrowserDiagnostic, error) {
	return NewBrowserDiagnosticService(r.browserProbe).Diagnose(ctx, req)
}

func (r *Runtime) ConnectBrowserResource(ctx context.Context, req ConnectBrowserResourceRequest) (BrowserResource, error) {
	if !req.AcknowledgeBrowserControlRisk {
		return BrowserResource{}, fmt.Errorf("browser control risk acknowledgement required")
	}
	endpoint, err := domainbrowser.NormalizeCDPEndpoint(req.CDPURL)
	if err != nil {
		return BrowserResource{}, err
	}
	if r.browserTargets == nil {
		return BrowserResource{}, fmt.Errorf("browser target backend required")
	}

	resourceID := domainbrowser.ResourceID(r.ids())
	targetCtx := ctx
	cancel := func() {}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultBrowserConnectTimeout
	}
	if timeout > 0 {
		targetCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	targets, err := r.browserTargets.ListTargets(targetCtx, endpoint, resourceID)
	if err != nil {
		return BrowserResource{}, err
	}
	resource := domainbrowser.Resource{
		ID:        resourceID,
		Name:      req.Name,
		CDPURL:    endpoint,
		Connected: true,
	}

	r.mu.Lock()
	connected, err := r.browserResources.ConnectResource(resource, targets)
	r.mu.Unlock()
	if err != nil {
		return BrowserResource{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventBrowserChanged})
	return connected, nil
}

func (r *Runtime) ListBrowserResources(_ context.Context) ([]BrowserResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.browserResources.ListResources(), nil
}

func (r *Runtime) ListBrowserTargets(_ context.Context, resourceID string) ([]BrowserTarget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.browserResources.ListTargets(domainbrowser.ResourceID(resourceID))
}

func (r *Runtime) DisconnectBrowserResource(ctx context.Context, resourceID string) error {
	r.mu.Lock()
	_, err := r.browserResources.DisconnectResource(domainbrowser.ResourceID(resourceID))
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.publish(ctx, RuntimeEvent{Type: EventBrowserChanged})
	return nil
}

func (s *BrowserDiagnosticService) Diagnose(ctx context.Context, req BrowserDiagnosticRequest) (BrowserDiagnostic, error) {
	result := BrowserDiagnostic{
		Status:               BrowserDiagnosticDisabled,
		ProductSecurityGates: domainbrowser.ProductSecurityGates(),
	}
	if req.ChromePath != "" || req.UserDataDir != "" || req.DebuggingPort != 0 {
		launch, err := domainbrowser.BuildChromeLaunchSpec(domainbrowser.LaunchRequest{
			ChromePath:    req.ChromePath,
			UserDataDir:   req.UserDataDir,
			DebuggingPort: req.DebuggingPort,
		})
		if err != nil {
			return BrowserDiagnostic{}, err
		}
		result.LaunchCommand = &launch
	}
	if req.CDPURL == "" {
		result.Error = "explicit -cdp-url required; browser launch remains deferred"
		return result, nil
	}
	endpoint, err := domainbrowser.NormalizeCDPEndpoint(req.CDPURL)
	if err != nil {
		return BrowserDiagnostic{}, err
	}
	result.Enabled = true
	result.CDPURL = endpoint
	if s.probe == nil {
		result.Error = "browser cdp probe backend disabled"
		return result, nil
	}

	probeCtx := ctx
	cancel := func() {}
	if req.Timeout > 0 {
		probeCtx, cancel = context.WithTimeout(ctx, req.Timeout)
	}
	defer cancel()
	probed, err := s.probe.ProbeCDP(probeCtx, endpoint)
	if err != nil {
		result.Status = BrowserDiagnosticError
		result.Error = err.Error()
		return result, nil
	}
	result.Status = BrowserDiagnosticOK
	result.Browser = probed.Browser
	result.ProtocolVersion = probed.ProtocolVersion
	result.Targets = probed.Targets
	result.TargetCount = len(probed.Targets)
	return result, nil
}
