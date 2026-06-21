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

func runPrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk prompt <list|resolve>")
	}
	switch args[0] {
	case "list":
		return runPromptList(args[1:])
	case "resolve":
		return runPromptResolve(args[1:])
	default:
		return fmt.Errorf("usage: whisk prompt <list|resolve>")
	}
}

func runPromptList(args []string) error {
	flags := flag.NewFlagSet("prompt list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	status := flags.String("status", "pending", "prompt status")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk prompt list [-status pending] [-json] [-url http://127.0.0.1:8787]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prompts, err := client.NewHTTP(*baseURL, nil).ListAgentPrompts(ctx, protocol.ListAgentPromptsRequest{Status: *status})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(prompts)
	}
	printPrompts(prompts)
	return nil
}

func runPromptResolve(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk prompt resolve <id> -answer <value> [-json] [-url http://127.0.0.1:8787]")
	}
	promptID := args[0]
	flags := flag.NewFlagSet("prompt resolve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	answer := flags.String("answer", "", "answer value")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *answer == "" {
		return fmt.Errorf("usage: whisk prompt resolve <id> -answer <value> [-json] [-url http://127.0.0.1:8787]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prompt, err := client.NewHTTP(*baseURL, nil).ResolveAgentPrompt(ctx, promptID, protocol.ResolveAgentPromptRequest{Answer: *answer})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(prompt)
	}
	fmt.Printf("%s\t%s\t%s\n", prompt.ID, prompt.Status, prompt.Answer)
	return nil
}

func printPrompts(prompts []protocol.AgentPrompt) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tKIND\tPROVIDER\tMESSAGE")
	for _, prompt := range prompts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", prompt.ID, prompt.Kind, prompt.Provider, prompt.Message)
	}
	_ = w.Flush()
}
