package server

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
)

// GetLastChange returns the current time in milliseconds.
func (s *Server) GetLastChange(
	ctx context.Context,
	req *connect.Request[api.GetLastChangeRequest],
) (*connect.Response[api.GetLastChangeResponse], error) {
	now := time.Now().UnixMilli()
	res := connect.NewResponse(&api.GetLastChangeResponse{
		MtimeMsecs: uint64(now),
	})
	return res, nil
}
