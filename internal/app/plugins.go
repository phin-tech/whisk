package app

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type PluginRegistry interface {
	ListPlugins(context.Context) ([]PluginStatus, error)
	RescanPlugins(context.Context) ([]PluginStatus, error)
	TrustPlugin(context.Context, string) (PluginStatus, error)
	UntrustPlugin(context.Context, string) (PluginStatus, error)
	RunProjectAttachmentTemplate(context.Context, RunPluginProjectAttachmentTemplateRequest) (AddProjectAttachmentRequest, error)
	ResolveProjectAttachmentProvider(string) ProjectContextResolver
}

type PluginStatus struct {
	ID                         string                      `json:"id"`
	Name                       string                      `json:"name"`
	Version                    string                      `json:"version"`
	Dir                        string                      `json:"dir"`
	ManifestPath               string                      `json:"manifestPath"`
	Trusted                    bool                        `json:"trusted"`
	Valid                      bool                        `json:"valid"`
	Error                      string                      `json:"error,omitempty"`
	Resolvers                  []PluginResolver            `json:"resolvers,omitempty"`
	ProjectAttachmentTemplates []ProjectAttachmentTemplate `json:"projectAttachmentTemplates,omitempty"`
}

type PluginResolver struct {
	Provider string   `json:"provider"`
	Kinds    []string `json:"kinds,omitempty"`
}

type ProjectAttachmentTemplate struct {
	ID       string                `json:"id"`
	Label    string                `json:"label"`
	Provider string                `json:"provider"`
	Kind     string                `json:"kind"`
	Fields   []PluginTemplateField `json:"fields,omitempty"`
}

type PluginTemplateField struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Placeholder string   `json:"placeholder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type RunPluginProjectAttachmentTemplateRequest struct {
	PluginID   string            `json:"pluginId"`
	TemplateID string            `json:"templateId"`
	ProjectID  string            `json:"projectId"`
	Values     map[string]string `json:"values,omitempty"`
}

type PluginProjectAttachmentOutput struct {
	Kind             string                            `json:"kind"`
	Scope            string                            `json:"scope,omitempty"`
	Title            string                            `json:"title,omitempty"`
	Path             string                            `json:"path,omitempty"`
	URL              string                            `json:"url,omitempty"`
	Note             string                            `json:"note,omitempty"`
	Provider         string                            `json:"provider,omitempty"`
	Target           string                            `json:"target,omitempty"`
	IncludeInContext bool                              `json:"includeInContext,omitempty"`
	Meta             map[string]workitem.MetadataValue `json:"meta,omitempty"`
}

func (r *Runtime) ListPlugins(ctx context.Context) ([]PluginStatus, error) {
	if r.plugins == nil {
		return nil, nil
	}
	return r.plugins.ListPlugins(ctx)
}

func (r *Runtime) RescanPlugins(ctx context.Context) ([]PluginStatus, error) {
	if r.plugins == nil {
		return nil, nil
	}
	return r.plugins.RescanPlugins(ctx)
}

func (r *Runtime) TrustPlugin(ctx context.Context, id string) (PluginStatus, error) {
	if r.plugins == nil {
		return PluginStatus{}, nil
	}
	return r.plugins.TrustPlugin(ctx, id)
}

func (r *Runtime) UntrustPlugin(ctx context.Context, id string) (PluginStatus, error) {
	if r.plugins == nil {
		return PluginStatus{}, nil
	}
	return r.plugins.UntrustPlugin(ctx, id)
}

func (r *Runtime) RunPluginProjectAttachmentTemplate(ctx context.Context, req RunPluginProjectAttachmentTemplateRequest) (workitem.Project, error) {
	if r.plugins == nil {
		return workitem.Project{}, fmt.Errorf("plugins are not configured")
	}
	attachment, err := r.plugins.RunProjectAttachmentTemplate(ctx, req)
	if err != nil {
		return workitem.Project{}, err
	}
	attachment.ProjectID = req.ProjectID
	return r.AddProjectAttachment(ctx, attachment)
}

func (r *Runtime) projectContextResolver(provider string) ProjectContextResolver {
	if r.plugins != nil {
		if resolver := r.plugins.ResolveProjectAttachmentProvider(provider); resolver != nil {
			return resolver
		}
	}
	return r.contextResolvers[provider]
}
