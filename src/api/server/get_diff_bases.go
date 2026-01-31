package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
)

// GetDiffBases returns the available diff bases and the current selection.
func (s *Server) GetDiffBases(
	ctx context.Context,
	req *connect.Request[api.GetDiffBasesRequest],
) (*connect.Response[api.GetDiffBasesResponse], error) {
	response := depanic(func() (*api.GetDiffBasesResponse, error) {
		return getDiffBasesImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getDiffBasesImpl(server *Server, req *api.GetDiffBasesRequest) (*api.GetDiffBasesResponse, error) {
	return &api.GetDiffBasesResponse{
		Bases:       server.session.GetDiffBases(),
		CurrentBase: server.session.GetCurrentBase(),
	}, nil
}
