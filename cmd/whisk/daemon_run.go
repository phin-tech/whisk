package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/plugins"
	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/adapters/sessionstore"
	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/adapters/workitemstore"
	"github.com/phin-tech/whisk/internal/adapters/worktrunk"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/events"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func runDaemonRun(args []string) error {
	flags := flag.NewFlagSet("daemon run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	addr := flags.String("addr", envOrDefault("WHISKD_ADDR", "127.0.0.1:8787"), "HTTP listen address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk daemon run [-addr 127.0.0.1:8787]")
	}
	return serveDaemon(*addr)
}

func serveDaemon(addr string) error {
	if err := validateListenAddr(addr); err != nil {
		return err
	}

	// Bind the listener before any other setup so a duplicate instance fails fast and
	// cheaply. Building NATS/sqlite/the runtime first would leave a heavyweight process
	// lingering only to discover the port is already taken — that is how duplicate
	// daemons piled up on the same address. If someone else already owns this addr we are
	// not needed, so exit cleanly rather than erroring.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			log.Printf("whisk daemon: another instance is already listening on %s; exiting", addr)
			return nil
		}
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	serveStarted := false
	defer func() {
		if !serveStarted {
			_ = listener.Close()
		}
	}()

	eventBus, err := events.NewNATSBus()
	if err != nil {
		return err
	}
	defer eventBus.Close()

	store, err := sessionstore.NewJSONStore("")
	if err != nil {
		return err
	}
	transcripts, err := transcriptstore.NewFileStore("")
	if err != nil {
		return err
	}
	workItems, err := workitemstore.NewSQLiteStore("")
	if err != nil {
		return err
	}
	settingsStore, err := appsettings.NewDefaultStore()
	if err != nil {
		return err
	}
	pluginManager, err := plugins.NewManager(pluginDirsFromEnv(), settingsStore)
	if err != nil {
		return err
	}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{
		PTYBackend:       native.NewBackend(),
		Worktrees:        worktrunk.NewBackendWithOptions(nil, worktrunk.BackendOptions{OverridePath: envOrDefault("WHISK_WORKTRUNK_PATH", "/opt/homebrew/bin/wt")}),
		Plugins:          pluginManager,
		EventSink:        eventBus,
		SessionStore:     store,
		TranscriptStore:  transcripts,
		WorkItemStore:    workItems,
		DaemonURL:        "http://" + addr,
		CLIPath:          whiskCLIPath(),
		DaemonAPIVersion: protocol.DaemonAPIVersion,
	})
	if err != nil {
		return err
	}
	defer func() { _ = runtime.Shutdown(context.Background()) }()

	shutdown := make(chan struct{})
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	mux.Handle("/", server.NewHTTP(runtime))
	mux.HandleFunc("POST /v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		select {
		case shutdown <- struct{}{}:
		default:
		}
	})

	serveErr := make(chan error, 1)
	serveStarted = true
	go func() {
		log.Printf("whisk daemon listening on http://%s", addr)
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case <-signals:
	case <-shutdown:
	case err := <-serveErr:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return <-serveErr
}

func pluginDirsFromEnv() []string {
	raw := os.Getenv("WHISK_PLUGIN_DIRS")
	if raw == "" {
		return nil
	}
	parts := filepath.SplitList(raw)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func trustedPluginsFromEnv() map[string]bool {
	out := map[string]bool{}
	for _, id := range strings.Split(os.Getenv("WHISK_TRUSTED_PLUGINS"), ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func validateListenAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("refusing non-loopback bind %q until daemon auth is implemented", addr)
	}
	return nil
}

func whiskCLIPath() string {
	candidates := []string{}
	if path := os.Getenv("WHISK_CLI"); path != "" {
		candidates = append(candidates, path)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, executable)
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "whisk"))
	}
	if path, err := exec.LookPath("whisk"); err == nil {
		candidates = append(candidates, path)
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err == nil && !info.IsDir() {
			return abs
		}
	}
	return ""
}
