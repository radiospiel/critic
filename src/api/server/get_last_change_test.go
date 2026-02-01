package server

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
)

func TestGetLastChange_ReturnsTimestamp(t *testing.T) {
	s := &Server{
		config: Config{},
	}

	// Record time before call
	before := time.Now().UnixMilli()

	req := connect.NewRequest(&api.GetLastChangeRequest{})
	resp, err := s.GetLastChange(context.Background(), req)

	// Record time after call
	after := time.Now().UnixMilli()

	assert.NoError(t, err, "GetLastChange should not return error")
	assert.True(t, resp.Msg.GetMtimeMsecs() >= uint64(before), "timestamp should be >= before")
	assert.True(t, resp.Msg.GetMtimeMsecs() <= uint64(after), "timestamp should be <= after")
}

func TestGetLastChange_ReturnsTrackedTimestamp(t *testing.T) {
	s := &Server{
		config: Config{},
	}

	// Set a specific tracked timestamp
	trackedTime := int64(1234567890123)
	s.SetLastChangeTime(trackedTime)

	req := connect.NewRequest(&api.GetLastChangeRequest{})
	resp, err := s.GetLastChange(context.Background(), req)

	assert.NoError(t, err, "GetLastChange should not return error")
	assert.Equals(t, resp.Msg.GetMtimeMsecs(), uint64(trackedTime), "should return tracked timestamp")
}

func TestGetLastChange_DefaultsToCurrentTime(t *testing.T) {
	s := &Server{
		config: Config{},
	}

	// Don't set any tracked time - should default to current time
	before := time.Now().UnixMilli()

	req := connect.NewRequest(&api.GetLastChangeRequest{})
	resp, err := s.GetLastChange(context.Background(), req)

	after := time.Now().UnixMilli()

	assert.NoError(t, err, "GetLastChange should not return error")
	assert.True(t, resp.Msg.GetMtimeMsecs() >= uint64(before), "timestamp should be >= before")
	assert.True(t, resp.Msg.GetMtimeMsecs() <= uint64(after), "timestamp should be <= after")
}
