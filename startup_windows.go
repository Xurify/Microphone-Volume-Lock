//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "MicrophoneVolumeLock"
)

func startupEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(runValueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val != "", nil
}

// refreshStartupEntry rewrites an existing Run entry in the current format.
// Version 1.0 wrote it without quotes or startHiddenFlag, and the path goes
// stale if the app is installed somewhere else.
func refreshStartupEntry() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(runValueName)
	if errors.Is(err, registry.ErrNotExist) || val == "" {
		return nil
	}
	if err != nil {
		return err
	}
	want, err := startupCommand()
	if err != nil || val == want {
		return err
	}
	return k.SetStringValue(runValueName, want)
}

func startupCommand() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s" %s`, exe, startHiddenFlag), nil
}

// setStartupEnabled writes or removes the Run entry. The entry carries
// startHiddenFlag so that a login start goes straight to the tray.
func setStartupEnabled(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !enabled {
		if err := k.DeleteValue(runValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}

	cmd, err := startupCommand()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValueName, cmd)
}
