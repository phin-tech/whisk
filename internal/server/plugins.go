package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) listPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.runtime.ListPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolPluginStatuses(plugins))
}

func (s *HTTPServer) rescanPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.runtime.RescanPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolPluginStatuses(plugins))
}

func (s *HTTPServer) listRegistryPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.runtime.ListRegistryPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolRegistryPlugins(plugins))
}

func (s *HTTPServer) installRegistryPlugin(w http.ResponseWriter, r *http.Request) {
	var req protocol.InstallRegistryPluginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := s.runtime.InstallPlugin(r.Context(), req.Registry, req.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, protocolPluginStatus(status))
}

func protocolRegistryPlugins(plugins []app.RegistryPlugin) []protocol.RegistryPlugin {
	out := make([]protocol.RegistryPlugin, 0, len(plugins))
	for _, plugin := range plugins {
		out = append(out, protocol.RegistryPlugin{
			Registry:    plugin.Registry,
			ID:          plugin.ID,
			Name:        plugin.Name,
			Description: plugin.Description,
			SourceType:  plugin.SourceType,
			Installed:   plugin.Installed,
			Trusted:     plugin.Trusted,
		})
	}
	return out
}

func (s *HTTPServer) trustPlugin(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.TrustPlugin(r.Context(), pathValue(r, "pluginID", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolPluginStatus(status))
}

func (s *HTTPServer) untrustPlugin(w http.ResponseWriter, r *http.Request) {
	status, err := s.runtime.UntrustPlugin(r.Context(), pathValue(r, "pluginID", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolPluginStatus(status))
}

func (s *HTTPServer) runPluginProjectAttachmentTemplate(w http.ResponseWriter, r *http.Request) {
	var req protocol.RunPluginProjectAttachmentTemplateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.runtime.RunPluginProjectAttachmentTemplate(r.Context(), app.RunPluginProjectAttachmentTemplateRequest{
		PluginID:   pathValue(r, "pluginID", ""),
		TemplateID: pathValue(r, "templateID", ""),
		ProjectID:  req.ProjectID,
		Values:     req.Values,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *HTTPServer) listUsageResolvers(w http.ResponseWriter, r *http.Request) {
	results, err := s.runtime.ListUsageResolverResults(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolUsageResolverReadModels(results))
}

func (s *HTTPServer) refreshUsageResolver(w http.ResponseWriter, r *http.Request) {
	var req protocol.RefreshUsageResolverRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := s.runtime.RefreshUsageResolver(r.Context(), app.RunUsageResolverRequest{
		PluginID:   pathValue(r, "pluginID", ""),
		ResolverID: pathValue(r, "resolverID", ""),
		Profile:    req.Profile,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolUsageResolverReadModel(result))
}

func protocolPluginStatuses(statuses []app.PluginStatus) []protocol.PluginStatus {
	out := make([]protocol.PluginStatus, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, protocolPluginStatus(status))
	}
	return out
}

func protocolUsageResolverReadModels(models []app.UsageResolverReadModel) []protocol.UsageResolverReadModel {
	out := make([]protocol.UsageResolverReadModel, 0, len(models))
	for _, model := range models {
		out = append(out, protocolUsageResolverReadModel(model))
	}
	return out
}

func protocolUsageResolverReadModel(model app.UsageResolverReadModel) protocol.UsageResolverReadModel {
	return protocol.UsageResolverReadModel{
		PluginID:     model.PluginID,
		ResolverID:   model.ResolverID,
		Provider:     model.Provider,
		Label:        model.Label,
		Profile:      model.Profile,
		Trusted:      model.Trusted,
		Valid:        model.Valid,
		Status:       model.Status,
		Error:        model.Error,
		RefreshedAt:  model.RefreshedAt,
		Stale:        model.Stale,
		MinRefreshMs: model.MinRefreshMs,
		StaleAfterMs: model.StaleAfterMs,
		Result:       protocolUsageResolverResult(model.Result),
	}
}

func protocolUsageResolverResult(result *app.UsageResolverResult) *protocol.UsageResolverResult {
	if result == nil {
		return nil
	}
	return &protocol.UsageResolverResult{
		Summary:   result.Summary,
		Metrics:   protocolUsageResolverMetrics(result.Metrics),
		FetchedAt: result.FetchedAt,
		Meta:      result.Meta,
	}
}

func protocolUsageResolverMetrics(metrics []app.UsageResolverMetric) []protocol.UsageResolverMetric {
	out := make([]protocol.UsageResolverMetric, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, protocol.UsageResolverMetric{
			ID:        metric.ID,
			Kind:      metric.Kind,
			Label:     metric.Label,
			Unit:      metric.Unit,
			Used:      metric.Used,
			Limit:     metric.Limit,
			Remaining: metric.Remaining,
			ResetAt:   metric.ResetAt,
		})
	}
	return out
}

func protocolPluginStatus(status app.PluginStatus) protocol.PluginStatus {
	resolvers := make([]protocol.PluginResolver, 0, len(status.Resolvers))
	for _, resolver := range status.Resolvers {
		resolvers = append(resolvers, protocol.PluginResolver{Provider: resolver.Provider, Kinds: resolver.Kinds})
	}
	usageResolvers := make([]protocol.PluginUsageResolver, 0, len(status.UsageResolvers))
	for _, resolver := range status.UsageResolvers {
		usageResolvers = append(usageResolvers, protocol.PluginUsageResolver{
			ID:             resolver.ID,
			Provider:       resolver.Provider,
			Label:          resolver.Label,
			Profiles:       resolver.Profiles,
			TimeoutMs:      resolver.TimeoutMs,
			OutputCapBytes: resolver.OutputCapBytes,
			MinRefreshMs:   resolver.MinRefreshMs,
			StaleAfterMs:   resolver.StaleAfterMs,
		})
	}
	templates := make([]protocol.ProjectAttachmentTemplate, 0, len(status.ProjectAttachmentTemplates))
	for _, template := range status.ProjectAttachmentTemplates {
		fields := make([]protocol.PluginTemplateField, 0, len(template.Fields))
		for _, field := range template.Fields {
			fields = append(fields, protocol.PluginTemplateField{
				ID:          field.ID,
				Label:       field.Label,
				Type:        field.Type,
				Placeholder: field.Placeholder,
				Required:    field.Required,
				Options:     field.Options,
			})
		}
		templates = append(templates, protocol.ProjectAttachmentTemplate{
			ID:       template.ID,
			Label:    template.Label,
			Provider: template.Provider,
			Kind:     template.Kind,
			Fields:   fields,
		})
	}
	return protocol.PluginStatus{
		ID:                         status.ID,
		Registry:                   status.Registry,
		Name:                       status.Name,
		Version:                    status.Version,
		Dir:                        status.Dir,
		ManifestPath:               status.ManifestPath,
		Trusted:                    status.Trusted,
		Valid:                      status.Valid,
		Error:                      status.Error,
		Resolvers:                  resolvers,
		UsageResolvers:             usageResolvers,
		ProjectAttachmentTemplates: templates,
		UIPanels:                   protocolPluginUIPanels(status.UIPanels),
		UICommands:                 protocolPluginUICommands(status.UICommands),
		ReviewActions:              protocolPluginReviewActions(status.ReviewActions),
		Permissions:                protocolPluginPermissions(status.Permissions),
	}
}

func protocolPluginUIPanels(panels []app.PluginUIPanel) []protocol.PluginUIPanel {
	out := make([]protocol.PluginUIPanel, 0, len(panels))
	for _, panel := range panels {
		out = append(out, protocol.PluginUIPanel{
			ID:      panel.ID,
			Title:   panel.Title,
			Scope:   protocol.PluginUIScope(panel.Scope),
			Kind:    panel.Kind,
			Read:    protocolPluginUICommandRef(panel.Read),
			Entry:   protocolPluginUIPanelEntry(panel.Entry),
			Actions: protocolPluginUICommandRefs(panel.Actions),
		})
	}
	return out
}

func protocolPluginUICommands(commands []app.PluginUICommand) []protocol.PluginUICommand {
	out := make([]protocol.PluginUICommand, 0, len(commands))
	for _, command := range commands {
		out = append(out, protocol.PluginUICommand{
			ID:             command.ID,
			Label:          command.Label,
			Scope:          protocol.PluginUIScope(command.Scope),
			TimeoutMs:      command.TimeoutMs,
			OutputCapBytes: command.OutputCapBytes,
		})
	}
	return out
}

func protocolPluginReviewActions(actions []app.PluginReviewAction) []protocol.PluginReviewAction {
	out := make([]protocol.PluginReviewAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, protocol.PluginReviewAction{
			ID:             action.ID,
			Label:          action.Label,
			Scope:          protocol.PluginUIScope(action.Scope),
			URLTemplate:    action.URLTemplate,
			HasSubmit:      action.HasSubmit,
			Blocking:       action.Blocking,
			TimeoutMs:      action.TimeoutMs,
			OutputCapBytes: action.OutputCapBytes,
		})
	}
	return out
}

func protocolPluginUICommandRefs(refs []app.PluginUICommandRef) []protocol.PluginUICommandRef {
	out := make([]protocol.PluginUICommandRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, protocol.PluginUICommandRef{
			ID:             ref.ID,
			Label:          ref.Label,
			TimeoutMs:      ref.TimeoutMs,
			OutputCapBytes: ref.OutputCapBytes,
		})
	}
	return out
}

func protocolPluginUICommandRef(ref *app.PluginUICommandRef) *protocol.PluginUICommandRef {
	if ref == nil {
		return nil
	}
	return &protocol.PluginUICommandRef{
		ID:             ref.ID,
		Label:          ref.Label,
		TimeoutMs:      ref.TimeoutMs,
		OutputCapBytes: ref.OutputCapBytes,
	}
}

func protocolPluginUIPanelEntry(entry *app.PluginUIPanelEntry) *protocol.PluginUIPanelEntry {
	if entry == nil {
		return nil
	}
	return &protocol.PluginUIPanelEntry{
		Path:    entry.Path,
		Forward: entry.Forward,
	}
}

func (s *HTTPServer) listUIContributions(w http.ResponseWriter, r *http.Request) {
	scope := app.UIContributionScope{
		ProjectID:    r.URL.Query().Get("projectId"),
		WorkItemID:   r.URL.Query().Get("workItemId"),
		RunID:        r.URL.Query().Get("runId"),
		SessionID:    r.URL.Query().Get("sessionId"),
		PaneID:       r.URL.Query().Get("paneId"),
		PTYID:        r.URL.Query().Get("ptyId"),
		GateReportID: r.URL.Query().Get("gateReportId"),
		Phase:        r.URL.Query().Get("phase"),
	}
	contributions, err := s.runtime.ListUIContributions(r.Context(), scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := protocol.UIContributionsResponse{
		Scope: protocol.UIContributionScope{
			ProjectID:    contributions.Scope.ProjectID,
			WorkItemID:   contributions.Scope.WorkItemID,
			RunID:        contributions.Scope.RunID,
			SessionID:    contributions.Scope.SessionID,
			PaneID:       contributions.Scope.PaneID,
			PTYID:        contributions.Scope.PTYID,
			GateReportID: contributions.Scope.GateReportID,
			Phase:        contributions.Scope.Phase,
		},
		Plugins: protocolUIContributionPlugins(contributions.Plugins),
	}
	writeJSON(w, http.StatusOK, resp)
}

func protocolUIContributionPlugins(plugins []app.UIContributionPlugin) []protocol.UIContributionPlugin {
	out := make([]protocol.UIContributionPlugin, 0, len(plugins))
	for _, p := range plugins {
		plugin := protocol.UIContributionPlugin{
			PluginID:       p.PluginID,
			Name:           p.Name,
			Version:        p.Version,
			Trusted:        p.Trusted,
			Enabled:        p.Enabled,
			DisabledReason: p.DisabledReason,
			Resolvers:      protocolPluginResolvers(p.Resolvers),
			Permissions:    protocolPluginPermissions(p.Permissions),
			Panels:         protocolPluginUIPanels(p.Panels),
			Commands:       protocolPluginUICommands(p.Commands),
			ReviewActions:  protocolPluginReviewActions(p.ReviewActions),
		}
		out = append(out, plugin)
	}
	return out
}

func protocolPluginResolvers(resolvers []app.PluginResolver) []protocol.PluginResolver {
	out := make([]protocol.PluginResolver, 0, len(resolvers))
	for _, r := range resolvers {
		out = append(out, protocol.PluginResolver{Provider: r.Provider, Kinds: r.Kinds})
	}
	return out
}

func protocolPluginPermissions(permissions *app.PluginPermissions) *protocol.PluginPermissions {
	if permissions == nil {
		return nil
	}
	return &protocol.PluginPermissions{
		PTYOutput:   permissions.PTYOutput,
		EnvPrefixes: append([]string(nil), permissions.EnvPrefixes...),
		Network:     append([]string(nil), permissions.Network...),
	}
}
