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
	status, err := s.runtime.InstallPlugin(r.Context(), pathValue(r, "pluginID", ""))
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

func protocolPluginStatuses(statuses []app.PluginStatus) []protocol.PluginStatus {
	out := make([]protocol.PluginStatus, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, protocolPluginStatus(status))
	}
	return out
}

func protocolPluginStatus(status app.PluginStatus) protocol.PluginStatus {
	resolvers := make([]protocol.PluginResolver, 0, len(status.Resolvers))
	for _, resolver := range status.Resolvers {
		resolvers = append(resolvers, protocol.PluginResolver{Provider: resolver.Provider, Kinds: resolver.Kinds})
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
		Name:                       status.Name,
		Version:                    status.Version,
		Dir:                        status.Dir,
		ManifestPath:               status.ManifestPath,
		Trusted:                    status.Trusted,
		Valid:                      status.Valid,
		Error:                      status.Error,
		Resolvers:                  resolvers,
		ProjectAttachmentTemplates: templates,
	}
}
