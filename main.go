package main

import (
	"embed"
	"log"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"

	"sentinel/backend/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "Sentinel",
		Description: "Steam game emulator manager",
		Services: []application.Service{
			application.NewService(&config.File{}),
			application.NewService(&steam.GameBasics{}),
			application.NewService(&watcher.Service{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create the main window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Sentinel",
		MinWidth:  1280,
		MinHeight: 720,
		Width:     1920,
		Height:    1080,
		URL:       "/",
	})

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}
