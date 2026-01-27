package server

import (
	"fmt"
	"net/http"

	"github.com/radiospiel/critic/src/api/apiconnect"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// Config holds the configuration for the API server.
type Config struct {
	Port      int
	GitRoot   string
	Messaging critic.Messaging
	Args      DiffArgs
}

// Server implements the CriticService API.
type Server struct {
	config  Config
	session *Session
}

// NewServer creates a new API server with the given configuration.
// It initializes a default session with the provided configuration values.
func NewServer(config Config) *Server {
	session := NewSession(config.GitRoot, config.Messaging, config.Args)
	return &Server{
		config:  config,
		session: session,
	}
}

// GetSession returns the server's session
func (s *Server) GetSession() *Session {
	return s.session
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
