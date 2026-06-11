package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTP(baseURL string, httpClient *http.Client) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  httpClient,
	}
}

func (c *HTTPClient) Health(ctx context.Context) error {
	var response struct {
		OK bool `json:"ok"`
	}
	if err := c.get(ctx, "/v1/health", nil, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("daemon health check failed")
	}
	return nil
}

func (c *HTTPClient) ListSessions(ctx context.Context) ([]session.Session, error) {
	var sessions []session.Session
	err := c.get(ctx, "/v1/sessions", nil, &sessions)
	return sessions, err
}

func (c *HTTPClient) CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	var created protocol.CreatedSession
	err := c.post(ctx, "/v1/sessions", req, &created)
	return created, err
}

func (c *HTTPClient) SplitPane(ctx context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	var result protocol.SplitPaneResult
	path := "/v1/sessions/" + url.PathEscape(req.SessionID) + "/split"
	err := c.post(ctx, path, req, &result)
	return result, err
}

func (c *HTTPClient) WritePTY(ctx context.Context, req protocol.WritePTYRequest) error {
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/write"
	return c.post(ctx, path, req, nil)
}

func (c *HTTPClient) ResizePTY(ctx context.Context, req protocol.ResizePTYRequest) error {
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/resize"
	return c.post(ctx, path, req, nil)
}

func (c *HTTPClient) Output(ctx context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	query := url.Values{"from": {strconv.FormatUint(req.FromOffset, 10)}}
	path := "/v1/ptys/" + url.PathEscape(req.PtyID) + "/output"
	var snapshot protocol.OutputSnapshot
	err := c.get(ctx, path, query, &snapshot)
	return snapshot, err
}

func (c *HTTPClient) ListPTYs(ctx context.Context) ([]protocol.PTYInfo, error) {
	var ptys []protocol.PTYInfo
	err := c.get(ctx, "/v1/ptys", nil, &ptys)
	return ptys, err
}

func (c *HTTPClient) NextEvent(ctx context.Context, req protocol.NextEventRequest) (protocol.RuntimeEvent, error) {
	query := url.Values{}
	if req.TimeoutMs > 0 {
		query.Set("timeoutMs", strconv.Itoa(req.TimeoutMs))
	}
	var event protocol.RuntimeEvent
	err := c.get(ctx, "/v1/events/next", query, &event)
	return event, err
}

func (c *HTTPClient) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *HTTPClient) post(ctx context.Context, path string, in any, out any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *HTTPClient) do(req *http.Request, out any) error {
	httpClient := c.client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		var errorResponse protocol.ErrorResponse
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return errors.New(errorResponse.Error)
		}
		return fmt.Errorf("daemon request failed: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}
	return json.NewDecoder(response.Body).Decode(out)
}
