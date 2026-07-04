package app

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type PluginRegistry interface {
	ListPlugins(context.Context) ([]PluginStatus, error)
	RescanPlugins(context.Context) ([]PluginStatus, error)
	TrustPlugin(context.Context, string) (PluginStatus, error)
	UntrustPlugin(context.Context, string) (PluginStatus, error)
	ListRegistryPlugins(context.Context) ([]RegistryPlugin, error)
	InstallPlugin(ctx context.Context, registry, id string) (PluginStatus, error)
	ListAgentProfiles(context.Context) ([]agents.ProfileInfo, error)
	RunProjectAttachmentTemplate(context.Context, RunPluginProjectAttachmentTemplateRequest) (AddProjectAttachmentRequest, error)
	ResolveProjectAttachmentProvider(string) ProjectContextResolver
}

// RegistryPlugin is one installable plugin advertised by a configured plugin
// registry, annotated with whether it is already installed and trusted locally.
type RegistryPlugin struct {
	Registry    string `json:"registry"`
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	SourceType  string `json:"sourceType"`
	Installed   bool   `json:"installed"`
	Trusted     bool   `json:"trusted"`
}

type PluginStatus struct {
	ID                         string                      `json:"id"`
	Registry                   string                      `json:"registry,omitempty"`
	Name                       string                      `json:"name"`
	Version                    string                      `json:"version"`
	Dir                        string                      `json:"dir"`
	ManifestPath               string                      `json:"manifestPath"`
	Trusted                    bool                        `json:"trusted"`
	Valid                      bool                        `json:"valid"`
	Error                      string                      `json:"error,omitempty"`
	Resolvers                  []PluginResolver            `json:"resolvers,omitempty"`
	UsageResolvers             []PluginUsageResolver       `json:"usageResolvers,omitempty"`
	ProjectAttachmentTemplates []ProjectAttachmentTemplate `json:"projectAttachmentTemplates,omitempty"`
	UIPanels                   []PluginUIPanel             `json:"uiPanels,omitempty"`
	UICommands                 []PluginUICommand           `json:"uiCommands,omitempty"`
	ReviewActions              []PluginReviewAction        `json:"reviewActions,omitempty"`
	Permissions                *PluginPermissions          `json:"permissions,omitempty"`
}

type PluginResolver struct {
	Provider string   `json:"provider"`
	Kinds    []string `json:"kinds,omitempty"`
}

type PluginUsageResolver struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	Label          string   `json:"label"`
	Profiles       []string `json:"profiles,omitempty"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	OutputCapBytes int      `json:"outputCapBytes,omitempty"`
	MinRefreshMs   int      `json:"minRefreshMs,omitempty"`
	StaleAfterMs   int      `json:"staleAfterMs,omitempty"`
}

type ProjectAttachmentTemplate struct {
	ID       string                `json:"id"`
	Label    string                `json:"label"`
	Provider string                `json:"provider"`
	Kind     string                `json:"kind"`
	Fields   []PluginTemplateField `json:"fields,omitempty"`
}

type PluginUIScope string

type PluginUIPanel struct {
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Scope   PluginUIScope        `json:"scope"`
	Kind    string               `json:"kind"`
	Read    *PluginUICommandRef  `json:"read,omitempty"`
	Entry   *PluginUIPanelEntry  `json:"entry,omitempty"`
	Actions []PluginUICommandRef `json:"actions,omitempty"`
}

type PluginUIPanelEntry struct {
	Path    string `json:"path,omitempty"`
	Forward string `json:"forward,omitempty"`
}

type PluginUICommand struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	Scope          PluginUIScope `json:"scope"`
	TimeoutMs      int           `json:"timeoutMs,omitempty"`
	OutputCapBytes int           `json:"outputCapBytes,omitempty"`
}

type PluginUICommandRef struct {
	ID             string `json:"id,omitempty"`
	Label          string `json:"label,omitempty"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type PluginReviewAction struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	Scope          PluginUIScope `json:"scope,omitempty"`
	URLTemplate    string        `json:"urlTemplate,omitempty"`
	HasSubmit      bool          `json:"hasSubmit,omitempty"`
	Blocking       bool          `json:"blocking,omitempty"`
	TimeoutMs      int           `json:"timeoutMs,omitempty"`
	OutputCapBytes int           `json:"outputCapBytes,omitempty"`
}

type PluginPermissions struct {
	PTYOutput   bool     `json:"ptyOutput,omitempty"`
	EnvPrefixes []string `json:"envPrefixes,omitempty"`
	Network     []string `json:"network,omitempty"`
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
	plugins, err := r.plugins.RescanPlugins(ctx)
	if err != nil {
		return nil, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPluginsChanged})
	return plugins, nil
}

func (r *Runtime) TrustPlugin(ctx context.Context, id string) (PluginStatus, error) {
	if r.plugins == nil {
		return PluginStatus{}, nil
	}
	status, err := r.plugins.TrustPlugin(ctx, id)
	if err != nil {
		return PluginStatus{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPluginsChanged})
	return status, nil
}

func (r *Runtime) UntrustPlugin(ctx context.Context, id string) (PluginStatus, error) {
	if r.plugins == nil {
		return PluginStatus{}, nil
	}
	status, err := r.plugins.UntrustPlugin(ctx, id)
	if err != nil {
		return PluginStatus{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPluginsChanged})
	return status, nil
}

func (r *Runtime) ListRegistryPlugins(ctx context.Context) ([]RegistryPlugin, error) {
	if r.plugins == nil {
		return nil, nil
	}
	return r.plugins.ListRegistryPlugins(ctx)
}

func (r *Runtime) InstallPlugin(ctx context.Context, registry, id string) (PluginStatus, error) {
	if r.plugins == nil {
		return PluginStatus{}, fmt.Errorf("plugins are not configured")
	}
	status, err := r.plugins.InstallPlugin(ctx, registry, id)
	if err != nil {
		return PluginStatus{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPluginsChanged})
	return status, nil
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
