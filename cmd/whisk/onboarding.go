package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runOnboarding(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk onboarding <status|apply>")
	}
	switch args[0] {
	case "status":
		return runOnboardingStatus(args[1:])
	case "apply":
		return runOnboardingApply(args[1:])
	default:
		return fmt.Errorf("usage: whisk onboarding <status|apply>")
	}
}

func runOnboardingStatus(args []string) error {
	flags := flag.NewFlagSet("onboarding status", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk onboarding status [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status, err := client.NewHTTP(*baseURL, nil).OnboardingStatus(ctx)
	if err != nil {
		return err
	}
	return printOnboarding(status, *outputJSON)
}

func runOnboardingApply(args []string) error {
	flags := flag.NewFlagSet("onboarding apply", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	selected := ""
	flags.StringVar(&selected, "items", "", "comma-separated item ids; defaults to selected status items")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk onboarding apply [-items id,id] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	daemon := client.NewHTTP(*baseURL, nil)
	status, err := daemon.OnboardingStatus(ctx)
	if err != nil {
		return err
	}
	itemIDs := selectedOnboardingItemIDs(status, selected)
	status, err = daemon.ApplyOnboarding(ctx, protocol.OnboardingApplyRequest{ItemIDs: itemIDs})
	if err != nil {
		return err
	}
	return printOnboarding(status, *outputJSON)
}

func selectedOnboardingItemIDs(status protocol.OnboardingStatus, selected string) []string {
	if strings.TrimSpace(selected) != "" {
		var ids []string
		for _, id := range strings.Split(selected, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	}
	var ids []string
	for _, item := range status.Items {
		if item.SelectedByDefault {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func printOnboarding(status protocol.OnboardingStatus, outputJSON bool) error {
	if outputJSON {
		return printJSON(status)
	}
	fmt.Printf("show=%v local=%v state=%s\n", status.ShouldShow, status.LocalDaemon, status.StatePath)
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "ID\tSTATUS\tSELECTED\tVERSION\tPATH")
	for _, item := range status.Items {
		fmt.Fprintf(writer, "%s\t%s\t%v\t%s\t%s\n", item.ID, item.Status, item.SelectedByDefault, item.LatestVersion, item.Path)
	}
	return nil
}
