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

// InspectForLog implements logger.Inspect for GetDiffSummaryRequest
func (x *GetDiffSummaryRequest) InspectForLog() string {
	return "GetDiffSummaryRequest{}"
}

// InspectForLog implements logger.Inspect for GetDiffSummaryResponse
func (x *GetDiffSummaryResponse) InspectForLog() string {
	if x == nil {
		return "GetDiffSummaryResponse{nil}"
	}
	fileCount := 0
	if x.Diff != nil {
		fileCount = len(x.Diff.Files)
	}
	return fmt.Sprintf("GetDiffSummaryResponse{state=%q, files=%d}", x.State, fileCount)
}

// InspectForLog implements logger.Inspect for GetDiffRequest
func (x *GetDiffRequest) InspectForLog() string {
	if x == nil {
		return "GetDiffRequest{nil}"
	}
	return fmt.Sprintf("GetDiffRequest{path=%q}", x.Path)
}

// InspectForLog implements logger.Inspect for GetDiffResponse
func (x *GetDiffResponse) InspectForLog() string {
	if x == nil {
		return "GetDiffResponse{nil}"
	}
	if x.File == nil {
		return "GetDiffResponse{file=nil}"
	}
	return fmt.Sprintf("GetDiffResponse{file=%s}", x.File.InspectForLog())
}

// InspectForLog implements logger.Inspect for DiffSummary
func (x *DiffSummary) InspectForLog() string {
	if x == nil {
		return "DiffSummary{nil}"
	}
	return fmt.Sprintf("DiffSummary{files=%d}", len(x.Files))
}

// InspectForLog implements logger.Inspect for FileSummary
func (x *FileSummary) InspectForLog() string {
	if x == nil {
		return "FileSummary{nil}"
	}
	path := x.NewPath
	if path == "" {
		path = x.OldPath
	}
	return fmt.Sprintf("FileSummary{path=%q, status=%s}", path, x.Status.String())
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
	return fmt.Sprintf("FileDiff{path=%q, status=%s, hunks=%d}", path, x.Status.String(), len(x.Hunks))
}
