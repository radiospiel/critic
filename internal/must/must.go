package must

import (
	"fmt"
	"os"
	"os/exec"
)

// Must panics if err is not nil.
func Must(err error) {
	if err != nil {
		panic(fmt.Sprintf("Must() failed: %v", err))
	}
}

// Must2 panics if err is not nil, otherwise returns val.
func Must2[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("Must2() failed: %v", err))
	}
	return val
}

// WriteFile writes content to a file, panicking on error.
// Accepts either string or []byte.
//
// Examples:
// must.WriteFile("text.txt", "hello world")              // string
// must.WriteFile("binary.dat", []byte{0x00, 0x01, 0x02}) // []byte
func WriteFile(filename string, content any) {
	var data []byte
	switch v := content.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		panic(fmt.Sprintf("WriteFile(%s): unsupported type %T, expected string or []byte", filename, content))
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		panic(fmt.Sprintf("WriteFile(%s, %d byte) failed: %v", filename, len(data), err))
	}
}

// Exec executes a command, panicking on error.
func Exec(name string, args ...string) []byte {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("Exec(%s %v) failed: %v", name, args, err))
	}
	return output
}

// MkdirAll creates a directory and all parents, panicking on error.
func MkdirAll(path string, perm os.FileMode) {
	if err := os.MkdirAll(path, perm); err != nil {
		panic(fmt.Sprintf("MkdirAll(%s) failed: %v", path, err))
	}
}

// Remove removes a file or directory, panicking on error.
func Remove(path string) {
	if err := os.Remove(path); err != nil {
		panic(fmt.Sprintf("Remove(%s) failed: %v", path, err))
	}
}
