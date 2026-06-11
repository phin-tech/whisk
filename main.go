package main

import (
	"context"
	"embed"
	"log"
	"os"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/wailsapp"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	daemonURL := envOrDefault("WHISKD_URL", "http://127.0.0.1:8787")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := daemon.Ensure(ctx, daemonURL); err != nil {
		log.Fatal(err)
	}

	whisk := wailsapp.NewService(client.NewHTTP(daemonURL, nil))

	desktop := application.New(application.Options{
		Name:        "Whisk",
		Description: "Terminal multiplexer and agent manager",
		Services: []application.Service{
			application.NewService(whisk),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Whisk",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		DevToolsEnabled:  true,
		BackgroundColour: application.NewRGB(14, 18, 24),
		URL:              "/",
	})

	if err := desktop.Run(); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
