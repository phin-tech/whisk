package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	adapterbrowser "github.com/phin-tech/whisk/internal/adapters/browser"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/browser"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runBrowser(args []string, deps runDeps) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk browser <attach|list|targets|detach|diagnose>")
	}
	switch args[0] {
	case "attach":
		return runBrowserAttach(args[1:])
	case "list":
		return runBrowserList(args[1:])
	case "targets":
		return runBrowserTargets(args[1:])
	case "detach":
		return runBrowserDetach(args[1:])
	case "diagnose":
		return runBrowserDiagnose(args[1:], deps)
	default:
		return fmt.Errorf("usage: whisk browser <attach|list|targets|detach|diagnose>")
	}
}

func runBrowserAttach(args []string) error {
	flags := flag.NewFlagSet("browser attach", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	cdpURL := flags.String("cdp-url", envOrDefault("WHISK_BROWSER_CDP_URL", ""), "loopback Chrome CDP endpoint")
	name := flags.String("name", "", "resource display name")
	timeout := flags.Duration("timeout", 5*time.Second, "daemon request timeout")
	outputJSON := flags.Bool("json", false, "write JSON output")
	acknowledge := false
	flags.BoolVar(&acknowledge, "acknowledge-browser-control-risk", false, "acknowledge Chrome CDP browser-control risk")
	flags.BoolVar(&acknowledge, "ack", false, "acknowledge Chrome CDP browser-control risk")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *cdpURL == "" {
		return fmt.Errorf("usage: whisk browser attach -cdp-url http://127.0.0.1:9222 -acknowledge-browser-control-risk [-name name] [-timeout 5s] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout+time.Second)
	defer cancel()
	resource, err := client.NewHTTP(*baseURL, nil).ConnectBrowserResource(ctx, protocol.ConnectBrowserResourceRequest{
		Name:                          *name,
		CDPURL:                        *cdpURL,
		AcknowledgeBrowserControlRisk: acknowledge,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(resource)
	}
	return printBrowserResources([]protocol.BrowserResource{resource})
}

func runBrowserList(args []string) error {
	flags := flag.NewFlagSet("browser list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk browser list [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resources, err := client.NewHTTP(*baseURL, nil).ListBrowserResources(ctx)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(resources)
	}
	return printBrowserResources(resources)
}

func runBrowserTargets(args []string) error {
	flags := flag.NewFlagSet("browser targets", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk browser targets [-json] [-url http://127.0.0.1:8787] <resource-id>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	targets, err := client.NewHTTP(*baseURL, nil).ListBrowserTargets(ctx, flags.Arg(0))
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(targets)
	}
	return printBrowserTargets(targets)
}

func runBrowserDetach(args []string) error {
	flags := flag.NewFlagSet("browser detach", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk browser detach [-json] [-url http://127.0.0.1:8787] <resource-id>")
	}
	resourceID := flags.Arg(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.NewHTTP(*baseURL, nil).DisconnectBrowserResource(ctx, resourceID); err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(map[string]any{"resourceId": resourceID, "detached": true})
	}
	fmt.Printf("detached %s\n", resourceID)
	return nil
}

func runBrowserDiagnose(args []string, deps runDeps) error {
	flags := flag.NewFlagSet("browser diagnose", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	cdpURL := flags.String("cdp-url", envOrDefault("WHISK_BROWSER_CDP_URL", ""), "loopback Chrome CDP endpoint")
	timeout := flags.Duration("timeout", 5*time.Second, "CDP probe timeout")
	outputJSON := flags.Bool("json", false, "write JSON output")
	chromePath := flags.String("chrome-path", "", "Chrome executable path for launch-command preview")
	userDataDir := flags.String("user-data-dir", "", "dedicated Chrome profile dir for launch-command preview")
	debuggingPort := flags.Int("debugging-port", browser.DefaultDebuggingPort, "loopback CDP port for launch-command preview")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk browser diagnose [-cdp-url http://127.0.0.1:9222] [-timeout 5s] [-json] [-chrome-path path -user-data-dir dir [-debugging-port 9222]]")
	}

	if deps.browserDiagnose == nil {
		deps.browserDiagnose = defaultRunDeps().browserDiagnose
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout+time.Second)
	defer cancel()
	result, err := deps.browserDiagnose(ctx, app.BrowserDiagnosticRequest{
		CDPURL:        *cdpURL,
		Timeout:       *timeout,
		ChromePath:    *chromePath,
		UserDataDir:   *userDataDir,
		DebuggingPort: launchDebuggingPort(*chromePath, *userDataDir, *debuggingPort),
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(result)
	}
	return printBrowserDiagnostic(result)
}

func defaultBrowserDiagnose(ctx context.Context, req app.BrowserDiagnosticRequest) (app.BrowserDiagnostic, error) {
	service := app.NewBrowserDiagnosticService(adapterbrowser.NewCDPProbe(nil))
	return service.Diagnose(ctx, req)
}

func launchDebuggingPort(chromePath string, userDataDir string, debuggingPort int) int {
	if chromePath == "" && userDataDir == "" {
		return 0
	}
	return debuggingPort
}

func printBrowserDiagnostic(result app.BrowserDiagnostic) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "STATUS\tCDP URL\tBROWSER\tTARGETS\tERROR")
	fmt.Fprintf(writer, "%s\t%s\t%s\t%d\t%s\n", result.Status, result.CDPURL, result.Browser, result.TargetCount, result.Error)
	if result.LaunchCommand != nil {
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "LAUNCH COMMAND\tENDPOINT")
		fmt.Fprintf(writer, "%s\t%s\n", formatLaunchCommand(result.LaunchCommand.Command, result.LaunchCommand.Args), result.LaunchCommand.Endpoint)
	}
	return nil
}

func printBrowserResources(resources []protocol.BrowserResource) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tCDP URL\tCONNECTED")
	for _, resource := range resources {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%t\n", resource.ID, resource.Name, resource.CDPURL, resource.Connected)
	}
	return writer.Flush()
}

func printBrowserTargets(targets []protocol.BrowserTarget) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tRESOURCE\tTYPE\tSTATUS\tTITLE\tURL")
	for _, target := range targets {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", target.ID, target.ResourceID, target.Type, target.Status, target.Title, target.URL)
	}
	return writer.Flush()
}

func formatLaunchCommand(command string, args []string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if isShellSafeUnquoted(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isShellSafeUnquoted(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("_@%+=:,./-", r):
		default:
			return false
		}
	}
	return true
}
