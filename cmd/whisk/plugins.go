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
		return fmt.Errorf("usage: whisk plugin <list|registry|install|rescan|trust|untrust|attach>")
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
	default:
		return fmt.Errorf("usage: whisk plugin <list|registry|install|rescan|trust|untrust|attach>")
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
	fmt.Fprintln(writer, "ID\tINSTALLED\tTRUSTED\tSOURCE\tNAME")
	for _, plugin := range plugins {
		fmt.Fprintf(writer, "%s\t%v\t%v\t%s\t%s\n", plugin.ID, plugin.Installed, plugin.Trusted, plugin.SourceType, plugin.Name)
	}
	return nil
}

func runPluginInstall(args []string) error {
	flags := flag.NewFlagSet("plugin install", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk plugin install <plugin-id> [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	status, err := client.NewHTTP(*baseURL, nil).InstallPlugin(ctx, flags.Arg(0))
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
