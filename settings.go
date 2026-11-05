package main

import (
	"log"
	"sync"
	"time"

	"MicrophoneVolumeLock/internal/appstate"
)

const saveDelay = 400 * time.Millisecond

type settingsSaver struct {
	path string

	mu    sync.Mutex
	saved appstate.State
	want  *appstate.State
	timer *time.Timer
}

func newSettingsSaver(path string, saved appstate.State) *settingsSaver {
	return &settingsSaver{path: path, saved: saved}
}

func (s *settingsSaver) Want(st appstate.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if st == s.saved {
		s.want = nil
		s.stopTimerLocked()
		return
	}
	s.want = &st
	if st.Locked != s.saved.Locked {
		s.flushLocked()
		return
	}
	s.stopTimerLocked()
	s.timer = time.AfterFunc(saveDelay, s.Flush)
}

func (s *settingsSaver) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *settingsSaver) flushLocked() {
	s.stopTimerLocked()
	if s.want == nil {
		return
	}
	if err := appstate.Save(s.path, *s.want); err != nil {
		log.Printf("save settings: %v", err)
		return
	}
	s.saved = *s.want
	s.want = nil
}

func (s *settingsSaver) stopTimerLocked() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}
