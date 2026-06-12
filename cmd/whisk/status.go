package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runStatus(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk status <question|done|blocked> -message <text> [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	kind := args[0]
	switch kind {
	case workitem.StatusKindQuestion, workitem.StatusKindDone, workitem.StatusKindBlocked:
	default:
		return fmt.Errorf("usage: whisk status <question|done|blocked> -message <text> [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}

	flags := flag.NewFlagSet("status "+kind, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	message := flags.String("message", "", "status message")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	projectID := flags.String("project", envOrDefault("WHISK_PROJECT_ID", envOrDefault("WHISK_PROJECT", "")), "project id")
	workItemID := flags.String("work-item", envOrDefault("WHISK_WORK_ITEM_ID", ""), "work item id")
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	sessionID := flags.String("session", envOrDefault("WHISK_SESSION_ID", ""), "session id")
	ptyID := flags.String("pty", envOrDefault("WHISK_PTY_ID", ""), "pty id")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *message == "" {
		return fmt.Errorf("usage: whisk status <question|done|blocked> -message <text> [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	if *runID == "" && *sessionID == "" && *ptyID == "" && *workItemID == "" {
		return fmt.Errorf("status context required; run inside a Whisk PTY or pass -run, -session, -pty, or -work-item")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	report, err := client.NewHTTP(*baseURL, nil).ReportStatus(ctx, protocol.ReportStatusRequest{
		Kind:       kind,
		Message:    *message,
		Actor:      *actor,
		ProjectID:  *projectID,
		WorkItemID: *workItemID,
		RunID:      *runID,
		SessionID:  *sessionID,
		PTYID:      *ptyID,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(report)
	}
	fmt.Printf("%s\t%s\t%s\n", report.Event.ID, report.Event.Kind, report.Event.Scope)
	return nil
}
