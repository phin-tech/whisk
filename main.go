package main

import (
	"context"
	"embed"
	"log"
	"os"
	"time"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/wailsapp"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The GUI app binary ("whisk-app") must never act as the daemon. If it is ever launched
	// with daemon-style arguments (e.g. because daemon discovery mistakenly selected this
	// executable), refuse instead of silently starting the GUI and re-spawning ourselves,
	// which would fork-loop. Daemon commands belong to the separate "whisk" CLI.
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		log.Fatal("whisk-app is the GUI application; run the `whisk` CLI for daemon commands")
	}

	daemonURL := envOrDefault("WHISKD_URL", "http://127.0.0.1:8787")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	startedDaemon, err := daemon.Ensure(ctx, daemonURL)
	if err != nil {
		log.Fatal(err)
	}

	settingsStore, settingsErr := appsettings.NewDefaultStore()
	if settingsErr != nil {
		log.Fatal(settingsErr)
	}
	whisk := wailsapp.NewServiceWithSettings(client.NewHTTP(daemonURL, nil), settingsStore)

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
		// Only tear down a daemon this process started. A daemon already running when the app
		// launched (e.g. `whisk daemon run` in a dev terminal) was adopted, not owned, so we
		// leave it alive. This stops app-spawned daemons from piling up across restarts without
		// killing one the developer is managing themselves.
		OnShutdown: func() {
			if startedDaemon {
				_ = daemon.StopPID(daemonURL)
			}
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
