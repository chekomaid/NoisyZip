//go:build gui
// +build gui

package main

import (
	"embed"
	"log"

	"noisyzip/internal/gui"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := gui.NewApp()

	err := wails.Run(&options.App{
		Title:     "NoisyZip",
		Width:     760,
		Height:    540,
		MinWidth:  760,
		MinHeight: 540,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: gui.StartupHandler(app),
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
