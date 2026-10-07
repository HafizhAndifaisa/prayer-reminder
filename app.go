package main

import (
	"log"
	"os"
	"path/filepath"
)

// version is the application version, kept in sync with the bundle manifests
// (Info.plist, info.json) and frontend/package.json.
const version = "1.0.1"

// Shared setting keys used by the store.
const (
	settingLocation = "location"
)

// logWarn logs a non-fatal error. Kept as a small helper so call sites stay
// readable and we can later route logs to a file or the UI.
func logWarn(what string, err error) {
	if err != nil {
		log.Printf("warn: %s: %v", what, err)
	}
}

// dataDir returns the application's data directory, creating it if needed.
// It follows the macOS convention of ~/Library/Application Support/<AppName>.
func dataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "PrayerReminder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
