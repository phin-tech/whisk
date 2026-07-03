package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) sendMail(w http.ResponseWriter, r *http.Request) {
	var req protocol.SendMailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	message, err := s.runtime.SendMail(r.Context(), app.SendMailRequest{
		From:       toDomainMailAddress(req.From),
		Recipients: toDomainMailAddresses(req.To),
		Type:       req.Type,
		Priority:   req.Priority,
		Subject:    req.Subject,
		Body:       req.Body,
		Payload:    req.Payload,
		ThreadID:   req.ThreadID,
		ReplyToID:  req.ReplyToID,
		ProjectID:  req.ProjectID,
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		DispatchID: req.DispatchID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProtocolMailMessage(message))
}

func (s *HTTPServer) listMail(w http.ResponseWriter, r *http.Request) {
	req, err := parseListMailRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	messages, err := s.runtime.ListMail(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolMailMessages(messages))
}

func (s *HTTPServer) nextMail(w http.ResponseWriter, r *http.Request) {
	req, err := parseNextMailRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.runtime.NextMail(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response := protocol.NextMailResponse{Timeout: result.Timeout}
	if result.Message != nil {
		message := toProtocolMailMessage(*result.Message)
		response.Message = &message
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *HTTPServer) markMailRead(w http.ResponseWriter, r *http.Request) {
	var req protocol.MarkMailReadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	message, err := s.runtime.MarkMailRead(r.Context(), app.MarkMailReadRequest{
		ID:        pathValue(r, "mailID", ""),
		Recipient: toDomainMailAddressPtr(req.To),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toProtocolMailMessage(message))
}

func (s *HTTPServer) replyMail(w http.ResponseWriter, r *http.Request) {
	var req protocol.ReplyMailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	message, err := s.runtime.ReplyMail(r.Context(), app.ReplyMailRequest{
		ID:       pathValue(r, "mailID", ""),
		From:     toDomainMailAddress(req.From),
		Type:     req.Type,
		Priority: req.Priority,
		Subject:  req.Subject,
		Body:     req.Body,
		Payload:  req.Payload,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProtocolMailMessage(message))
}

func parseListMailRequest(r *http.Request) (app.ListMailRequest, error) {
	query := r.URL.Query()
	to, err := parseMailQueryAddresses(query["to"])
	if err != nil {
		return app.ListMailRequest{}, err
	}
	types, err := parseMailQueryCSV(query["types"])
	if err != nil {
		return app.ListMailRequest{}, err
	}
	unreadOnly := false
	if raw := query.Get("unread"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return app.ListMailRequest{}, fmt.Errorf("invalid unread")
		}
		unreadOnly = parsed
	}
	limit := 0
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return app.ListMailRequest{}, fmt.Errorf("invalid limit")
		}
		limit = parsed
	}
	return app.ListMailRequest{
		To:         to,
		UnreadOnly: unreadOnly,
		Types:      types,
		ProjectID:  query.Get("projectId"),
		WorkItemID: query.Get("workItemId"),
		RunID:      query.Get("runId"),
		ThreadID:   query.Get("threadId"),
		Limit:      limit,
	}, nil
}

func parseNextMailRequest(r *http.Request) (app.NextMailRequest, error) {
	query := r.URL.Query()
	to, err := parseMailQueryAddresses(query["to"])
	if err != nil {
		return app.NextMailRequest{}, err
	}
	types, err := parseMailQueryCSV(query["types"])
	if err != nil {
		return app.NextMailRequest{}, err
	}
	timeoutMs := 0
	if raw := query.Get("timeoutMs"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return app.NextMailRequest{}, fmt.Errorf("invalid timeoutMs")
		}
		timeoutMs = parsed
	}
	return app.NextMailRequest{
		To:        to,
		Types:     types,
		Timeout:   time.Duration(timeoutMs) * time.Millisecond,
		ProjectID: query.Get("projectId"),
	}, nil
}

func parseMailQueryAddresses(values []string) ([]mailbox.Address, error) {
	var out []mailbox.Address
	for _, value := range values {
		addresses, err := mailbox.ParseAddressList(value)
		if err != nil {
			return nil, err
		}
		out = append(out, addresses...)
	}
	return mailbox.DeduplicateAddresses(out), nil
}

func parseMailQueryCSV(values []string) ([]string, error) {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
	}
	return out, nil
}

func toDomainMailAddress(address protocol.MailAddress) mailbox.Address {
	return mailbox.Address{Kind: address.Kind, ID: address.ID}
}

func toDomainMailAddressPtr(address *protocol.MailAddress) *mailbox.Address {
	if address == nil {
		return nil
	}
	out := toDomainMailAddress(*address)
	return &out
}

func toDomainMailAddresses(addresses []protocol.MailAddress) []mailbox.Address {
	out := make([]mailbox.Address, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, toDomainMailAddress(address))
	}
	return out
}

func toProtocolMailMessage(message mailbox.Message) protocol.MailMessage {
	return protocol.MailMessage{
		ID:         message.ID,
		ThreadID:   message.ThreadID,
		ReplyToID:  message.ReplyToID,
		From:       toProtocolMailAddress(message.From),
		Recipients: toProtocolMailRecipients(message.Recipients),
		Type:       message.Type,
		Priority:   message.Priority,
		Subject:    message.Subject,
		Body:       message.Body,
		Payload:    message.Payload,
		ProjectID:  message.ProjectID,
		WorkItemID: message.WorkItemID,
		RunID:      message.RunID,
		SessionID:  message.SessionID,
		PTYID:      message.PTYID,
		DispatchID: message.DispatchID,
		CreatedAt:  message.CreatedAt,
	}
}

func toProtocolMailMessages(messages []mailbox.Message) []protocol.MailMessage {
	out := make([]protocol.MailMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, toProtocolMailMessage(message))
	}
	return out
}

func toProtocolMailAddress(address mailbox.Address) protocol.MailAddress {
	return protocol.MailAddress{Kind: address.Kind, ID: address.ID}
}

func toProtocolMailRecipients(recipients []mailbox.Recipient) []protocol.MailRecipient {
	out := make([]protocol.MailRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		out = append(out, protocol.MailRecipient{
			Address: toProtocolMailAddress(recipient.Address),
			ReadAt:  recipient.ReadAt,
		})
	}
	return out
}
