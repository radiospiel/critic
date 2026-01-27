package version

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

var (
	version     string
	versionOnce sync.Once
)

// Version returns the binary version based on its modification timestamp.
func Version() string {
	versionOnce.Do(func() {
		version = computeVersion()
	})
	return version
}

func computeVersion() string {
	exe, err := os.Executable()
	if err != nil {
		logger.Info("Could not determine executable path: %v", err)
		return "unknown"
	}

	info, err := os.Stat(exe)
	if err != nil {
		logger.Info("Could not stat executable: %v", err)
		return "unknown"
	}

	return info.ModTime().UTC().Format(time.RFC3339)
}

// versionFileName is the name of the file that stores the last seen version.
const versionFileName = ".critic.version"

// IsFirstRunForVersion checks if this is the first time running this version.
// Returns true if:
// - The version file doesn't exist
// - The version file contains a different version
func IsFirstRunForVersion(gitRoot string) bool {
	versionFile := filepath.Join(gitRoot, versionFileName)

	data, err := os.ReadFile(versionFile)
	if err != nil {
		// File doesn't exist or can't be read - this is a first run
		logger.Info("Version file not found or unreadable: %v", err)
		return true
	}

	lastVersion := strings.TrimSpace(string(data))
	if lastVersion != Version() {
		logger.Info("Version changed from %s to %s", lastVersion, Version())
		return true
	}

	return false
}

// MarkVersionSeen writes the current version to the version file.
// This should be called after showing the screensaver to prevent
// showing it again for the same version.
func MarkVersionSeen(gitRoot string) error {
	versionFile := filepath.Join(gitRoot, versionFileName)
	return os.WriteFile(versionFile, []byte(Version()+"\n"), 0644)
}
