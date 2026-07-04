package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	UsageResolverStatusPending   = "pending"
	UsageResolverStatusOK        = "ok"
	UsageResolverStatusError     = "error"
	UsageResolverStatusUntrusted = "untrusted"
	UsageResolverStatusInvalid   = "invalid"

	UsageMetricKindUsage     = "usage"
	UsageMetricKindRateLimit = "rateLimit"
)

type RunUsageResolverRequest struct {
	PluginID   string `json:"pluginId"`
	ResolverID string `json:"resolverId"`
	Profile    string `json:"profile,omitempty"`
}

type UsageResolverResult struct {
	Summary   string                `json:"summary,omitempty"`
	Metrics   []UsageResolverMetric `json:"metrics"`
	FetchedAt *time.Time            `json:"fetchedAt,omitempty"`
	Meta      map[string]string     `json:"meta,omitempty"`
}

type UsageResolverMetric struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Label     string     `json:"label,omitempty"`
	Unit      string     `json:"unit,omitempty"`
	Used      *float64   `json:"used,omitempty"`
	Limit     *float64   `json:"limit,omitempty"`
	Remaining *float64   `json:"remaining,omitempty"`
	ResetAt   *time.Time `json:"resetAt,omitempty"`
}

type UsageResolverReadModel struct {
	PluginID     string               `json:"pluginId"`
	ResolverID   string               `json:"resolverId"`
	Provider     string               `json:"provider"`
	Label        string               `json:"label"`
	Profile      string               `json:"profile,omitempty"`
	Trusted      bool                 `json:"trusted"`
	Valid        bool                 `json:"valid"`
	Status       string               `json:"status"`
	Error        string               `json:"error,omitempty"`
	RefreshedAt  *time.Time           `json:"refreshedAt,omitempty"`
	Stale        bool                 `json:"stale,omitempty"`
	MinRefreshMs int                  `json:"minRefreshMs,omitempty"`
	StaleAfterMs int                  `json:"staleAfterMs,omitempty"`
	Result       *UsageResolverResult `json:"result,omitempty"`
}

type UsageResolverRunner interface {
	RunUsageResolver(context.Context, RunUsageResolverRequest) (UsageResolverResult, error)
}

func NormalizeUsageResolverResult(in UsageResolverResult) (UsageResolverResult, error) {
	out := UsageResolverResult{
		Summary: strings.TrimSpace(in.Summary),
		Metrics: make([]UsageResolverMetric, 0, len(in.Metrics)),
	}
	if in.FetchedAt != nil {
		fetchedAt := in.FetchedAt.UTC()
		if fetchedAt.IsZero() {
			return UsageResolverResult{}, fmt.Errorf("fetchedAt must not be zero")
		}
		out.FetchedAt = &fetchedAt
	}
	if len(in.Metrics) == 0 {
		return UsageResolverResult{}, fmt.Errorf("metrics required")
	}
	for i, metric := range in.Metrics {
		normalized, err := normalizeUsageResolverMetric(metric)
		if err != nil {
			return UsageResolverResult{}, fmt.Errorf("metrics[%d]: %w", i, err)
		}
		out.Metrics = append(out.Metrics, normalized)
	}
	if len(in.Meta) > 0 {
		out.Meta = map[string]string{}
		for key, value := range in.Meta {
			key = strings.TrimSpace(key)
			if key == "" {
				return UsageResolverResult{}, fmt.Errorf("meta contains empty key")
			}
			out.Meta[key] = strings.TrimSpace(value)
		}
	}
	return out, nil
}

func normalizeUsageResolverMetric(metric UsageResolverMetric) (UsageResolverMetric, error) {
	out := UsageResolverMetric{
		ID:    strings.TrimSpace(metric.ID),
		Kind:  strings.TrimSpace(metric.Kind),
		Label: strings.TrimSpace(metric.Label),
		Unit:  strings.TrimSpace(metric.Unit),
	}
	if out.ID == "" {
		return UsageResolverMetric{}, fmt.Errorf("id required")
	}
	if out.Kind == "" {
		out.Kind = UsageMetricKindUsage
	}
	switch out.Kind {
	case UsageMetricKindUsage, UsageMetricKindRateLimit:
	default:
		return UsageResolverMetric{}, fmt.Errorf("kind %q is unsupported", out.Kind)
	}
	var numericFields int
	for name, value := range map[string]*float64{
		"used":      metric.Used,
		"limit":     metric.Limit,
		"remaining": metric.Remaining,
	} {
		if value == nil {
			continue
		}
		if math.IsNaN(*value) || math.IsInf(*value, 0) {
			return UsageResolverMetric{}, fmt.Errorf("%s must be finite", name)
		}
		if *value < 0 {
			return UsageResolverMetric{}, fmt.Errorf("%s must be non-negative", name)
		}
		numericFields++
	}
	if numericFields == 0 {
		return UsageResolverMetric{}, fmt.Errorf("used, limit, or remaining required")
	}
	out.Used = copyFloat64(metric.Used)
	out.Limit = copyFloat64(metric.Limit)
	out.Remaining = copyFloat64(metric.Remaining)
	if metric.ResetAt != nil {
		resetAt := metric.ResetAt.UTC()
		if resetAt.IsZero() {
			return UsageResolverMetric{}, fmt.Errorf("resetAt must not be zero")
		}
		out.ResetAt = &resetAt
	}
	return out, nil
}

func copyFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func (r *Runtime) ListUsageResolverResults(ctx context.Context) ([]UsageResolverReadModel, error) {
	plugins, err := r.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}
	cache := r.usageResolverResultsSnapshot()
	return BuildUsageResolverReadModels(plugins, cache, time.Now().UTC()), nil
}

func (r *Runtime) RefreshUsageResolver(ctx context.Context, req RunUsageResolverRequest) (UsageResolverReadModel, error) {
	if r.plugins == nil {
		return UsageResolverReadModel{}, fmt.Errorf("plugins are not configured")
	}
	plugins, err := r.ListPlugins(ctx)
	if err != nil {
		return UsageResolverReadModel{}, err
	}
	resolver, err := findUsageResolver(plugins, req.PluginID, req.ResolverID, req.Profile)
	if err != nil {
		return UsageResolverReadModel{}, err
	}
	if !resolver.Valid {
		return resolver, fmt.Errorf("plugin %s is invalid", resolver.PluginID)
	}
	if !resolver.Trusted {
		return resolver, fmt.Errorf("plugin %s is not trusted", resolver.PluginID)
	}
	runner, ok := r.plugins.(UsageResolverRunner)
	if !ok {
		return UsageResolverReadModel{}, fmt.Errorf("usage resolver execution is not configured")
	}
	runReq := RunUsageResolverRequest{
		PluginID:   resolver.PluginID,
		ResolverID: resolver.ResolverID,
		Profile:    resolver.Profile,
	}
	refreshedAt := time.Now().UTC()
	result, runErr := runner.RunUsageResolver(ctx, runReq)
	if runErr == nil {
		result, runErr = NormalizeUsageResolverResult(result)
	}
	if runErr != nil {
		resolver.Status = UsageResolverStatusError
		resolver.Error = runErr.Error()
		resolver.RefreshedAt = &refreshedAt
		resolver.Result = nil
	} else {
		resolver.Status = UsageResolverStatusOK
		resolver.Error = ""
		resolver.RefreshedAt = &refreshedAt
		resolver.Stale = false
		resolver.Result = &result
	}
	r.cacheUsageResolverResult(resolver)
	r.publish(ctx, RuntimeEvent{Type: EventPluginsChanged})
	return resolver, nil
}

func BuildUsageResolverReadModels(plugins []PluginStatus, cache map[string]UsageResolverReadModel, now time.Time) []UsageResolverReadModel {
	now = now.UTC()
	out := []UsageResolverReadModel{}
	for _, plugin := range plugins {
		for _, resolver := range plugin.UsageResolvers {
			profiles := resolver.Profiles
			if len(profiles) == 0 {
				profiles = []string{""}
			}
			for _, profile := range profiles {
				model := UsageResolverReadModel{
					PluginID:     plugin.ID,
					ResolverID:   resolver.ID,
					Provider:     resolver.Provider,
					Label:        resolver.Label,
					Profile:      strings.TrimSpace(profile),
					Trusted:      plugin.Trusted,
					Valid:        plugin.Valid,
					Status:       UsageResolverStatusPending,
					MinRefreshMs: resolver.MinRefreshMs,
					StaleAfterMs: resolver.StaleAfterMs,
				}
				if !plugin.Valid {
					model.Status = UsageResolverStatusInvalid
					model.Error = plugin.Error
				} else if !plugin.Trusted {
					model.Status = UsageResolverStatusUntrusted
				}
				if cached, ok := cache[usageResolverCacheKey(model.PluginID, model.ResolverID, model.Provider, model.Profile)]; ok && plugin.Valid && plugin.Trusted {
					model.Status = cached.Status
					model.Error = cached.Error
					model.RefreshedAt = copyTime(cached.RefreshedAt)
					model.Result = copyUsageResolverResultPtr(cached.Result)
					model.Stale = usageResolverStale(model.RefreshedAt, model.StaleAfterMs, now)
				}
				out = append(out, model)
			}
		}
	}
	return out
}

