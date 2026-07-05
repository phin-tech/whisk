package client

import (
	"context"
	"net/url"

	"github.com/phin-tech/whisk/internal/protocol"
)

func (c *HTTPClient) ConnectBrowserResource(ctx context.Context, req protocol.ConnectBrowserResourceRequest) (protocol.BrowserResource, error) {
	var resource protocol.BrowserResource
	err := c.post(ctx, "/v1/browser-resources", req, &resource)
	return resource, err
}

func (c *HTTPClient) ListBrowserResources(ctx context.Context) ([]protocol.BrowserResource, error) {
	var resources []protocol.BrowserResource
	err := c.get(ctx, "/v1/browser-resources", nil, &resources)
	return resources, err
}

func (c *HTTPClient) ListBrowserTargets(ctx context.Context, resourceID string) ([]protocol.BrowserTarget, error) {
	var targets []protocol.BrowserTarget
	path := "/v1/browser-resources/" + url.PathEscape(resourceID) + "/targets"
	err := c.get(ctx, path, nil, &targets)
	return targets, err
}

func (c *HTTPClient) DisconnectBrowserResource(ctx context.Context, resourceID string) error {
	return c.delete(ctx, "/v1/browser-resources/"+url.PathEscape(resourceID))
}
