package appstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type State struct {
	Volume float64 `json:"volume"`
	Locked bool    `json:"locked"`
}

func Default() State {
	return State{Volume: 75}
}

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
	defer os.Remove(tmpName)

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
