package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

// GetDiffs returns the current diff summary (file list without hunks) and state.
// Note: This implements GetDiffSummary behavior - hunks are not included.
// The proto definition has been updated to use GetDiffSummary naming,
// but until protos are regenerated, this maintains backward compatibility.
func (s *Server) GetDiffs(
	ctx context.Context,
	req *connect.Request[api.GetDiffsRequest],
) (*connect.Response[api.GetDiffsResponse], error) {
	session := s.GetSession()
	state := session.GetState()
	diff := session.GetDiffSummary()

	res := connect.NewResponse(&api.GetDiffsResponse{
		State: string(state),
		Diff:  convertDiffWithoutHunks(diff),
	})
	return res, nil
}

// convertDiffWithoutHunks converts a types.Diff to an api.Diff without including hunks
func convertDiffWithoutHunks(d *types.Diff) *api.Diff {
	if d == nil {
		return nil
	}

	files := make([]*api.FileDiff, len(d.Files))
	for i, f := range d.Files {
		files[i] = convertFileDiffSummary(f)
	}

	return &api.Diff{
		Files: files,
	}
}

// convertFileDiffSummary converts a types.FileDiff to an api.FileDiff without hunks
func convertFileDiffSummary(f *types.FileDiff) *api.FileDiff {
	if f == nil {
		return nil
	}

	status := api.FileStatus_FILE_STATUS_MODIFIED
	switch {
	case f.IsNew:
		status = api.FileStatus_FILE_STATUS_NEW
	case f.IsDeleted:
		status = api.FileStatus_FILE_STATUS_DELETED
	case f.IsRenamed:
		status = api.FileStatus_FILE_STATUS_RENAMED
	}

	// Note: Hunks are intentionally not included in summary responses
	return &api.FileDiff{
		OldPath:     f.OldPath,
		NewPath:     f.NewPath,
		FileModeOld: f.OldMode,
		FileModeNew: f.NewMode,
		Status:      status,
		IsBinary:    f.IsBinary,
		Hunks:       nil, // Summary doesn't include hunks
	}
}
