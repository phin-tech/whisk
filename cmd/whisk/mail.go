package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runMail(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk mail <send|list|check|read|reply>")
	}
	switch args[0] {
	case "send":
		return runMailSend(args[1:])
	case "list":
		return runMailList(args[1:])
	case "check", "next":
		return runMailCheck(args[1:])
	case "read":
		return runMailRead(args[1:])
	case "reply":
		return runMailReply(args[1:])
	default:
		return fmt.Errorf("usage: whisk mail <send|list|check|read|reply>")
	}
}

func runMailSend(args []string) error {
	flags := flag.NewFlagSet("mail send", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	fromRaw := flags.String("from", "", "sender address")
	toRaw := flags.String("to", "", "comma-separated recipient addresses")
	messageType := flags.String("type", "", "message type")
	priority := flags.String("priority", mailbox.PriorityNormal, "message priority")
	subject := flags.String("subject", "", "message subject")
	body := flags.String("body", "", "message body")
	bodyFile := flags.String("body-file", "", "read message body from file")
	payloadRaw := flags.String("payload-json", "", "JSON payload")
	threadID := flags.String("thread", "", "thread id")
	replyToID := flags.String("reply-to", "", "reply target mail id")
	projectID := flags.String("project", envOrDefault("WHISK_PROJECT_ID", envOrDefault("WHISK_PROJECT", "")), "project id")
	workItemID := flags.String("work-item", envOrDefault("WHISK_WORK_ITEM_ID", ""), "work item id")
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	sessionID := flags.String("session", envOrDefault("WHISK_SESSION_ID", ""), "session id")
	ptyID := flags.String("pty", envOrDefault("WHISK_PTY_ID", ""), "pty id")
	dispatchID := flags.String("dispatch", envOrDefault("WHISK_DISPATCH_ID", ""), "dispatch id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *toRaw == "" || *messageType == "" {
		return fmt.Errorf("usage: whisk mail send -to <addr>[,<addr>] [-from <addr>] -type <type> [-priority normal] [-subject text] [-body text|-body-file path] [-payload-json json] [-thread id] [-reply-to id] [-project id] [-work-item id] [-run id] [-session id] [-pty id] [-dispatch id] [-json] [-url URL]")
	}
	from, err := mailFromFlagOrEnv(*fromRaw)
	if err != nil {
		return err
	}
	to, err := parseProtocolAddressList(*toRaw)
	if err != nil {
		return err
	}
	bodyValue, payload, err := mailBodyAndPayload(*body, *bodyFile, *payloadRaw)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	message, err := client.NewHTTP(*baseURL, nil).SendMail(ctx, protocol.SendMailRequest{
		From:       from,
		To:         to,
		Type:       *messageType,
		Priority:   *priority,
		Subject:    *subject,
		Body:       bodyValue,
		Payload:    payload,
		ThreadID:   *threadID,
		ReplyToID:  *replyToID,
		ProjectID:  *projectID,
		WorkItemID: *workItemID,
		RunID:      *runID,
		SessionID:  *sessionID,
		PTYID:      *ptyID,
		DispatchID: *dispatchID,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(message)
	}
	printMailMessages([]protocol.MailMessage{message})
	return nil
}

func runMailList(args []string) error {
	flags := flag.NewFlagSet("mail list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	toRaw := flags.String("to", "", "comma-separated recipient addresses")
	unreadOnly := flags.Bool("unread", false, "only unread messages")
	typesRaw := flags.String("types", "", "comma-separated message types")
	projectID := flags.String("project", "", "project id")
	workItemID := flags.String("work-item", "", "work item id")
	runID := flags.String("run", "", "run id")
	threadID := flags.String("thread", "", "thread id")
	limit := flags.Int("limit", 0, "maximum messages")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk mail list [-to addr[,addr]] [-unread] [-types csv] [-project id] [-work-item id] [-run id] [-thread id] [-limit n] [-json] [-url URL]")
	}
	to, err := parseOptionalProtocolAddressList(*toRaw)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	messages, err := client.NewHTTP(*baseURL, nil).ListMail(ctx, protocol.ListMailRequest{
		To:         to,
		UnreadOnly: *unreadOnly,
		Types:      parseCSV(*typesRaw),
		ProjectID:  *projectID,
		WorkItemID: *workItemID,
		RunID:      *runID,
		ThreadID:   *threadID,
		Limit:      *limit,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(messages)
	}
	printMailMessages(messages)
	return nil
}

func runMailCheck(args []string) error {
	flags := flag.NewFlagSet("mail check", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	toRaw := flags.String("to", "", "comma-separated recipient addresses")
	typesRaw := flags.String("types", "", "comma-separated message types")
	wait := flags.Bool("wait", false, "wait until a matching message arrives")
	timeout := flags.Duration("timeout", 10*time.Minute, "maximum wait duration")
	ack := flags.Bool("ack", false, "mark the returned message read")
	projectID := flags.String("project", "", "project id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk mail check [-to addr[,addr]] [-types csv] [-wait] [-timeout 10m] [-ack] [-json] [-url URL]")
	}
	to, err := parseOptionalProtocolAddressList(*toRaw)
	if err != nil {
		return err
	}
	if len(to) == 0 {
		to = defaultMailCheckAddresses()
	}
	if len(to) == 0 {
		return fmt.Errorf("mail check recipient required; run inside a Whisk PTY or pass -to")
	}
	timeoutMs := 0
	ctxTimeout := 5 * time.Second
	if *wait {
		timeoutMs = int(timeout.Milliseconds())
		ctxTimeout = *timeout + 5*time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	stopHeartbeat := startMailCheckHeartbeat(ctx, *wait)
	defer stopHeartbeat()

	daemonClient := client.NewHTTP(*baseURL, nil)
	response, err := daemonClient.NextMail(ctx, protocol.NextMailRequest{
		To:        to,
		Types:     parseCSV(*typesRaw),
		TimeoutMs: timeoutMs,
		ProjectID: *projectID,
	})
	if err != nil {
		return err
	}
	if *ack && response.Message != nil {
		read, err := daemonClient.MarkMailRead(ctx, response.Message.ID, protocol.MarkMailReadRequest{To: ackAddress(to)})
		if err != nil {
			return err
		}
		response.Message = &read
	}
	if *outputJSON {
		return printJSON(response)
	}
	if response.Timeout || response.Message == nil {
		fmt.Println("no mail")
		return nil
	}
	printMailMessages([]protocol.MailMessage{*response.Message})
	return nil
}

func runMailRead(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk mail read <mail-id> [-to addr] [-json] [-url URL]")
	}
	mailID := args[0]
	flags := flag.NewFlagSet("mail read", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	toRaw := flags.String("to", "", "recipient address")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk mail read <mail-id> [-to addr] [-json] [-url URL]")
	}
	to, err := mailReadAddress(*toRaw)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	message, err := client.NewHTTP(*baseURL, nil).MarkMailRead(ctx, mailID, protocol.MarkMailReadRequest{To: to})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(message)
	}
	printMailMessages([]protocol.MailMessage{message})
	return nil
}

func runMailReply(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk mail reply <mail-id> [-from addr] [-type status] [-priority normal] [-subject text] [-body text|-body-file path] [-payload-json json] [-json] [-url URL]")
	}
	mailID := args[0]
	flags := flag.NewFlagSet("mail reply", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	fromRaw := flags.String("from", "", "sender address")
	messageType := flags.String("type", mailbox.TypeStatus, "message type")
	priority := flags.String("priority", mailbox.PriorityNormal, "message priority")
	subject := flags.String("subject", "", "message subject")
	body := flags.String("body", "", "message body")
	bodyFile := flags.String("body-file", "", "read message body from file")
	payloadRaw := flags.String("payload-json", "", "JSON payload")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk mail reply <mail-id> [-from addr] [-type status] [-priority normal] [-subject text] [-body text|-body-file path] [-payload-json json] [-json] [-url URL]")
	}
	from, err := mailFromFlagOrEnv(*fromRaw)
	if err != nil {
		return err
	}
	bodyValue, payload, err := mailBodyAndPayload(*body, *bodyFile, *payloadRaw)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	message, err := client.NewHTTP(*baseURL, nil).ReplyMail(ctx, mailID, protocol.ReplyMailRequest{
		From:     from,
		Type:     *messageType,
		Priority: *priority,
		Subject:  *subject,
		Body:     bodyValue,
		Payload:  payload,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(message)
	}
	printMailMessages([]protocol.MailMessage{message})
	return nil
}

func mailFromFlagOrEnv(raw string) (protocol.MailAddress, error) {
	if raw != "" {
		return parseProtocolAddress(raw)
	}
	for _, candidate := range []struct {
		kind string
		env  string
	}{
		{kind: mailbox.AddressKindPTY, env: "WHISK_PTY_ID"},
		{kind: mailbox.AddressKindRun, env: "WHISK_RUN_ID"},
		{kind: mailbox.AddressKindSession, env: "WHISK_SESSION_ID"},
	} {
		if value := os.Getenv(candidate.env); value != "" {
			return protocol.MailAddress{Kind: candidate.kind, ID: value}, nil
		}
	}
	return protocol.MailAddress{}, fmt.Errorf("mail sender required; run inside a Whisk PTY or pass -from")
}

func defaultMailCheckAddresses() []protocol.MailAddress {
	var addresses []protocol.MailAddress
	for _, candidate := range []struct {
		kind string
		envs []string
	}{
		{kind: mailbox.AddressKindPTY, envs: []string{"WHISK_PTY_ID"}},
		{kind: mailbox.AddressKindRun, envs: []string{"WHISK_RUN_ID"}},
		{kind: mailbox.AddressKindSession, envs: []string{"WHISK_SESSION_ID"}},
		{kind: mailbox.AddressKindWorkItem, envs: []string{"WHISK_WORK_ITEM_ID"}},
		{kind: mailbox.AddressKindProject, envs: []string{"WHISK_PROJECT_ID", "WHISK_PROJECT"}},
	} {
		for _, env := range candidate.envs {
			if value := os.Getenv(env); value != "" {
				addresses = append(addresses, protocol.MailAddress{Kind: candidate.kind, ID: value})
				break
			}
		}
	}
	return addresses
}

func mailReadAddress(raw string) (*protocol.MailAddress, error) {
	if raw != "" {
		address, err := parseProtocolAddress(raw)
		if err != nil {
			return nil, err
		}
		return &address, nil
	}
	address, err := mailFromFlagOrEnv("")
	if err != nil {
		return nil, nil
	}
	return &address, nil
}

func ackAddress(to []protocol.MailAddress) *protocol.MailAddress {
	if len(to) == 1 {
		return &to[0]
	}
	return nil
}

func mailBodyAndPayload(body string, bodyFile string, payloadRaw string) (string, json.RawMessage, error) {
	if body != "" && bodyFile != "" {
		return "", nil, fmt.Errorf("use either -body or -body-file, not both")
	}
	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", nil, err
		}
		body = string(data)
	}
	var payload json.RawMessage
	if strings.TrimSpace(payloadRaw) != "" {
		payload = json.RawMessage(payloadRaw)
		if !json.Valid(payload) {
			return "", nil, fmt.Errorf("payload-json must be valid JSON")
		}
	}
	return body, payload, nil
}

