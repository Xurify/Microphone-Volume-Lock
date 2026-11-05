package main

import (
	"log"
	"os"
	"path/filepath"
)

const maxLogBytes = 1 << 20

// appDataDir is the per-user folder for settings and the log, for example
// %AppData%\MicrophoneVolumeLock on Windows.
func appDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "MicrophoneVolumeLock"), nil
}

func statePath() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func setupLogging() func() {
	noop := func() {}
	dir, err := appDataDir()
	if err != nil || os.MkdirAll(dir, 0o755) != nil {
		return noop
	}

	path := filepath.Join(dir, "app.log")
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogBytes {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return noop
	}
	log.SetOutput(f)
	return func() { _ = f.Close() }
}
