package main

import (
	"context"
	"errors"
	"log"
	"math"
	"slices"
	"sync"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"MicrophoneVolumeLock/internal/appstate"
	"MicrophoneVolumeLock/internal/volumelock"
)

type endpoint interface {
	volumelock.Device
	Close() error
}

type brokenDevice struct{ err error }

func (b brokenDevice) Level() (float64, error) { return 0, b.err }
func (b brokenDevice) SetLevel(float64) error  { return b.err }
func (b brokenDevice) Close() error            { return nil }

type App struct {
	ctx   context.Context
	dev   endpoint
	ctl   *volumelock.Controller
	saver *settingsSaver

	trayMu   sync.Mutex
	lockItem *systray.MenuItem

	shutdownOnce sync.Once
}

type State struct {
	Level     int    `json:"level"`
	Target    int    `json:"target"`
	Locked    bool   `json:"locked"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

func NewApp() *App {
	a := &App{}

	var dev endpoint
	captured, err := newCaptureDevice()
	if err != nil {
		log.Printf("audio: %v", err)
		dev = brokenDevice{err: err}
	} else {
		dev = captured
	}
	a.dev = dev
	a.ctl = volumelock.New(dev, volumelock.Options{OnChange: a.onVolumeChange})
	return a
}

func (a *App) Version() string {
	return version
}

func (a *App) GetState() State {
	return toState(a.ctl.Snapshot())
}

func (a *App) SetLevel(percent float64) (State, error) {
	err := a.ctl.SetLevel(percent / 100)
	return toState(a.ctl.Snapshot()), err
}

func (a *App) ToggleLock() (State, error) {
	err := a.ctl.Toggle()
	return toState(a.ctl.Snapshot()), err
}

func (a *App) IsStartupEnabled() (bool, error) {
	return startupEnabled()
}

func (a *App) SetStartupEnabled(enabled bool) (bool, error) {
	err := setStartupEnabled(enabled)
	now, readErr := startupEnabled()
	if readErr == nil {
		a.syncStartup(now)
	}
	if err == nil {
		err = readErr
	}
	return now, err
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cleanupLegacyNircmd()
	if err := refreshStartupEntry(); err != nil {
		log.Printf("refresh startup entry: %v", err)
	}

	path, err := statePath()
	if err != nil {
		log.Printf("settings path: %v", err)
	}
	saved, err := appstate.Load(path)
	if err != nil {
		log.Printf("settings: %v", err)
	}
	a.saver = newSettingsSaver(path, saved)

	a.ctl.Restore(saved.Volume/100, saved.Locked)
	a.ctl.Run()
	go systray.Run(a.onReady, a.onExit)
}

func (a *App) shutdown(context.Context) {
	a.shutdownOnce.Do(func() {
		a.ctl.Close()
		if a.saver != nil {
			a.saver.Flush()
		}
		if err := a.dev.Close(); err != nil {
			log.Printf("close audio device: %v", err)
		}
	})
}

func (a *App) onVolumeChange(s volumelock.Snapshot) {
	if s.Err != nil {
		log.Printf("microphone: %v", s.Err)
	}
	if a.saver != nil {
		a.saver.Want(appstate.State{Volume: math.Round(s.Target * 100), Locked: s.Locked})
	}
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "state", toState(s))
	a.updateTray(s)
}

func (a *App) onSecondInstance(data options.SecondInstanceData) {
	if slices.Contains(data.Args, startHiddenFlag) {
		return
	}
	a.showWindow()
}

func (a *App) showWindow() {
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
}

func (a *App) Quit() {
	systray.Quit()
	runtime.Quit(a.ctx)
}

func (a *App) onExit() {
	a.shutdown(a.ctx)
}

func (a *App) syncStartup(on bool) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "startup", on)
	}
}

func toState(s volumelock.Snapshot) State {
	st := State{
		Level:     percent(s.Level),
		Target:    percent(s.Target),
		Locked:    s.Locked,
		Available: s.Available,
	}
	if s.Err != nil {
		st.Message = userMessage(s.Err)
	}
	return st
}

func userMessage(err error) string {
	if errors.Is(err, errNoMicrophone) {
		return "No microphone found."
	}
	return "Microphone is not responding. Retrying…"
}

func percent(v float64) int {
	return int(math.Round(v * 100))
}
