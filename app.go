package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

const (
	registryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName      = "MicrophoneVolumeLock"
)

type App struct {
	ctx       context.Context
	locked    bool
	volume    float64
	nircmdCmd *exec.Cmd
}

func NewApp() *App {
	state, err := loadState()
	if err != nil {
		return &App{
			volume: 75,
			locked: false,
		}
	}

	app := &App{
		volume: state.Volume,
		locked: state.Locked,
	}

	return app
}

func (a *App) createMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// File Menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Hide to Tray", keys.CmdOrCtrl("H"), func(_ *menu.CallbackData) {
		a.HideWindow()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Exit", keys.CmdOrCtrl("Q"), func(_ *menu.CallbackData) {
		a.stopNircmd()
		runtime.Quit(a.ctx)
	})

	// Options Menu
	optionsMenu := appMenu.AddSubmenu("Options")
	startupEnabled, _ := a.IsStartupEnabled()
	optionsMenu.AddCheckbox("Run at Windows Startup", startupEnabled, keys.CmdOrCtrl("S"), func(_ *menu.CallbackData) {
		if err := a.SetStartupEnabled(startupEnabled); err != nil {
			runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Error",
				Message: fmt.Sprintf("Failed to set startup option: %v", err),
			})
		}
	})

	// Help Menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("About", nil, func(_ *menu.CallbackData) {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "About",
			Message: "Microphone Volume Lock v1.0\nA tool to lock your microphone volume.",
		})
	})

	return appMenu
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.locked {
		err := a.setMicrophoneVolume(a.volume)
		if err != nil {
			a.locked = false
			_ = a.saveCurrentState()
		}
	}
	go systray.Run(a.onReady, a.onExit)
}

func (a *App) saveCurrentState() error {
	return saveState(AppState{
		Volume: a.volume,
		Locked: a.locked,
	})
}

func (a *App) SetVolume(volume float64) error {
	if !a.locked {
		a.volume = volume
		return a.saveCurrentState()
	}
	return fmt.Errorf("volume is locked")
}

func (a *App) GetVolume() float64 {
	return a.volume
}

func (a *App) IsLocked() bool {
	return a.locked
}

func (a *App) ToggleLock() error {
	if a.locked {
		a.locked = false
		a.stopNircmd()
	} else {
		a.locked = true
		err := a.setMicrophoneVolume(a.volume)
		if err != nil {
			a.locked = false
			return err
		}
	}

	return a.saveCurrentState()
}

func (a *App) setMicrophoneVolume(volume float64) error {
	volumeInt := int(volume / 100 * 65535)
	a.nircmdCmd = exec.Command("nircmdc.exe", "loop", "172800", "500", "setsysvolume", fmt.Sprintf("%d", volumeInt), "default_record")
	a.nircmdCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := a.nircmdCmd.Start()
	if err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: fmt.Sprintf("Failed to set volume: %v", err),
		})
		return err
	}
	return nil
}

func (a *App) stopNircmd() {
	if a.nircmdCmd != nil && a.nircmdCmd.Process != nil {
		nircmdKillCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", a.nircmdCmd.Process.Pid))
		nircmdKillCmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
		err := nircmdKillCmd.Start()
		if err != nil {
			runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Error",
				Message: fmt.Sprintf("Error stopping nircmd: %v", err),
			})
		}
		a.nircmdCmd = nil
	}
}

func (a *App) IsStartupEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(appName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return val != "", nil
}

func (a *App) SetStartupEnabled(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enabled {
		execPath, err := os.Executable()
		if err != nil {
			return err
		}
		execPath = filepath.Clean(execPath)
		err = k.SetStringValue(appName, execPath)
		if err != nil {
			return err
		}
	} else {
		err = k.DeleteValue(appName)
		if err != nil && err != registry.ErrNotExist {
			return err
		}
	}

	return nil
}

func (a *App) StopAllNircmd() {
	cmd := exec.Command("taskkill", "/IM", "nircmdc.exe", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := cmd.Start()
	if err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: fmt.Sprintf("Error stopping all nircmd processes: %v", err),
		})
	}
}

func (a *App) onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("Microphone Volume Lock")
	systray.SetTooltip("Microphone Volume Lock")

	showItem := systray.AddMenuItem("Show", "Show the application window")
	lockItem := systray.AddMenuItem("Lock Volume", "Lock/Unlock microphone volume")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-showItem.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-lockItem.ClickedCh:
				if err := a.ToggleLock(); err != nil {
					fmt.Printf("Error toggling lock: %v\n", err)
				}
				if a.IsLocked() {
					lockItem.SetTitle("Unlock Volume")
					runtime.EventsEmit(a.ctx, "lock-state-changed", true)
				} else {
					lockItem.SetTitle("Lock Volume")
					runtime.EventsEmit(a.ctx, "lock-state-changed", false)
				}
			case <-quitItem.ClickedCh:
				a.stopNircmd()
				systray.Quit()
				runtime.Quit(a.ctx)
				return
			}
		}
	}()

	runtime.EventsOn(a.ctx, "update-tray-lock", func(data ...interface{}) {
		if len(data) > 0 {
			if isLocked, ok := data[0].(bool); ok {
				text := "Lock Volume"
				if isLocked {
					text = "Unlock Volume"
				}
				lockItem.SetTitle(text)
			}
		}
	})
}

func (a *App) onExit() {
	a.stopNircmd()
	_ = a.saveCurrentState()
}

func (a *App) HideWindow() {
	runtime.WindowHide(a.ctx)
}

func getIcon() []byte {
	return iconBytes
}
