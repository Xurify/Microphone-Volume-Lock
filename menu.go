package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) createMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Hide to Tray", keys.CmdOrCtrl("H"), func(*menu.CallbackData) {
		a.hideWindow()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Exit", keys.CmdOrCtrl("Q"), func(*menu.CallbackData) {
		a.quit()
	})

	optionsMenu := appMenu.AddSubmenu("Options")
	startupOn, _ := startupEnabled()
	a.startupItem = optionsMenu.AddCheckbox("Start with Windows", startupOn, keys.CmdOrCtrl("S"), func(d *menu.CallbackData) {
		a.setStartupFromMenu(d.MenuItem.Checked)
	})

	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("About", nil, func(*menu.CallbackData) {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "About",
			Message: fmt.Sprintf("Microphone Volume Lock %s\nKeeps your microphone volume at a fixed level.", version),
		})
	})

	return appMenu
}
