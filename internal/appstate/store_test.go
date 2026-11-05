package appstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s != Default() {
		t.Fatalf("got %+v, want %+v", s, Default())
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := State{Volume: 42, Locked: true}

	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadDamagedFileIsMovedAside(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"volume": 4`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load(path)

	if err == nil {
		t.Fatal("damaged file did not produce an error")
	}
	if s != Default() {
		t.Fatalf("got %+v, want defaults", s)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("damaged file is still at its original path")
	}
	if _, statErr := os.Stat(path + ".corrupt"); statErr != nil {
		t.Fatalf("damaged file was not kept: %v", statErr)
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	for i := 0; i < 3; i++ {
		if err := Save(path, State{Volume: float64(i * 10)}); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("folder contains %v, want only state.json", names)
	}
}

func TestLoadClampsVolume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"volume": 250, "locked": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Volume != 100 || !s.Locked {
		t.Fatalf("got %+v, want volume 100 and locked", s)
	}
}
