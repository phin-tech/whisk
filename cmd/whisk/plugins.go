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

func runPlugin(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk plugin <list|registry|install|rescan|trust|untrust|attach|contributions|usage>")
	}
	switch args[0] {
	case "list":
		return runPluginList(args[1:])
	case "registry":
		return runPluginRegistry(args[1:])
	case "install":
		return runPluginInstall(args[1:])
	case "rescan":
		return runPluginRescan(args[1:])
	case "trust":
		return runPluginTrust(args[1:])
	case "untrust":
		return runPluginUntrust(args[1:])
	case "attach":
		return runPluginAttach(args[1:])
	case "contributions":
		return runPluginContributions(args[1:])
	case "usage":
		return runPluginUsage(args[1:])
	default:
		return fmt.Errorf("usage: whisk plugin <list|registry|install|rescan|trust|untrust|attach|contributions|usage>")
	}
}

func runPluginRegistry(args []string) error {
	flags := flag.NewFlagSet("plugin registry", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk plugin registry [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	plugins, err := client.NewHTTP(*baseURL, nil).ListRegistryPlugins(ctx)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(plugins)
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "REGISTRY\tID\tINSTALLED\tTRUSTED\tSOURCE\tNAME")
	for _, plugin := range plugins {
		fmt.Fprintf(writer, "%s\t%s\t%v\t%v\t%s\t%s\n", plugin.Registry, plugin.ID, plugin.Installed, plugin.Trusted, plugin.SourceType, plugin.Name)
	}
	return nil
}

func runPluginInstall(args []string) error {
	flags := flag.NewFlagSet("plugin install", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	registry := flags.String("registry", "", "registry to install from (defaults to the only configured registry)")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk plugin install <plugin-id>[@registry] [-registry name] [-json] [-url http://127.0.0.1:8787]")
	}
	// Accept "<id>@<registry>" shorthand in addition to the -registry flag.
	id := flags.Arg(0)
	reg := *registry
	if at := strings.LastIndex(id, "@"); at > 0 {
		reg = id[at+1:]
		id = id[:at]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	status, err := client.NewHTTP(*baseURL, nil).InstallPlugin(ctx, reg, id)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(status)
	}
	fmt.Printf("%s\tinstalled\ttrusted=%v\t%s\n", status.ID, status.Trusted, status.Dir)
	return nil
}

func runPluginList(args []string) error {
	flags := flag.NewFlagSet("plugin list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk plugin list [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	plugins, err := client.NewHTTP(*baseURL, nil).ListPlugins(ctx)
	if err != nil {
		return err
	}
	return printPlugins(plugins, *outputJSON)
}

func runPluginRescan(args []string) error {
	flags := flag.NewFlagSet("plugin rescan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk plugin rescan [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	plugins, err := client.NewHTTP(*baseURL, nil).RescanPlugins(ctx)
	if err != nil {
		return err
	}
	return printPlugins(plugins, *outputJSON)
}

func runPluginTrust(args []string) error {
	return runPluginTrustCommand("trust", args)
}

func runPluginUntrust(args []string) error {
	return runPluginTrustCommand("untrust", args)
}

func runPluginAttach(args []string) error {
	flags := flag.NewFlagSet("plugin attach", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	projectID := flags.String("project", "", "project id")
	values := map[string]string{}
	flags.Func("field", "template field key=value", func(value string) error {
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return fmt.Errorf("field must be key=value")
		}
		values[key] = val
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 || *projectID == "" {
		return fmt.Errorf("usage: whisk plugin attach <plugin-id> <template-id> -project <project-id> [-field key=value ...] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	project, err := client.NewHTTP(*baseURL, nil).RunPluginProjectAttachmentTemplate(ctx, flags.Arg(0), flags.Arg(1), protocol.RunPluginProjectAttachmentTemplateRequest{
		ProjectID: *projectID,
		Values:    values,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(project)
	}
	if len(project.Attachments) == 0 {
		return nil
	}
	fmt.Println(project.Attachments[len(project.Attachments)-1].ID)
	return nil
}

func runPluginContributions(args []string) error {
	flags := flag.NewFlagSet("plugin contributions", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	projectID := flags.String("project", "", "project id")
	workItemID := flags.String("work-item", "", "work item id")
	runID := flags.String("run", "", "run id")
	sessionID := flags.String("session", "", "session id")
	paneID := flags.String("pane", "", "pane id")
	ptyID := flags.String("pty", "", "pty id")
	gateReportID := flags.String("gate-report", "", "gate report id")
	phase := flags.String("phase", "", "workflow phase")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk plugin contributions [-project id] [-work-item id] [-run id] [-session id] [-pane id] [-pty id] [-gate-report id] [-phase phase] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	contributions, err := client.NewHTTP(*baseURL, nil).ListUIContributions(ctx, protocol.UIContributionScope{
		ProjectID:    *projectID,
		WorkItemID:   *workItemID,
		RunID:        *runID,
		SessionID:    *sessionID,
		PaneID:       *paneID,
		PTYID:        *ptyID,
		GateReportID: *gateReportID,
		Phase:        *phase,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(contributions)
	}
	return printUIContributions(contributions)
}

func runPluginUsage(args []string) error {
	if len(args) > 0 && args[0] == "refresh" {
		return runPluginUsageRefresh(args[1:])
	}
	flags := flag.NewFlagSet("plugin usage", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk plugin usage [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results, err := client.NewHTTP(*baseURL, nil).ListUsageResolvers(ctx)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(results)
	}
	return printUsageResolvers(results)
}

func runPluginUsageRefresh(args []string) error {
	flags := flag.NewFlagSet("plugin usage refresh", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	profile := flags.String("profile", "", "usage resolver profile")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return fmt.Errorf("usage: whisk plugin usage refresh <plugin-id> <resolver-id> [-profile profile] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	result, err := client.NewHTTP(*baseURL, nil).RefreshUsageResolver(ctx, flags.Arg(0), flags.Arg(1), protocol.RefreshUsageResolverRequest{
		Profile: *profile,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(result)
	}
	return printUsageResolvers([]protocol.UsageResolverReadModel{result})
}

func runPluginTrustCommand(command string, args []string) error {
	flags := flag.NewFlagSet("plugin "+command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk plugin %s <plugin-id> [-json] [-url http://127.0.0.1:8787]", command)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	daemon := client.NewHTTP(*baseURL, nil)
	var status protocol.PluginStatus
	var err error
	if command == "trust" {
		status, err = daemon.TrustPlugin(ctx, flags.Arg(0))
	} else {
		status, err = daemon.UntrustPlugin(ctx, flags.Arg(0))
	}
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(status)
	}
	fmt.Printf("%s\ttrusted=%v\n", status.ID, status.Trusted)
	return nil
}

func printPlugins(plugins []protocol.PluginStatus, outputJSON bool) error {
	if outputJSON {
		return printJSON(plugins)
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "ID\tTRUSTED\tVALID\tVERSION\tNAME")
	for _, plugin := range plugins {
		fmt.Fprintf(writer, "%s\t%v\t%v\t%s\t%s\n", plugin.ID, plugin.Trusted, plugin.Valid, plugin.Version, plugin.Name)
	}
	return nil
}

func printUIContributions(contributions protocol.UIContributionsResponse) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "PLUGIN\tTYPE\tSCOPE\tID\tLABEL")
	for _, plugin := range contributions.Plugins {
		for _, panel := range plugin.Panels {
			fmt.Fprintf(writer, "%s\tpanel\t%s\t%s\t%s\n", plugin.PluginID, panel.Scope, panel.ID, panel.Title)
		}
		for _, command := range plugin.Commands {
			fmt.Fprintf(writer, "%s\tcommand\t%s\t%s\t%s\n", plugin.PluginID, command.Scope, command.ID, command.Label)
		}
		for _, action := range plugin.ReviewActions {
			fmt.Fprintf(writer, "%s\treviewAction\t%s\t%s\t%s\n", plugin.PluginID, action.Scope, action.ID, action.Label)
		}
	}
	return nil
}

func printUsageResolvers(results []protocol.UsageResolverReadModel) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "PLUGIN\tRESOLVER\tPROVIDER\tPROFILE\tSTATUS\tSTALE\tREFRESHED\tSUMMARY\tERROR")
	for _, result := range results {
		refreshed := "-"
		if result.RefreshedAt != nil {
			refreshed = result.RefreshedAt.Format(time.RFC3339)
		}
		summary := ""
		if result.Result != nil {
			summary = result.Result.Summary
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%v\t%s\t%s\t%s\n",
			result.PluginID,
			result.ResolverID,
			result.Provider,
			result.Profile,
			result.Status,
			result.Stale,
			refreshed,
			summary,
			result.Error,
		)
	}
	return nil
}
