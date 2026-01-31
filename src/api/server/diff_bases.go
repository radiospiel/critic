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
	response := depanic2(func() (*api.GetDiffBasesResponse, error) {
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

// SetDiffBase sets the current diff base.
func (s *Server) SetDiffBase(
	ctx context.Context,
	req *connect.Request[api.SetDiffBaseRequest],
) (*connect.Response[api.SetDiffBaseResponse], error) {
	response := depanic2(func() (*api.SetDiffBaseResponse, error) {
		return setDiffBaseImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func setDiffBaseImpl(server *Server, req *api.SetDiffBaseRequest) (*api.SetDiffBaseResponse, error) {
	base := req.GetBase()
	if base == "" {
		return &api.SetDiffBaseResponse{
			Success: false,
			Error:   api.InvalidArgument("base is required"),
		}, nil
	}

	// Check if the base is in the list of available bases
	bases := server.session.GetDiffBases()
	found := false
	for _, b := range bases {
		if b == base {
			found = true
			break
		}
	}
	if !found {
		return &api.SetDiffBaseResponse{
			Success: false,
			Error:   api.InvalidArgument("invalid diff base: " + base),
		}, nil
	}

	err := server.session.SetCurrentDiffBase(base)
	if err != nil {
		return &api.SetDiffBaseResponse{
			Success: false,
			Error:   api.InternalError(err.Error()),
		}, nil
	}

	return &api.SetDiffBaseResponse{
		Success: true,
	}, nil
}
