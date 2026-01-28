package api

import "fmt"

// InspectForLog implements logger.Inspect for GetLastChangeRequest
func (x *GetLastChangeRequest) InspectForLog() string {
	return "GetLastChangeRequest{}"
}

// InspectForLog implements logger.Inspect for GetLastChangeResponse
func (x *GetLastChangeResponse) InspectForLog() string {
	if x == nil {
		return "GetLastChangeResponse{nil}"
	}
	return fmt.Sprintf("GetLastChangeResponse{mtime_msecs=%d}", x.MtimeMsecs)
}

// InspectForLog implements logger.Inspect for GetDiffsRequest
func (x *GetDiffsRequest) InspectForLog() string {
	return "GetDiffsRequest{}"
}

// InspectForLog implements logger.Inspect for GetDiffsResponse
func (x *GetDiffsResponse) InspectForLog() string {
	if x == nil {
		return "GetDiffsResponse{nil}"
	}
	fileCount := 0
	if x.Diff != nil {
		fileCount = len(x.Diff.Files)
	}
	return fmt.Sprintf("GetDiffsResponse{state=%q, files=%d}", x.State, fileCount)
}

// InspectForLog implements logger.Inspect for Diff
func (x *Diff) InspectForLog() string {
	if x == nil {
		return "Diff{nil}"
	}
	return fmt.Sprintf("Diff{files=%d}", len(x.Files))
}

// InspectForLog implements logger.Inspect for FileDiff
func (x *FileDiff) InspectForLog() string {
	if x == nil {
		return "FileDiff{nil}"
	}
	path := x.NewPath
	if path == "" {
		path = x.OldPath
	}
	return fmt.Sprintf("FileDiff{path=%q, status=%s}", path, x.Status.String())
}
