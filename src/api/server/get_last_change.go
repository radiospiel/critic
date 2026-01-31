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
	response := depanic(func() (*api.GetLastChangeResponse, error) {
		return getLastChangeImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getLastChangeImpl(server *Server, req *api.GetLastChangeRequest) (*api.GetLastChangeResponse, error) {
	now := time.Now().UnixMilli()
	return &api.GetLastChangeResponse{
		MtimeMsecs: uint64(now),
	}, nil
}
