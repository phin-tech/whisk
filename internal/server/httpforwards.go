package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/httpforward"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) createHTTPForward(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateHTTPForwardRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	created, err := s.runtime.CreateHTTPForward(r.Context(), app.CreateHTTPForwardRequest{
		Name:      req.Name,
		TargetURL: req.TargetURL,
		SessionID: req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProtocolHTTPForward(created))
}

func (s *HTTPServer) listHTTPForwards(w http.ResponseWriter, r *http.Request) {
	forwards, err := s.runtime.ListHTTPForwards(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.HTTPForward, 0, len(forwards))
	for _, forward := range forwards {
		out = append(out, toProtocolHTTPForward(forward))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *HTTPServer) deleteHTTPForward(w http.ResponseWriter, r *http.Request) {
	if err := s.runtime.DeleteHTTPForward(r.Context(), r.PathValue("forwardID")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) proxyHTTPForward(w http.ResponseWriter, r *http.Request) {
	forwardID := r.PathValue("forwardID")
	forward, err := s.runtime.GetHTTPForward(r.Context(), forwardID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	target, err := url.Parse(forward.TargetURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	proxyPrefix := "/v1/http-forwards/" + forwardID + "/proxy"
	proxy := reverseProxy(target, proxyPrefix)
	proxy.ServeHTTP(w, r)
}

func reverseProxy(target *url.URL, prefix string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(proxyReq *httputil.ProxyRequest) {
			in := proxyReq.In
			proxyReq.SetURL(target)
			proxyReq.SetXForwarded()
			proxyReq.Out.Host = target.Host
			proxyReq.Out.URL.Path = targetPath(target.Path, in.URL.Path, prefix)
			proxyReq.Out.URL.RawPath = ""
			proxyReq.Out.URL.RawQuery = mergeRawQuery(target.RawQuery, in.URL.RawQuery)
		},
		FlushInterval: -1 * time.Nanosecond,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadGateway, fmt.Errorf("http forward proxy failed: %w", err))
		},
	}
}

func targetPath(targetBasePath string, incomingPath string, prefix string) string {
	requestPath := strings.TrimPrefix(incomingPath, prefix)
	if requestPath == incomingPath {
		requestPath = incomingPath
	}
	path, err := httpforward.ProxyPath(targetBasePath, requestPath)
	if err != nil {
		return "/"
	}
	return path
}

func mergeRawQuery(base string, incoming string) string {
	if base == "" {
		return incoming
	}
	if incoming == "" {
		return base
	}
	return base + "&" + incoming
}

func toProtocolHTTPForward(forward app.HTTPForward) protocol.HTTPForward {
	return protocol.HTTPForward{
		ID:        forward.ID,
		Name:      forward.Name,
		TargetURL: forward.TargetURL,
		SessionID: forward.SessionID,
	}
}
