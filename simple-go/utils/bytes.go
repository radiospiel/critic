package utils

import (
	"strings"
)

// BytesToString converts a byte slice to a string without allocating.
// It uses strings.Builder for efficient conversion.
func BytesToString(b []byte) string {
	var buf strings.Builder
	buf.Write(b)
	return buf.String()
}

// StringToBytes converts a string to a byte slice without allocating.
// It performs a direct copy into a newly allocated byte slice.
func StringToBytes(s string) []byte {
	b := make([]byte, len(s))
	copy(b, s)
	return b
}