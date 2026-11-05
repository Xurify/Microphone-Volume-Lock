//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// legacyCopyPattern matches the file names version 1.0 gave to its extracted
// copy of nircmdc.exe.
const legacyCopyPattern = "nircmdc_*.exe"

// cleanupLegacyNircmd stops copies of nircmdc.exe left running by version 1.0,
// which extracted nircmdc.exe to the temp folder and never stopped it when the
// window closed. Only processes running exactly such a file are touched.
func cleanupLegacyNircmd() {
	tempDir := filepath.Clean(os.TempDir())

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		log.Printf("legacy cleanup: process snapshot: %v", err)
	} else {
		defer windows.CloseHandle(snapshot)

		var entry windows.ProcessEntry32
		entry.Size = uint32(unsafe.Sizeof(entry))
		for err := windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
			image, ok := imagePath(entry.ProcessID)
			if !ok || !isLegacyCopy(image, tempDir) {
				continue
			}
			if err := terminate(entry.ProcessID); err != nil {
				log.Printf("legacy cleanup: stop %s (pid %d): %v", image, entry.ProcessID, err)
				continue
			}
			log.Printf("stopped legacy nircmd process %s (pid %d)", image, entry.ProcessID)
		}
	}

	matches, _ := filepath.Glob(filepath.Join(tempDir, legacyCopyPattern))
	for _, m := range matches {
		if err := os.Remove(m); err == nil {
			log.Printf("removed legacy copy %s", m)
		}
	}
}

func imagePath(pid uint32) (string, bool) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, 32768) // long-path limit
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", false
	}
	return windows.UTF16ToString(buf[:size]), true
}

func isLegacyCopy(image, tempDir string) bool {
	dir, base := filepath.Split(filepath.Clean(image))
	if !strings.EqualFold(filepath.Clean(dir), tempDir) {
		return false
	}
	ok, _ := filepath.Match(legacyCopyPattern, strings.ToLower(base))
	return ok
}

func terminate(pid uint32) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.TerminateProcess(h, 0)
}
