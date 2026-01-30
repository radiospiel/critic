package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api/apiconnect"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/webui"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		logger.Info("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

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
	config    Config
	session   *Session
	wsHub     *webui.Hub
	devServer *webui.DevServer
}

// NewServer creates a new API server with the given configuration.
// It initializes a default session with the provided configuration values.
func NewServer(config Config) *Server {
	session := NewSession(config.GitRoot, config.Messaging, config.Args)
	must.Must(session.SetRefs("master"))
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

	// Connect RPC API with logging and validation interceptors
	interceptors := connect.WithInterceptors(loggingInterceptor(), validatorInterceptor())
	path, handler := apiconnect.NewCriticServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	// WebSocket
	mux.HandleFunc("GET /ws", webui.WebSocketHandler(s.wsHub))

	// REST API endpoints (temporary, until proto is regenerated)
	mux.HandleFunc("GET /api/comments", s.GetCommentsHandler())

	// Serve React app
	if s.config.Dev {
		// Try to start Vite dev server
		s.devServer = webui.NewDevServer("src/webui/frontend", 5173)
		if s.devServer.Start() {
			mux.Handle("/", s.devServer.Handler())
			fmt.Println("Development mode: proxying to Vite dev server at http://localhost:5173")
		}
	}

	if s.devServer == nil || !s.devServer.Started() {
		// Production mode or dev fallback: serve embedded files
		distFS, err := webui.DistFS()
		if err != nil {
			return fmt.Errorf("failed to get dist fs: %w", err)
		}
		mux.Handle("/", http.FileServer(distFS))
	}

	// Set up HTTP server with graceful shutdown
	addr := fmt.Sprintf(":%d", s.config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: loggingMiddleware(mux),
	}

	// Channel to listen for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("Critic running at http://localhost%s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-stop:
		logger.Info("Shutdown signal received")
	case err := <-serverErr:
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error: %v", err)
	}

	// Stop dev server if running
	if s.devServer != nil {
		s.devServer.Stop()
	}

	return nil
}
