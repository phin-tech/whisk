package main

import (
	"context"
	"embed"
	"log"
	"os"
	"runtime"

	"github.com/phin-tech/whisk/internal/appmenu"
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
	ctx, cancel := context.WithTimeout(context.Background(), daemon.DefaultControlTimeout())
	defer cancel()
	if _, err := daemon.Ensure(ctx, daemonURL); err != nil {
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
		// Leave the daemon running by default so sessions persist across app restarts. Only stop
		// it when the user opted out via the KeepDaemonAlive preference and the current state file
		// still identifies a daemon this app owns.
		OnShutdown: func() {
			wailsapp.StopDaemonStatusWatcher(whisk)
			if !daemon.IsManaged(daemonURL) {
				return
			}
			settings, loadErr := settingsStore.Load(context.Background())
			if loadErr != nil || settings.KeepDaemonAlive {
				return
			}
			stopCtx, stopCancel := context.WithTimeout(context.Background(), daemon.DefaultControlTimeout())
			defer stopCancel()
			_ = daemon.Stop(stopCtx, daemonURL)
		},
	})
	wailsapp.AttachEventEmitter(whisk, desktop.Event)

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

	// Install the native menu bar (macOS-first) and let the service re-apply accelerators and the
	// session list to it at runtime. Done after the window exists so the app menu has somewhere to
	// attach.
	if runtime.GOOS == "darwin" {
		settings, loadErr := settingsStore.Load(context.Background())
		if loadErr != nil {
			settings = appsettings.Default()
		}
		menuController := appmenu.NewController(desktop, settings)
		wailsapp.AttachMenuController(whisk, menuController)
		menuController.Rebuild()
	}

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
