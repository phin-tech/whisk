package app

import (
	"context"

	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
)

type AgentHookIntegration = agenthooks.Integration

type AgentHookIntegrationRequest struct {
	Provider string
}

func (r *Runtime) ListAgentHookIntegrations(ctx context.Context) ([]AgentHookIntegration, error) {
	installer, err := r.agentHookInstaller()
	if err != nil {
		return nil, err
	}
	return installer.List(ctx)
}

func (r *Runtime) CheckAgentHookIntegration(ctx context.Context, req AgentHookIntegrationRequest) (AgentHookIntegration, error) {
	installer, err := r.agentHookInstaller()
	if err != nil {
		return AgentHookIntegration{}, err
	}
	return installer.Check(ctx, req.Provider)
}

func (r *Runtime) InstallAgentHookIntegration(ctx context.Context, req AgentHookIntegrationRequest) (AgentHookIntegration, error) {
	installer, err := r.agentHookInstaller()
	if err != nil {
		return AgentHookIntegration{}, err
	}
	return installer.Install(ctx, req.Provider)
}

func (r *Runtime) RemoveAgentHookIntegration(ctx context.Context, req AgentHookIntegrationRequest) (AgentHookIntegration, error) {
	installer, err := r.agentHookInstaller()
	if err != nil {
		return AgentHookIntegration{}, err
	}
	return installer.Remove(ctx, req.Provider)
}

func (r *Runtime) agentHookInstaller() (*agenthooks.Installer, error) {
	if r.agentHookPaths != nil {
		paths := *r.agentHookPaths
		if paths.HelperSourcePath == "" {
			paths.HelperSourcePath = r.cliPath
		}
		return agenthooks.NewInstaller(paths), nil
	}
	paths, err := agenthooks.DefaultPaths(r.cliPath)
	if err != nil {
		return nil, err
	}
	return agenthooks.NewInstaller(paths), nil
}
