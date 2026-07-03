package client

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/phin-tech/whisk/internal/protocol"
)

func (c *HTTPClient) SendMail(ctx context.Context, req protocol.SendMailRequest) (protocol.MailMessage, error) {
	var message protocol.MailMessage
	err := c.post(ctx, "/v1/mail", req, &message)
	return message, err
}

func (c *HTTPClient) ListMail(ctx context.Context, req protocol.ListMailRequest) ([]protocol.MailMessage, error) {
	var messages []protocol.MailMessage
	err := c.get(ctx, "/v1/mail", mailListQuery(req), &messages)
	return messages, err
}

func (c *HTTPClient) NextMail(ctx context.Context, req protocol.NextMailRequest) (protocol.NextMailResponse, error) {
	var response protocol.NextMailResponse
	err := c.get(ctx, "/v1/mail/next", mailNextQuery(req), &response)
	return response, err
}

func (c *HTTPClient) MarkMailRead(ctx context.Context, mailID string, req protocol.MarkMailReadRequest) (protocol.MailMessage, error) {
	var message protocol.MailMessage
	path := "/v1/mail/" + url.PathEscape(mailID) + "/read"
	err := c.post(ctx, path, req, &message)
	return message, err
}

func (c *HTTPClient) ReplyMail(ctx context.Context, mailID string, req protocol.ReplyMailRequest) (protocol.MailMessage, error) {
	var message protocol.MailMessage
	path := "/v1/mail/" + url.PathEscape(mailID) + "/reply"
	err := c.post(ctx, path, req, &message)
	return message, err
}

func mailListQuery(req protocol.ListMailRequest) url.Values {
	query := url.Values{}
	addMailAddresses(query, "to", req.To)
	if req.UnreadOnly {
		query.Set("unread", "true")
	}
	addCSV(query, "types", req.Types)
	if req.ProjectID != "" {
		query.Set("projectId", req.ProjectID)
	}
	if req.WorkItemID != "" {
		query.Set("workItemId", req.WorkItemID)
	}
	if req.RunID != "" {
		query.Set("runId", req.RunID)
	}
	if req.ThreadID != "" {
		query.Set("threadId", req.ThreadID)
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	return query
}

func mailNextQuery(req protocol.NextMailRequest) url.Values {
	query := url.Values{}
	addMailAddresses(query, "to", req.To)
	addCSV(query, "types", req.Types)
	if req.TimeoutMs > 0 {
		query.Set("timeoutMs", strconv.Itoa(req.TimeoutMs))
	}
	if req.ProjectID != "" {
		query.Set("projectId", req.ProjectID)
	}
	return query
}

func addMailAddresses(query url.Values, key string, addresses []protocol.MailAddress) {
	if len(addresses) == 0 {
		return
	}
	values := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.Kind == "" || address.ID == "" {
			continue
		}
		values = append(values, address.Kind+":"+address.ID)
	}
	if len(values) > 0 {
		query.Set(key, strings.Join(values, ","))
	}
}

func addCSV(query url.Values, key string, values []string) {
	if len(values) == 0 {
		return
	}
	compact := values[:0]
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			compact = append(compact, value)
		}
	}
	if len(compact) > 0 {
		query.Set(key, strings.Join(compact, ","))
	}
}
