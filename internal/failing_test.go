package internal

import "testing"

// This test is intentionally failing to verify CI catches failures.
// Remove this file after confirming CI works.
func TestIntentionallyFailing(t *testing.T) {
	t.Error("This test intentionally fails to verify CI is working")
}
