package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/protocol"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runWithDeps(args, defaultRunDeps())
}

type runDeps struct {
	context      func() (context.Context, context.CancelFunc)
	startForward func(ctx context.Context, baseURL string, req protocol.StartHTTPForwardRequest) (protocol.StartedHTTPForward, func(context.Context) error, error)
}

func defaultRunDeps() runDeps {
	return runDeps{
		context: func() (context.Context, context.CancelFunc) {
			return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		},
		startForward: startLocalForward,
	}
}

func runWithDeps(args []string, deps runDeps) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk <daemon|forward>")
	}
	switch args[0] {
	case "daemon":
		return runDaemon(args[1:])
	case "forward":
		return runForward(args[1:], deps)
	default:
		return fmt.Errorf("usage: whisk <daemon|forward>")
	}
}

func runDaemon(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: whisk daemon <start|stop|status> [-url http://127.0.0.1:8787]")
	}

	flags := flag.NewFlagSet("daemon "+args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch args[0] {
	case "start":
		if err := daemon.Ensure(ctx, *baseURL); err != nil {
			return err
		}
		fmt.Printf("whiskd running at %s\n", *baseURL)
		return nil
	case "status":
		if err := client.NewHTTP(*baseURL, nil).Health(ctx); err != nil {
			return fmt.Errorf("whiskd unavailable at %s: %w", *baseURL, err)
		}
		fmt.Printf("whiskd running at %s\n", *baseURL)
		return nil
	case "stop":
		return stop(ctx, *baseURL)
	default:
		return fmt.Errorf("unknown daemon command %q", args[0])
	}
}

func runForward(args []string, deps runDeps) error {
	if len(args) == 0 || args[0] != "create" {
		return fmt.Errorf("usage: whisk forward create <target-url> [-name name] [-url http://127.0.0.1:8787]")
	}
	targetURL, name, baseURL, err := parseForwardCreate(args[1:])
	if err != nil {
		return err
	}
	if deps.context == nil {
		deps.context = defaultRunDeps().context
	}
	if deps.startForward == nil {
		deps.startForward = defaultRunDeps().startForward
	}
	ctx, cancel := deps.context()
	defer cancel()
	started, stopForward, err := deps.startForward(ctx, baseURL, protocol.StartHTTPForwardRequest{
		Name:      name,
		TargetURL: targetURL,
	})
	if err != nil {
		return err
	}
	fmt.Println(started.LocalURL)
	<-ctx.Done()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	return stopForward(stopCtx)
}

func parseForwardCreate(args []string) (targetURL string, name string, baseURL string, err error) {
	baseURL = envOrDefault("WHISKD_URL", "http://127.0.0.1:8787")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-name":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("-name requires a value")
			}
			name = args[i]
		case "-url":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("-url requires a value")
			}
			baseURL = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", fmt.Errorf("unknown forward create flag %q", args[i])
			}
			if targetURL != "" {
				return "", "", "", fmt.Errorf("usage: whisk forward create <target-url> [-name name] [-url http://127.0.0.1:8787]")
			}
			targetURL = args[i]
		}
	}
	if targetURL == "" {
		return "", "", "", fmt.Errorf("usage: whisk forward create <target-url> [-name name] [-url http://127.0.0.1:8787]")
	}
	return targetURL, name, baseURL, nil
}

func startLocalForward(ctx context.Context, baseURL string, req protocol.StartHTTPForwardRequest) (protocol.StartedHTTPForward, func(context.Context) error, error) {
	daemonClient := client.NewHTTP(baseURL, nil)
	forwarder := client.NewLocalForwarder(daemonClient, nil)
	started, err := forwarder.Start(ctx, req)
	if err != nil {
		return protocol.StartedHTTPForward{}, nil, err
	}
	return started, func(ctx context.Context) error {
		return forwarder.Stop(ctx, started.ID)
	}, nil
}

func stop(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/shutdown", bytes.NewReader(nil))
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("stop whiskd: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		if response.StatusCode == http.StatusNotFound {
			if err := daemon.StopPID(baseURL); err == nil {
				fmt.Printf("whiskd stopped at %s\n", baseURL)
				return nil
			}
		}
		return fmt.Errorf("stop whiskd: %s", response.Status)
	}
	fmt.Printf("whiskd stopped at %s\n", baseURL)
	return nil
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
