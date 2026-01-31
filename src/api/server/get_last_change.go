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
	return depanic(func() *connect.Response[api.GetLastChangeResponse] {
		response := getLastChangeImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func getLastChangeImpl(server *Server, req *api.GetLastChangeRequest) *api.GetLastChangeResponse {
	now := time.Now().UnixMilli()
	return &api.GetLastChangeResponse{
		MtimeMsecs: uint64(now),
	}
}
