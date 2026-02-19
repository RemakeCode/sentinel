package main

import (
	"context"
	"embed"
	"sentinel/backend"
	configFile "sentinel/backend/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := &backend.App{}
	configFile := &configFile.CfgFile{}

	// Create application with options
	err := wails.Run(&options.App{
		Title:    "Sentinel",
		Width:    1024,
		Height:   768,
		LogLevel: logger.ERROR,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			configFile.SetContext(ctx)

		},
		OnDomReady: app.DomReady,
		Bind: []interface{}{
			app,
			configFile,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
