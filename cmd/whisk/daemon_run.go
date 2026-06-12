package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/bookmarkstore"
	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/adapters/sessionstore"
	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/adapters/workitemstore"
	"github.com/phin-tech/whisk/internal/adapters/worktrunk"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/events"
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
	bookmarks, err := bookmarkstore.NewJSONStore("")
	if err != nil {
		return err
	}
	workItems, err := workitemstore.NewSQLiteStore("")
	if err != nil {
		return err
	}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{
		PTYBackend:      native.NewBackend(),
		Worktrees:       worktrunk.NewBackend(nil),
		EventSink:       eventBus,
		SessionStore:    store,
		TranscriptStore: transcripts,
		BookmarkStore:   bookmarks,
		WorkItemStore:   workItems,
		DaemonURL:       "http://" + addr,
		CLIPath:         whiskCLIPath(),
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
	go func() {
		log.Printf("whisk daemon listening on http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
