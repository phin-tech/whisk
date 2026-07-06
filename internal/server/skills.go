package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/app"
	domainskills "github.com/phin-tech/whisk/internal/domain/skills"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) listSkills(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.runtime.ListSkills(r.Context(), app.ListSkillsRequest{
		ProjectID: r.URL.Query().Get("projectId"),
		SessionID: r.URL.Query().Get("sessionId"),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolSkillCatalog(catalog))
}

func (s *HTTPServer) rescanSkills(w http.ResponseWriter, r *http.Request) {
	var req protocol.ListSkillsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	catalog, err := s.runtime.RescanSkills(r.Context(), app.ListSkillsRequest{
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, protocolSkillCatalog(catalog))
}

func protocolSkillCatalog(catalog domainskills.DiscoveryResult) protocol.SkillCatalog {
	skills := make([]protocol.Skill, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		skills = append(skills, protocol.Skill{
			ID:            skill.ID,
			Name:          skill.Name,
			Description:   skill.Description,
			Providers:     protocolSkillProviders(skill.Providers),
			SourceKind:    string(skill.SourceKind),
			SourceLabel:   skill.SourceLabel,
			RootPath:      skill.RootPath,
			DirectoryPath: skill.DirectoryPath,
			SkillFilePath: skill.SkillFilePath,
			FileCount:     skill.FileCount,
			UpdatedAt:     skill.UpdatedAt,
		})
	}
	sources := make([]protocol.SkillSource, 0, len(catalog.Sources))
	for _, source := range catalog.Sources {
		sources = append(sources, protocol.SkillSource{
			ID:            source.ID,
			Label:         source.Label,
			Path:          source.Path,
			Kind:          string(source.Kind),
			Providers:     protocolSkillProviders(source.Providers),
			Exists:        source.Exists,
			SkippedReason: source.SkippedReason,
		})
	}
	return protocol.SkillCatalog{
		Skills:    skills,
		Sources:   sources,
		ScannedAt: catalog.ScannedAt,
	}
}

func protocolSkillProviders(providers []domainskills.Provider) []string {
	out := make([]string, 0, len(providers))
	for _, provider := range providers {
		out = append(out, string(provider))
	}
	return out
}
