package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
)

// SetDiffBase sets the current diff base.
func (s *Server) SetDiffBase(
	ctx context.Context,
	req *connect.Request[api.SetDiffBaseRequest],
) (*connect.Response[api.SetDiffBaseResponse], error) {
	response := depanic(func() (*api.SetDiffBaseResponse, error) {
		return setDiffBaseImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func setDiffBaseImpl(server *Server, req *api.SetDiffBaseRequest) (*api.SetDiffBaseResponse, error) {
	base := req.GetBase()
	if base == "" {
		return nil, api.InvalidArgumentError("base is required")
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
		return nil, api.InvalidArgumentError("invalid diff base: " + base)
	}

	if err := server.session.SetCurrentDiffBase(base); err != nil {
		return nil, api.WrapError(err, "failed to set diff base")
	}

	return &api.SetDiffBaseResponse{
		Success: true,
	}, nil
}
