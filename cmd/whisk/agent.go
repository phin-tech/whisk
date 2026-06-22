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

func runAgent(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk agent <profiles>")
	}
	switch args[0] {
	case "profiles":
		return runAgentProfiles(args[1:])
	default:
		return fmt.Errorf("usage: whisk agent <profiles>")
	}
}

func runAgentProfiles(args []string) error {
	flags := flag.NewFlagSet("agent profiles", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk agent profiles [-json] [-url http://127.0.0.1:8787]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	profiles, err := client.NewHTTP(*baseURL, nil).ListAgentProfiles(ctx)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(profiles)
	}
	printAgentProfiles(profiles)
	return nil
}

func printAgentProfiles(profiles []protocol.AgentProfile) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tLABEL\tDESCRIPTION")
	for _, profile := range profiles {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", profile.ID, profile.Provider, profile.Label, profile.Description)
	}
	_ = w.Flush()
}
