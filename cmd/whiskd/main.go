package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/bookmarkstore"
	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/adapters/sessionstore"
	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/adapters/worktrunk"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/events"
	"github.com/phin-tech/whisk/internal/server"
)

func main() {
	addr := flag.String("addr", envOrDefault("WHISKD_ADDR", "127.0.0.1:8787"), "HTTP listen address")
	flag.Parse()
	if err := validateListenAddr(*addr); err != nil {
		log.Fatal(err)
	}

	eventBus, err := events.NewNATSBus()
	if err != nil {
		log.Fatal(err)
	}
	defer eventBus.Close()

	store, err := sessionstore.NewJSONStore("")
	if err != nil {
		log.Fatal(err)
	}
	transcripts, err := transcriptstore.NewFileStore("")
	if err != nil {
		log.Fatal(err)
	}
	bookmarks, err := bookmarkstore.NewJSONStore("")
	if err != nil {
		log.Fatal(err)
	}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{
		PTYBackend:      native.NewBackend(),
		Worktrees:       worktrunk.NewBackend(nil),
		EventSink:       eventBus,
		SessionStore:    store,
		TranscriptStore: transcripts,
		BookmarkStore:   bookmarks,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = runtime.Shutdown(context.Background()) }()

	shutdown := make(chan struct{})
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:              *addr,
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

	go func() {
		log.Printf("whiskd listening on http://%s", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve whiskd: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case <-signals:
	case <-shutdown:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown whiskd: %v", err)
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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
