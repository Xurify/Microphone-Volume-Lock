package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var iconBytes []byte

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Microphone Volume Lock",
		Width:            400,
		Height:           215,
		MinWidth:         400,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 27, G: 27, B: 27, A: 1},
		OnStartup:        app.startup,
		Menu:             app.createMenu(),
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			IsZoomControlEnabled: false,
			Theme:                windows.Dark,
			BackdropType:         windows.Mica,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
