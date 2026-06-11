package client

import (
	"context"
	"net/url"

	"github.com/phin-tech/whisk/internal/protocol"
)

func (c *HTTPClient) CreateHTTPForward(ctx context.Context, req protocol.CreateHTTPForwardRequest) (protocol.HTTPForward, error) {
	var forward protocol.HTTPForward
	err := c.post(ctx, "/v1/http-forwards", req, &forward)
	return forward, err
}

func (c *HTTPClient) ListHTTPForwards(ctx context.Context) ([]protocol.HTTPForward, error) {
	var forwards []protocol.HTTPForward
	err := c.get(ctx, "/v1/http-forwards", nil, &forwards)
	return forwards, err
}

func (c *HTTPClient) DeleteHTTPForward(ctx context.Context, id string) error {
	return c.delete(ctx, "/v1/http-forwards/"+url.PathEscape(id))
}

func (c *HTTPClient) forwardProxyURL(id string) string {
	return c.baseURL + "/v1/http-forwards/" + url.PathEscape(id) + "/proxy"
}
