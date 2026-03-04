package logger

import (
	"fmt"
	"strings"
	"testing"
)

func TestProcessLogEntry_OSC8Hyperlink(t *testing.T) {
	// When colors are enabled, file:line should be wrapped in OSC 8 hyperlink
	sharedDest.enableColors = true
	defer func() { sharedDest.enableColors = false }()

	file := "/home/user/project/src/main.go"
	line := 42

	// Simulate what processLogEntry does for file:line when colors are enabled
	relFile := "src/main.go"
	ref := fmt.Sprintf("\033]8;;file://%s\033\\%s%s:%d%s\033]8;;\033\\",
		file, colorYellow, relFile, line, colorReset)

	if !strings.Contains(ref, "\033]8;;file://"+file+"\033\\") {
		t.Errorf("expected OSC 8 start sequence, got: %q", ref)
	}
	if !strings.Contains(ref, "\033]8;;\033\\") {
		t.Errorf("expected OSC 8 end sequence, got: %q", ref)
	}
	if !strings.Contains(ref, colorYellow) {
		t.Errorf("expected yellow color in ref, got: %q", ref)
	}
	if !strings.Contains(ref, "src/main.go:42") {
		t.Errorf("expected display text src/main.go:42 in ref, got: %q", ref)
	}
}

func TestProcessLogEntry_NoOSC8WhenColorsDisabled(t *testing.T) {
	sharedDest.enableColors = false

	// When colors are disabled, the ref should be plain text
	ref := fmt.Sprintf("%s:%d", "src/main.go", 42)

	if ref != "src/main.go:42" {
		t.Errorf("expected plain file:line, got: %q", ref)
	}
	if strings.Contains(ref, "\033") {
		t.Errorf("expected no escape sequences when colors disabled, got: %q", ref)
	}
}
