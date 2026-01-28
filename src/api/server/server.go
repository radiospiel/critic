package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/radiospiel/critic/src/api/apiconnect"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/webui"
)

// Config holds the configuration for the API server.
type Config struct {
	Port      int
	GitRoot   string
	Messaging critic.Messaging
	Args      DiffArgs
	Dev       bool // Development mode: proxy to Vite dev server instead of serving embedded files
}

// Server implements the CriticService API.
type Server struct {
	config  Config
	session *Session
	wsHub   *webui.Hub
}

// NewServer creates a new API server with the given configuration.
// It initializes a default session with the provided configuration values.
func NewServer(config Config) *Server {
	session := NewSession(config.GitRoot, config.Messaging, config.Args)
	return &Server{
		config:  config,
		session: session,
		wsHub:   webui.NewHub(),
	}
}

// GetSession returns the server's session
func (s *Server) GetSession() *Session {
	return s.session
}

// Start starts the API server and blocks until it receives an error.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	mux := http.NewServeMux()

	// Connect RPC API
	path, handler := apiconnect.NewCriticServiceHandler(s)
	mux.Handle(path, handler)

	// WebSocket
	mux.HandleFunc("GET /ws", webui.WebSocketHandler(s.wsHub))

	// Serve React app
	if s.config.Dev {
		// Development mode: proxy to Vite dev server
		viteURL, err := url.Parse("http://localhost:5173")
		if err != nil {
			return fmt.Errorf("failed to parse vite URL: %w", err)
		}
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		mux.Handle("/", proxy)
		fmt.Println("Development mode: proxying to Vite dev server at http://localhost:5173")
		fmt.Println("Run 'npm run dev' in src/webui/frontend/ to start the Vite dev server")
	} else {
		// Production mode: serve embedded files
		distFS, err := webui.DistFS()
		if err != nil {
			return fmt.Errorf("failed to get dist fs: %w", err)
		}
		mux.Handle("/", http.FileServer(distFS))
	}

	addr := fmt.Sprintf(":%d", s.config.Port)
	fmt.Printf("Critic running at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}
