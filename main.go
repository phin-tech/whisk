package main

import (
	"embed"
	"log"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/wailsapp"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	whisk := wailsapp.NewService(runtime)

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
		Title:           "Whisk",
		Width:           1280,
		Height:          820,
		MinWidth:        960,
		MinHeight:       640,
		DevToolsEnabled: true,
		BackgroundColour: application.NewRGB(14, 18, 24),
		URL:             "/",
	})

	if err := desktop.Run(); err != nil {
		log.Fatal(err)
	}
}