func findUsageResolver(plugins []PluginStatus, pluginID, resolverID, profile string) (UsageResolverReadModel, error) {
	pluginID = strings.TrimSpace(pluginID)
	resolverID = strings.TrimSpace(resolverID)
	profile = strings.TrimSpace(profile)
	if pluginID == "" {
		return UsageResolverReadModel{}, fmt.Errorf("pluginId required")
	}
	if resolverID == "" {
		return UsageResolverReadModel{}, fmt.Errorf("resolverId required")
	}
	for _, plugin := range plugins {
		if plugin.ID != pluginID {
			continue
		}
		for _, resolver := range plugin.UsageResolvers {
			if resolver.ID != resolverID {
				continue
			}
			resolvedProfile, err := resolveUsageResolverProfile(resolver, profile)
			if err != nil {
				return UsageResolverReadModel{}, err
			}
			return UsageResolverReadModel{
				PluginID:     plugin.ID,
				ResolverID:   resolver.ID,
				Provider:     resolver.Provider,
				Label:        resolver.Label,
				Profile:      resolvedProfile,
				Trusted:      plugin.Trusted,
				Valid:        plugin.Valid,
				Status:       UsageResolverStatusPending,
				Error:        plugin.Error,
				MinRefreshMs: resolver.MinRefreshMs,
				StaleAfterMs: resolver.StaleAfterMs,
			}, nil
		}
		return UsageResolverReadModel{}, fmt.Errorf("usage resolver %s not found in plugin %s", resolverID, pluginID)
	}
	return UsageResolverReadModel{}, fmt.Errorf("plugin %s not found", pluginID)
}

func resolveUsageResolverProfile(resolver PluginUsageResolver, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if len(resolver.Profiles) == 0 {
		return requested, nil
	}
	if requested == "" {
		if len(resolver.Profiles) == 1 {
			return strings.TrimSpace(resolver.Profiles[0]), nil
		}
		return "", fmt.Errorf("profile required for usage resolver %s", resolver.ID)
	}
	for _, profile := range resolver.Profiles {
		if strings.TrimSpace(profile) == requested {
			return requested, nil
		}
	}
	return "", fmt.Errorf("profile %s is not supported by usage resolver %s", requested, resolver.ID)
}

func (r *Runtime) cacheUsageResolverResult(model UsageResolverReadModel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.usageResolverResults == nil {
		r.usageResolverResults = map[string]UsageResolverReadModel{}
	}
	r.usageResolverResults[usageResolverCacheKey(model.PluginID, model.ResolverID, model.Provider, model.Profile)] = copyUsageResolverReadModel(model)
}

func (r *Runtime) usageResolverResultsSnapshot() map[string]UsageResolverReadModel {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]UsageResolverReadModel{}
	for key, model := range r.usageResolverResults {
		out[key] = copyUsageResolverReadModel(model)
	}
	return out
}

func usageResolverCacheKey(pluginID, resolverID, provider, profile string) string {
	return strings.TrimSpace(pluginID) + "\x00" + strings.TrimSpace(resolverID) + "\x00" + strings.TrimSpace(provider) + "\x00" + strings.TrimSpace(profile)
}

func usageResolverStale(refreshedAt *time.Time, staleAfterMs int, now time.Time) bool {
	if refreshedAt == nil || staleAfterMs <= 0 {
		return false
	}
	return !refreshedAt.Add(time.Duration(staleAfterMs) * time.Millisecond).After(now)
}

func copyUsageResolverReadModel(in UsageResolverReadModel) UsageResolverReadModel {
	out := in
	out.RefreshedAt = copyTime(in.RefreshedAt)
	out.Result = copyUsageResolverResultPtr(in.Result)
	return out
}

func copyUsageResolverResultPtr(in *UsageResolverResult) *UsageResolverResult {
	if in == nil {
		return nil
	}
	out := *in
	out.FetchedAt = copyTime(in.FetchedAt)
	out.Metrics = make([]UsageResolverMetric, 0, len(in.Metrics))
	for _, metric := range in.Metrics {
		copied := metric
		copied.Used = copyFloat64(metric.Used)
		copied.Limit = copyFloat64(metric.Limit)
		copied.Remaining = copyFloat64(metric.Remaining)
		copied.ResetAt = copyTime(metric.ResetAt)
		out.Metrics = append(out.Metrics, copied)
	}
	if in.Meta != nil {
		out.Meta = map[string]string{}
		for key, value := range in.Meta {
			out.Meta[key] = value
		}
	}
	return &out
}

func copyTime(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	out := in.UTC()
	return &out
}