func parseProtocolAddress(raw string) (protocol.MailAddress, error) {
	address, err := mailbox.ParseAddress(raw)
	if err != nil {
		return protocol.MailAddress{}, err
	}
	return protocol.MailAddress{Kind: address.Kind, ID: address.ID}, nil
}

func parseProtocolAddressList(raw string) ([]protocol.MailAddress, error) {
	addresses, err := mailbox.ParseAddressList(raw)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.MailAddress, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, protocol.MailAddress{Kind: address.Kind, ID: address.ID})
	}
	return out, nil
}

func parseOptionalProtocolAddressList(raw string) ([]protocol.MailAddress, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	return parseProtocolAddressList(raw)
}

func parseCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func startMailCheckHeartbeat(ctx context.Context, enabled bool) func() {
	if !enabled {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Fprintln(os.Stderr, "waiting for mail...")
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

func printMailMessages(messages []protocol.MailMessage) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tTYPE\tPRIORITY\tFROM\tTO\tSUBJECT")
	for _, message := range messages {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
			message.ID,
			message.Type,
			message.Priority,
			mailAddressString(message.From),
			mailRecipientsString(message.Recipients),
			message.Subject,
		)
	}
	_ = writer.Flush()
}

func mailAddressString(address protocol.MailAddress) string {
	if address.Kind == "" && address.ID == "" {
		return ""
	}
	return address.Kind + ":" + address.ID
}

func mailRecipientsString(recipients []protocol.MailRecipient) string {
	values := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		values = append(values, mailAddressString(recipient.Address))
	}
	return strings.Join(values, ",")
}
