package main

import (
	"fmt"
	"log"

	"github.com/getlantern/systray"

	"MicrophoneVolumeLock/internal/volumelock"
)

func (a *App) onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Microphone Volume Lock")

	show := systray.AddMenuItem("Show", "Show the window")
	lock := systray.AddMenuItem("Lock Volume", "Lock or unlock the microphone volume")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit Microphone Volume Lock")

	a.trayMu.Lock()
	a.lockItem = lock
	a.trayMu.Unlock()
	a.updateTray(a.ctl.Snapshot())

	// systray drops a click when nothing is receiving, so actions run on their own goroutine.
	go func() {
		for {
			select {
			case <-show.ClickedCh:
				a.showWindow()
			case <-lock.ClickedCh:
				go a.toggleFromTray()
			case <-quit.ClickedCh:
				a.Quit()
				return
			}
		}
	}()
}

func (a *App) toggleFromTray() {
	if err := a.ctl.Toggle(); err != nil {
		log.Printf("tray lock: %v", err)
	}
}

func (a *App) updateTray(s volumelock.Snapshot) {
	a.trayMu.Lock()
	defer a.trayMu.Unlock()
	if a.lockItem == nil {
		return
	}
	if s.Locked {
		a.lockItem.SetTitle("Unlock Volume")
	} else {
		a.lockItem.SetTitle("Lock Volume")
	}
	systray.SetTooltip(trayTooltip(s))
}

func trayTooltip(s volumelock.Snapshot) string {
	switch {
	case s.Locked && !s.Available:
		return "Microphone Volume Lock: waiting for a microphone"
	case !s.Available:
		return "Microphone Volume Lock: no microphone"
	case s.Locked:
		return fmt.Sprintf("Microphone Volume Lock: locked at %d%%", percent(s.Target))
	default:
		return fmt.Sprintf("Microphone Volume Lock: not locked (mic at %d%%)", percent(s.Level))
	}
}
