package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	skilladapter "github.com/phin-tech/whisk/internal/adapters/skills"
	domainskills "github.com/phin-tech/whisk/internal/domain/skills"
)

type ListSkillsRequest struct {
	ProjectID string
	SessionID string
}

func (r *Runtime) ListSkills(_ context.Context, req ListSkillsRequest) (domainskills.DiscoveryResult, error) {
	sources, err := r.skillDiscoverySources(req)
	if err != nil {
		return domainskills.DiscoveryResult{}, err
	}
	return skilladapter.Discover(sources, r.skillNow()), nil
}

func (r *Runtime) RescanSkills(ctx context.Context, req ListSkillsRequest) (domainskills.DiscoveryResult, error) {
	return r.ListSkills(ctx, req)
}

func (r *Runtime) skillDiscoverySources(req ListSkillsRequest) ([]domainskills.Source, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	sessionID := strings.TrimSpace(req.SessionID)
	projectPaths := []string{}

	r.mu.Lock()
	defer r.mu.Unlock()

	if projectID != "" {
		project, ok := r.workItems.GetProject(projectID)
		if !ok {
			return nil, fmt.Errorf("project %s not found", projectID)
		}
		projectPaths = append(projectPaths, project.RootDir)
	} else {
		for _, project := range r.workItems.ListProjects() {
			projectPaths = append(projectPaths, project.RootDir)
		}
	}

	if sessionID != "" {
		session, ok := r.state.Get(sessionID)
		if !ok {
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		projectPaths = append(projectPaths, session.RootDir)
		if session.ProjectID != "" && session.ProjectID != projectID {
			if project, ok := r.workItems.GetProject(session.ProjectID); ok {
				projectPaths = append(projectPaths, project.RootDir)
			}
		}
	} else {
		for _, session := range r.state.List() {
			projectPaths = append(projectPaths, session.RootDir)
		}
	}

	if len(r.skillSources) > 0 {
		sources := append([]domainskills.Source(nil), r.skillSources...)
		sources = append(sources, domainskills.DefaultSources(domainskills.SourceConfig{ProjectPaths: projectPaths})...)
		return sources, nil
	}
	return domainskills.DefaultSources(domainskills.SourceConfig{
		HomeDir:      r.skillHomeDirOrDefault(),
		BundledPath:  r.bundledSkillDirOrDefault(),
		ProjectPaths: projectPaths,
	}), nil
}

func (r *Runtime) skillHomeDirOrDefault() string {
	if strings.TrimSpace(r.skillHomeDir) != "" {
		return r.skillHomeDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func (r *Runtime) bundledSkillDirOrDefault() string {
	if strings.TrimSpace(r.bundledSkillDir) != "" {
		return r.bundledSkillDir
	}
	sourceDir := r.skillSourceDir()
	if filepath.Base(sourceDir) == "whisk" {
		return filepath.Dir(sourceDir)
	}
	return sourceDir
}
