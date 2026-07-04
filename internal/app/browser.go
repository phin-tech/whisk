package app

import (
	"context"
	"time"

	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
)

const (
	BrowserDiagnosticDisabled = "disabled"
	BrowserDiagnosticOK       = "ok"
	BrowserDiagnosticError    = "error"
)

type BrowserProbeBackend interface {
	ProbeCDP(ctx context.Context, endpoint string) (domainbrowser.CDPProbeResult, error)
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
