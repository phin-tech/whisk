package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/app"
	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) connectBrowserResource(w http.ResponseWriter, r *http.Request) {
	var req protocol.ConnectBrowserResourceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resource, err := s.runtime.ConnectBrowserResource(r.Context(), app.ConnectBrowserResourceRequest{
		Name:                          req.Name,
		CDPURL:                        req.CDPURL,
		AcknowledgeBrowserControlRisk: req.AcknowledgeBrowserControlRisk,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProtocolBrowserResource(resource))
}

func (s *HTTPServer) listBrowserResources(w http.ResponseWriter, r *http.Request) {
	resources, err := s.runtime.ListBrowserResources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.BrowserResource, 0, len(resources))
	for _, resource := range resources {
		out = append(out, toProtocolBrowserResource(resource))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) listBrowserTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := s.runtime.ListBrowserTargets(r.Context(), r.PathValue("resourceID"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	out := make([]protocol.BrowserTarget, 0, len(targets))
	for _, target := range targets {
		out = append(out, toProtocolBrowserTarget(target))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) disconnectBrowserResource(w http.ResponseWriter, r *http.Request) {
	if err := s.runtime.DisconnectBrowserResource(r.Context(), r.PathValue("resourceID")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toProtocolBrowserResource(resource domainbrowser.Resource) protocol.BrowserResource {
	return protocol.BrowserResource{
		ID:        string(resource.ID),
		Name:      resource.Name,
		CDPURL:    resource.CDPURL,
		Connected: resource.Connected,
	}
}

func toProtocolBrowserTarget(target domainbrowser.Target) protocol.BrowserTarget {
	return protocol.BrowserTarget{
		ID:         string(target.ID),
		ResourceID: string(target.ResourceID),
		Type:       string(target.Type),
		Status:     string(target.Status),
		URL:        target.URL,
		Title:      target.Title,
	}
}
