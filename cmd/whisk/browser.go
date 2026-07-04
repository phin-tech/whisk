package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	adapterbrowser "github.com/phin-tech/whisk/internal/adapters/browser"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/browser"
)

func runBrowser(args []string, deps runDeps) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk browser <diagnose>")
	}
	switch args[0] {
	case "diagnose":
		return runBrowserDiagnose(args[1:], deps)
	default:
		return fmt.Errorf("usage: whisk browser <diagnose>")
	}
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

func formatLaunchCommand(command string, args []string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\"'\\$`") {
		return strconv.Quote(value)
	}
	return value
}
