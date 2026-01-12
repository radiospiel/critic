package must

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"git.15b.it/eno/critic/simple-go/logger"
	"github.com/samber/lo"
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

func escapeIfNecessary(s string, _ int) string {
	// TODO: make me better
	if strings.ContainsAny(s, "\"' \t\n\r") {
		return "'" + s + "'"
	}
	return s
}

// Exec executes a command, panicking on error.
func Exec(name string, args ...string) []byte {
	stringified := lo.Map(args, escapeIfNecessary)
	logger.Info("%s %s", name, strings.Join(stringified, " "))

	cmd := exec.Command(name, args...)
	output, err := cmd.Output()

	if err != nil {
		msg := fmt.Sprintf("in %s: Exec(%s %v) failed: %v", Getwd(), name, args, err)
		if err, ok := err.(*exec.ExitError); ok {
			msg = msg + "\nerrout: " + string(err.Stderr)
		}
		panic(msg)
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

// OpenFile opens a file with the specified flags and permissions, panicking on error.
func OpenFile(name string, flag int, perm os.FileMode) *os.File {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		panic(fmt.Sprintf("OpenFile(%s) failed: %v", name, err))
	}
	return f
}

// Getwd() returns the current working dir
func Getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Getwd() failed: %v", err))
	}
	return wd
}

func Chdir(dir string) {
	err := os.Chdir(dir)
	if err != nil {
		logger.Fatal("in %s: %v", Getwd(), err)
	}
}

func ParseInt(str string, base int) int64 {
	val, err := strconv.ParseInt(str, base, 64)
	if err != nil {
		logger.Fatal("cannot parseInt %s: %v", str, err)
	}
	return val
}
