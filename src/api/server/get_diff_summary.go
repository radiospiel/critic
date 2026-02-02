package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

// GetDiffSummary returns the current diff summary (file list without hunks) and state.
func (s *Server) GetDiffSummary(
	ctx context.Context,
	req *connect.Request[api.GetDiffSummaryRequest],
) (*connect.Response[api.GetDiffSummaryResponse], error) {
	response := depanic(func() (*api.GetDiffSummaryResponse, error) {
		return getDiffSummaryImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getDiffSummaryImpl(server *Server, req *api.GetDiffSummaryRequest) (*api.GetDiffSummaryResponse, error) {
	session := server.GetSession()
	state := session.GetState()
	diff := session.GetDiffSummary()

	return &api.GetDiffSummaryResponse{
		State: string(state),
		Diff:  convertDiffSummary(diff),
	}, nil
}

// convertDiffSummary converts a []*types.FileDiff to an api.DiffSummary (without hunks)
func convertDiffSummary(files []*types.FileDiff) *api.DiffSummary {
	if files == nil {
		return nil
	}

	apiFiles := make([]*api.FileSummary, len(files))
	for i, f := range files {
		apiFiles[i] = convertFileSummary(f)
	}

	return &api.DiffSummary{
		Files: apiFiles,
	}
}

// convertFileSummary converts a types.FileDiff to an api.FileSummary (without hunks)
func convertFileSummary(f *types.FileDiff) *api.FileSummary {
	if f == nil {
		return nil
	}

	status := api.FileStatus_FILE_STATUS_MODIFIED
	switch {
	case f.IsUntracked:
		status = api.FileStatus_FILE_STATUS_UNTRACKED
	case f.IsNew:
		status = api.FileStatus_FILE_STATUS_NEW
	case f.IsDeleted:
		status = api.FileStatus_FILE_STATUS_DELETED
	case f.IsRenamed:
		status = api.FileStatus_FILE_STATUS_RENAMED
	}

	return &api.FileSummary{
		OldPath:     f.OldPath,
		NewPath:     f.NewPath,
		FileModeOld: f.OldMode,
		FileModeNew: f.NewMode,
		Status:      status,
		IsBinary:    f.IsBinary,
	}
}
