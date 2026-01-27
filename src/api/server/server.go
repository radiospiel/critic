package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/api/apiconnect"
)

// Config holds the configuration for the API server.
type Config struct {
	Port int
}

// Server implements the CriticService API.
type Server struct {
	config Config
}

// NewServer creates a new API server with the given configuration.
func NewServer(config Config) *Server {
	return &Server{
		config: config,
	}
}

// GetLastChange returns the current time in milliseconds.
func (s *Server) GetLastChange(
	ctx context.Context,
	req *connect.Request[api.GetLastChangeRequest],
) (*connect.Response[api.GetLastChangeResponse], error) {
	now := time.Now().UnixMilli()
	res := connect.NewResponse(&api.GetLastChangeResponse{
		MtimeMsecs: uint64(now),
	})
	return res, nil
}

// Start starts the API server and blocks until it receives an error.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register the CriticService handler under /api prefix
	path, handler := apiconnect.NewCriticServiceHandler(s)
	mux.Handle("/api"+path, http.StripPrefix("/api", handler))

	// Also register at root path for grpcurl compatibility
	mux.Handle(path, handler)

	addr := fmt.Sprintf(":%d", s.config.Port)
	fmt.Printf("API server listening on %s\n", addr)
	fmt.Printf("\nTest with grpcurl:\n")
	fmt.Printf("  grpcurl -plaintext -import-path src/api/proto -proto critic.proto localhost:%d critic.v1.CriticService/GetLastChange\n", s.config.Port)
	fmt.Printf("\nTest with curl:\n")
	fmt.Printf("  curl -X POST http://localhost:%d/api/critic.v1.CriticService/GetLastChange -H 'Content-Type: application/json' -d '{}'\n", s.config.Port)
	return http.ListenAndServe(addr, mux)
}
