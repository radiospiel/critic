package critic

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestFileState_String(t *testing.T) {
	tests := []struct {
		state FileState
		want  string
	}{
		{FileCreated, "created"},
		{FileDeleted, "deleted"},
		{FileChanged, "changed"},
		{FileState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equals(t, tt.state.String(), tt.want)
		})
	}
}
