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

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/server"
)

func main() {
	addr := flag.String("addr", envOrDefault("WHISKD_ADDR", "127.0.0.1:8787"), "HTTP listen address")
	flag.Parse()
	if err := validateListenAddr(*addr); err != nil {
		log.Fatal(err)
	}

	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	defer func() { _ = runtime.Shutdown(context.Background()) }()

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.NewHTTP(runtime),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("whiskd listening on http://%s", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve whiskd: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals

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
