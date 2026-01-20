package version

import (
	"os"
	"path/filepath"
	"strings"

	"git.15b.it/eno/critic/simple-go/logger"
)

// Version is the current binary version.
// This should be updated with each release.
const Version = "0.1.0"

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
	if lastVersion != Version {
		logger.Info("Version changed from %s to %s", lastVersion, Version)
		return true
	}

	return false
}

// MarkVersionSeen writes the current version to the version file.
// This should be called after showing the screensaver to prevent
// showing it again for the same version.
func MarkVersionSeen(gitRoot string) error {
	versionFile := filepath.Join(gitRoot, versionFileName)
	return os.WriteFile(versionFile, []byte(Version+"\n"), 0644)
}
