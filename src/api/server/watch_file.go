package server

import (
	"context"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/git"
)

// WatchFile sets up a file watcher for the specified file path.
// When the file changes, a "file-changed" message is broadcast via WebSocket.
func (s *Server) WatchFile(
	ctx context.Context,
	req *connect.Request[api.WatchFileRequest],
) (*connect.Response[api.WatchFileResponse], error) {
	response := depanic(func() (*api.WatchFileResponse, error) {
		return watchFileImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func watchFileImpl(server *Server, req *api.WatchFileRequest) (*api.WatchFileResponse, error) {
	path := req.GetPath()

	// Stop existing file watcher if any
	server.session.StopFileWatcher()

	// If path is empty, just stop watching
	if path == "" {
		logger.Info("WatchFile: Stopped watching files")
		return &api.WatchFileResponse{}, nil
	}

	// Convert to absolute path
	absPath := filepath.Join(server.config.GitRoot, path)

	// Create a new file watcher
	watcher, err := git.NewFileWatcher(absPath, 100) // 100ms debounce
	if err != nil {
		logger.Error("WatchFile: Failed to create watcher for %s: %v", path, err)
		return &api.WatchFileResponse{}, err
	}

	server.session.SetFileWatcher(watcher)

	// Start listening for changes
	go handleFileChanges(server, watcher)

	logger.Info("WatchFile: Now watching %s", path)
	return &api.WatchFileResponse{}, nil
}

// handleFileChanges listens for file changes and broadcasts file-changed messages.
func handleFileChanges(server *Server, watcher *git.FileWatcher) {
	path := watcher.Path()

	for range watcher.Changes() {
		// Check if this is still the active watcher
		currentWatcher := server.session.GetFileWatcher()
		if currentWatcher != watcher {
			return
		}

		logger.Info("File change detected: %s, broadcasting file-changed", path)

		// Broadcast file-changed message to all connected clients
		msg := fmt.Sprintf(`{"type":"file-changed","path":"%s"}`, path)
		server.wsHub.Broadcast([]byte(msg))
	}
}
