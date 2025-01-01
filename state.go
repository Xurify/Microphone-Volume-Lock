package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppState struct {
	Volume float64 `json:"volume"`
	Locked bool    `json:"locked"`
}

func getStateFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "MicrophoneVolumeLock")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "state.json"), nil
}

func saveState(state AppState) error {
	stateFile, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

func loadState() (AppState, error) {
	state := AppState{
		Volume: 75,
		Locked: false,
	}

	stateFile, err := getStateFilePath()
	if err != nil {
		return state, err
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return state, err
	}

	return state, nil
}
