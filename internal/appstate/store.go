// Package appstate stores the settings that survive a restart.
package appstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// State is saved between runs. Volume is a percentage from 0 to 100.
type State struct {
	Volume float64 `json:"volume"`
	Locked bool    `json:"locked"`
}

func Default() State {
	return State{Volume: 75}
}

// Load reads the state stored at path. A missing file gives Default and no
// error. A damaged file is renamed to path+".corrupt" so it can be inspected,
// and Load returns Default together with an error that says so.
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}

	s := Default()
	if err := json.Unmarshal(data, &s); err != nil {
		aside := path + ".corrupt"
		if rerr := os.Rename(path, aside); rerr != nil {
			return Default(), fmt.Errorf("state file is damaged (%v) and could not be moved aside: %w", err, rerr)
		}
		return Default(), fmt.Errorf("state file is damaged, moved to %s: %w", aside, err)
	}
	s.Volume = clampVolume(s.Volume)
	return s, nil
}

// Save writes s to path. The data goes to a temporary file in the same folder,
// which replaces path only after it has been fully written and flushed to disk.
// A crash mid-save therefore leaves either the old file or the new one.
func Save(path string, s State) error {
	s.Volume = clampVolume(s.Volume)
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // harmless once the rename has moved the file

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func clampVolume(v float64) float64 {
	switch {
	case v != v || v < 0: // v != v is true only for NaN
		return 0
	case v > 100:
		return 100
	default:
		return v
	}
}
