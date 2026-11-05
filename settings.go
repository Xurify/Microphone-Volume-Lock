package main

import (
	"log"
	"sync"
	"time"

	"MicrophoneVolumeLock/internal/appstate"
)

// saveDelay groups slider movements into one disk write.
const saveDelay = 400 * time.Millisecond

// settingsSaver writes state.json. Changes to the lock flag are written at
// once; slider movements are written after the slider has been still for
// saveDelay, so dragging does not hit the disk on every step.
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
