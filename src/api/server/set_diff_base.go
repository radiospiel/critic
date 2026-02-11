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
	start := req.GetStart()
	end := req.GetEnd()
	if start == "" {
		return nil, api.InvalidArgumentError("start is required")
	}

	// Check if start is in the list of available bases
	bases := server.session.GetDiffBases()
	isValidBase := func(ref string) bool {
		for _, b := range bases {
			if b == ref {
				return true
			}
		}
		return false
	}

	if !isValidBase(start) {
		return nil, api.InvalidArgumentError("invalid diff base: " + start)
	}

	// End is optional; if provided, must also be a valid base
	if end != "" && !isValidBase(end) {
		return nil, api.InvalidArgumentError("invalid diff end: " + end)
	}

	if err := server.session.SetCurrentDiffRange(start, end); err != nil {
		return nil, api.WrapError(err, "failed to set diff range")
	}

	return &api.SetDiffBaseResponse{
		Success: true,
	}, nil
}
