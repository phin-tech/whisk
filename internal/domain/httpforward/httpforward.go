package httpforward

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"slices"
	"strings"
)

type Record struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	SessionID string `json:"sessionId"`
}

type CreateRequest struct {
	ID        string
	Name      string
	TargetURL string
	SessionID string
}

type State struct {
	records []Record
}

func NewState() *State {
	return &State{}
}

func (s *State) Create(req CreateRequest) (Record, error) {
	if req.ID == "" {
		return Record{}, fmt.Errorf("http forward id required")
	}
	target, err := ValidateTarget(req.TargetURL)
	if err != nil {
		return Record{}, err
	}
	record := Record{
		ID:        req.ID,
		Name:      req.Name,
		TargetURL: target.String(),
		SessionID: req.SessionID,
	}
	s.records = append(s.records, record)
	return record, nil
}

func (s *State) List() []Record {
	return slices.Clone(s.records)
}

func (s *State) Get(id string) (Record, bool) {
	for _, record := range s.records {
		if record.ID == id {
			return record, true
		}
	}
	return Record{}, false
}

func (s *State) Delete(id string) bool {
	for i, record := range s.records {
		if record.ID == id {
			s.records = slices.Delete(s.records, i, i+1)
			return true
		}
	}
	return false
}

func ValidateTarget(raw string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid target url: %w", err)
	}
	if target.Scheme != "http" {
		return nil, fmt.Errorf("http forward target must use http")
	}
	if target.Host == "" {
		return nil, fmt.Errorf("http forward target host required")
	}
	host := target.Hostname()
	if host == "localhost" {
		return target, nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("http forward target must be loopback")
	}
	return target, nil
}

func ProxyPath(targetBasePath string, requestPath string) (string, error) {
	if targetBasePath != "" && !strings.HasPrefix(targetBasePath, "/") {
		return "", fmt.Errorf("target base path must start with slash")
	}
	if requestPath != "" && !strings.HasPrefix(requestPath, "/") {
		return "", fmt.Errorf("request path must start with slash")
	}
	if requestPath == "" {
		requestPath = "/"
	}
	if targetBasePath == "" || targetBasePath == "/" {
		return "/" + strings.TrimLeft(requestPath, "/"), nil
	}
	return path.Clean(strings.TrimRight(targetBasePath, "/") + "/" + strings.TrimLeft(requestPath, "/")), nil
}
