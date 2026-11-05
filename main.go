package main

import (
	"embed"
	"log"
	"os"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var iconBytes []byte

// Keep in sync with info.productVersion in wails.json.
var version = "1.2.0"

const (
	uniqueID        = "com.xurify.microphone-volume-lock"
	startHiddenFlag = "--hidden"
	contentHeight   = 226
	titleBarHeight  = 32
	windowHeight    = contentHeight + titleBarHeight
)

func main() {
	closeLog := setupLogging()
	defer closeLog()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "Microphone Volume Lock",
		Width:             400,
		Height:            windowHeight,
		MinWidth:          300,
		MaxWidth:          400,
		MinHeight:         windowHeight,
		MaxHeight:         windowHeight,
		Assets:            assets,
		BackgroundColour:  &options.RGBA{R: 17, G: 17, B: 16, A: 255},
		StartHidden:       slices.Contains(os.Args[1:], startHiddenFlag),
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               uniqueID,
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
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
		log.Printf("wails: %v", err)
	}
}
