package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runWorkItemRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk run <list|start|cancel>")
	}
	switch args[0] {
	case "list":
		return runRunList(args[1:])
	case "start":
		return runRunStart(args[1:])
	case "cancel":
		return runRunCancel(args[1:])
	default:
		return fmt.Errorf("usage: whisk run <list|start|cancel>")
	}
}

func runRunList(args []string) error {
	flags := flag.NewFlagSet("run list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	workItemID := flags.String("work-item", "", "work item id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk run list [-work-item id] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runs, err := client.NewHTTP(*baseURL, nil).ListWorkItemRuns(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(runs)
	}
	printRuns(runs)
	return nil
}

func runRunStart(args []string) error {
	flags := flag.NewFlagSet("run start", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	workItemID := flags.String("work-item", "", "work item id")
	preset := flags.String("preset", "", "capability preset")
	templateID := flags.String("template", "", "prompt template id")
	sessionID := flags.String("session", "", "session id")
	ptyID := flags.String("pty", "", "pty id")
	launch := flags.Bool("launch", true, "launch an agent PTY for the run")
	agentProfileID := flags.String("agent-profile", envOrDefault("WHISK_AGENT_PROFILE", "codex"), "agent profile id")
	systemPrompt := flags.String("system-prompt", "", "agent system prompt")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" {
		return fmt.Errorf("usage: whisk run start -work-item <id> [-preset writer] [-template implement] [-launch=false] [-agent-profile codex] [-system-prompt text] [-session id] [-pty id] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := client.NewHTTP(*baseURL, nil).StartWorkItemRun(ctx, protocol.StartWorkItemRunRequest{
		WorkItemID:       *workItemID,
		Preset:           *preset,
		PromptTemplateID: *templateID,
		SessionID:        *sessionID,
		PTYID:            *ptyID,
		Launch:           *launch,
		AgentProfileID:   *agentProfileID,
		SystemPrompt:     *systemPrompt,
		Actor:            *actor,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(run)
	}
	fmt.Printf("%s\t%s\t%s\t%s\t%s\n", run.ID, run.Status, run.Preset, run.SessionID, run.PTYID)
	return nil
}

func runRunCancel(args []string) error {
	flags := flag.NewFlagSet("run cancel", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk run cancel <run-id> [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := client.NewHTTP(*baseURL, nil).CancelWorkItemRun(ctx, protocol.CancelWorkItemRunRequest{
		ID:    flags.Arg(0),
		Actor: *actor,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(run)
	}
	fmt.Printf("%s\t%s\t%s\t%s\t%s\n", run.ID, run.Status, run.Preset, run.SessionID, run.PTYID)
	return nil
}

func printRuns(runs []protocol.WorkItemRun) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tWORK_ITEM\tSTATUS\tPRESET\tTEMPLATE\tSESSION\tPTY")
	for _, run := range runs {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", run.ID, run.WorkItemID, run.Status, run.Preset, run.PromptTemplateID, run.SessionID, run.PTYID)
	}
	writer.Flush()
}
