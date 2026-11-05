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

// version is set at build time: -ldflags "-X main.version=1.1.0".
var version = "dev"

const (
	// uniqueID identifies the running instance to Wails' single-instance check.
	uniqueID = "com.xurify.microphone-volume-lock"
	// startHiddenFlag is passed by the Windows startup entry.
	startHiddenFlag = "--hidden"
	// windowHeight is the content height (192px, measured in the browser) plus
	// an estimated 56px for the Windows title bar and menu bar. Not resizable.
	windowHeight = 248
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
		BackgroundColour:  &options.RGBA{R: 27, G: 27, B: 27, A: 1},
		StartHidden:       slices.Contains(os.Args[1:], startHiddenFlag),
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               uniqueID,
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		OnStartup:  app.startup,
		OnShutdown: app.onWailsShutdown,
		Menu:       app.createMenu(),
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
