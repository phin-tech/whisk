package app

import (
	"context"
	"sort"
	"strings"
)

type UIContributionScope struct {
	ProjectID    string
	WorkItemID   string
	RunID        string
	SessionID    string
	PaneID       string
	PTYID        string
	GateReportID string
	Phase        string
}

type UIContributionPlugin struct {
	PluginID       string
	Name           string
	Version        string
	Trusted        bool
	Enabled        bool
	DisabledReason string
	Resolvers      []PluginResolver
	Permissions    *PluginPermissions
	Panels         []PluginUIPanel
	Commands       []PluginUICommand
	ReviewActions  []PluginReviewAction
}

type UIContributions struct {
	Scope   UIContributionScope
	Plugins []UIContributionPlugin
}

func NormalizeUIContributionScope(scope UIContributionScope) UIContributionScope {
	return UIContributionScope{
		ProjectID:    strings.TrimSpace(scope.ProjectID),
		WorkItemID:   strings.TrimSpace(scope.WorkItemID),
		RunID:        strings.TrimSpace(scope.RunID),
		SessionID:    strings.TrimSpace(scope.SessionID),
		PaneID:       strings.TrimSpace(scope.PaneID),
		PTYID:        strings.TrimSpace(scope.PTYID),
		GateReportID: strings.TrimSpace(scope.GateReportID),
		Phase:        strings.TrimSpace(scope.Phase),
	}
}

func (s UIContributionScope) hasEntityScope() bool {
	return s.ProjectID != "" || s.WorkItemID != "" || s.RunID != "" ||
		s.SessionID != "" || s.PaneID != "" || s.PTYID != "" ||
		s.GateReportID != ""
}

func contributionsMatchScope(scope PluginUIScope, reqScope UIContributionScope) bool {
	switch string(scope) {
	case "global":
		return true
	case "project":
		return reqScope.hasEntityScope() && reqScope.ProjectID != ""
	case "workItem":
		return reqScope.hasEntityScope() && reqScope.WorkItemID != ""
	case "run":
		return reqScope.hasEntityScope() && reqScope.RunID != ""
	case "gate":
		return reqScope.hasEntityScope() && reqScope.GateReportID != ""
	case "session":
		return reqScope.hasEntityScope() && reqScope.SessionID != ""
	case "pane":
		return reqScope.hasEntityScope() && reqScope.PaneID != ""
	case "pty":
		return reqScope.hasEntityScope() && reqScope.PTYID != ""
	default:
		return false
	}
}

func filterUIPanels(panels []PluginUIPanel, scope UIContributionScope) []PluginUIPanel {
	out := make([]PluginUIPanel, 0, len(panels))
	for _, p := range panels {
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" {
			continue
		}
		if contributionsMatchScope(p.Scope, scope) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func filterUICommands(cmds []PluginUICommand, scope UIContributionScope) []PluginUICommand {
	out := make([]PluginUICommand, 0, len(cmds))
	for _, c := range cmds {
		c.ID = strings.TrimSpace(c.ID)
		if c.ID == "" {
			continue
		}
		if contributionsMatchScope(c.Scope, scope) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func filterReviewActions(actions []PluginReviewAction, scope UIContributionScope) []PluginReviewAction {
	out := make([]PluginReviewAction, 0, len(actions))
	for _, a := range actions {
		a.ID = strings.TrimSpace(a.ID)
		if a.ID == "" {
			continue
		}
		if contributionsMatchScope(a.Scope, scope) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func uiContributionPluginDisabledReason(p PluginStatus) string {
	if !p.Valid {
		if strings.TrimSpace(p.Error) != "" {
			return strings.TrimSpace(p.Error)
		}
		return "plugin manifest is invalid"
	}
	if !p.Trusted {
		return "plugin is not trusted"
	}
	return ""
}

func BuildUIContributions(plugins []PluginStatus, scope UIContributionScope) UIContributions {
	normalizedScope := NormalizeUIContributionScope(scope)
	return UIContributions{
		Scope:   normalizedScope,
		Plugins: AggregateUIContributions(plugins, normalizedScope),
	}
}

func AggregateUIContributions(plugins []PluginStatus, scope UIContributionScope) []UIContributionPlugin {
	scope = NormalizeUIContributionScope(scope)
	var result []UIContributionPlugin
	for _, p := range plugins {
		pluginID := strings.TrimSpace(p.ID)
		if pluginID == "" {
			continue
		}
		disabledReason := uiContributionPluginDisabledReason(p)
		if disabledReason != "" {
			continue
		}
		panels := filterUIPanels(p.UIPanels, scope)
		commands := filterUICommands(p.UICommands, scope)
		reviewActions := filterReviewActions(p.ReviewActions, scope)
		if len(panels) == 0 && len(commands) == 0 && len(reviewActions) == 0 {
			continue
		}
		result = append(result, UIContributionPlugin{
			PluginID:       pluginID,
			Name:           p.Name,
			Version:        p.Version,
			Trusted:        p.Trusted,
			Enabled:        disabledReason == "",
			DisabledReason: disabledReason,
			Resolvers:      p.Resolvers,
			Permissions:    p.Permissions,
			Panels:         panels,
			Commands:       commands,
			ReviewActions:  reviewActions,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PluginID < result[j].PluginID
	})
	return result
}

func (r *Runtime) ListUIContributions(ctx context.Context, scope UIContributionScope) (UIContributions, error) {
	plugins, err := r.ListPlugins(ctx)
	if err != nil {
		return UIContributions{}, err
	}
	return BuildUIContributions(plugins, scope), nil
}
