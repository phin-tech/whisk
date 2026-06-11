package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 || args[0] != "daemon" {
		return fmt.Errorf("usage: whisk daemon <start|stop|status> [-url http://127.0.0.1:8787]")
	}

	flags := flag.NewFlagSet("daemon "+args[1], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args[2:]); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch args[1] {
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
		return fmt.Errorf("unknown daemon command %q", args[1])
	}
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
