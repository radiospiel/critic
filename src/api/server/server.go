package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/api/apiconnect"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/webui"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

// Hijack implements http.Hijacker for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
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
	Dev       bool // Development mode: proxy to Vite dev server instead of serving embedded files
	DiffBases []string
	Paths     []string
}

// Server implements the CriticService API.
type Server struct {
	config         Config
	session        *Session
	wsHub          *webui.Hub
	devServer      *webui.DevServer
	gitWatcher     *git.GitWatcher
	lastChangeTime atomic.Int64 // Unix milliseconds of last detected change
}

// NewServer creates a new API server with the given configuration.
// It initializes a default session with the provided configuration values.
func NewServer(config Config) *Server {
	session := NewSession(config.GitRoot, config.Messaging, config.Paths, config.DiffBases)
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

// SetLastChangeTime sets the timestamp of the last detected change.
func (s *Server) SetLastChangeTime(t int64) {
	s.lastChangeTime.Store(t)
}

// LastChangeTime returns the timestamp of the last detected change.
// Returns 0 if no change has been recorded.
func (s *Server) LastChangeTime() int64 {
	return s.lastChangeTime.Load()
}

// handleGitChanges listens for git directory changes and broadcasts reload messages.
func (s *Server) handleGitChanges() {
	if s.gitWatcher == nil {
		return
	}

	for range s.gitWatcher.Changes() {
		// Update last change time
		s.SetLastChangeTime(s.gitWatcher.LastChangeTime())
		logger.Info("Git change detected, broadcasting reload")

		// Broadcast reload message to all connected clients
		s.wsHub.Broadcast([]byte(`{"type":"reload"}`))
	}
}

// Start starts the API server and blocks until it receives an error.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Start git directory watcher
	if s.config.GitRoot != "" {
		gitDir := s.config.GitRoot + "/.git"
		watcher, err := git.NewGitWatcher(gitDir, 100) // 100ms debounce
		if err != nil {
			logger.Error("Failed to start git watcher: %v", err)
		} else {
			s.gitWatcher = watcher
			// Set initial last change time
			s.SetLastChangeTime(watcher.LastChangeTime())
			// Listen for changes
			go s.handleGitChanges()
		}
	}

	mux := http.NewServeMux()

	// Connect RPC API with logging and validation interceptors
	interceptors := connect.WithInterceptors(loggingInterceptor(), validatorInterceptor())
	path, handler := apiconnect.NewCriticServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	// WebSocket
	mux.HandleFunc("GET /ws", webui.WebSocketHandler(s.wsHub))

	// In dev mode, we need to start the HTTP server first so Vite's proxy can connect.
	// We'll add the frontend handler after the server is listening.
	var frontendHandler http.Handler

	if s.config.Dev {
		// Defer Vite startup until HTTP server is ready
		s.devServer = webui.NewDevServer("src/webui/frontend", 5173)
	}

	if s.devServer == nil {
		// Production mode: serve embedded files
		distFS, err := webui.DistFS()
		if err != nil {
			return fmt.Errorf("failed to get dist fs: %w", err)
		}
		frontendHandler = http.FileServer(distFS)
	}

	// Use a dynamic handler that can be updated after Vite starts
	var dynamicFrontendHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if frontendHandler != nil {
			frontendHandler.ServeHTTP(w, r)
		} else {
			// Vite not ready yet, return a simple "loading" response
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`<!DOCTYPE html><html><head><meta http-equiv="refresh" content="1"></head><body>Starting dev server...</body></html>`))
		}
	})
	mux.Handle("/", dynamicFrontendHandler)

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

	// Start Vite dev server after HTTP server is listening (in dev mode)
	if s.devServer != nil {
		// Give HTTP server a moment to start listening
		time.Sleep(100 * time.Millisecond)
		if !s.devServer.Start() {
			return fmt.Errorf("failed to start Vite dev server")
		}
		frontendHandler = s.devServer.Handler()
		fmt.Println("Development mode: proxying to Vite dev server at http://localhost:5173")
	}

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

	// Stop git watcher if running
	if s.gitWatcher != nil {
		s.gitWatcher.Close()
	}

	return nil
}

func rpcErrorFromGoError(reason any) *api.RpcError {
	if reason == nil {
		return nil
	}

	// Check if it's already an RpcErr (typed error from api package)
	if rpcErr, ok := reason.(*api.RpcErr); ok {
		return rpcErr.RpcError()
	}

	// Convert to error if not already
	err, ok := reason.(error)
	if !ok {
		err = fmt.Errorf("panic: %v", reason)
	}

	// Check if the error wraps an RpcErr
	var rpcErr *api.RpcErr
	if errors.As(err, &rpcErr) {
		return rpcErr.RpcError()
	}

	// Default: internal server error
	return &api.RpcError{
		Code:    api.ErrorCode_ERROR_CODE_INTERNAL,
		Message: "internal server error",
		Details: err.Error(),
	}
}

// setResponseError sets the "error" field on a proto message using reflection.
func setResponseError(msg proto.Message, rpcErr *api.RpcError) {
	if rpcErr == nil {
		return
	}
	reflect := msg.ProtoReflect()
	errorField := reflect.Descriptor().Fields().ByName("error")
	if errorField != nil {
		reflect.Set(errorField, protoreflect.ValueOfMessage(rpcErr.ProtoReflect()))
	}
}

// depanic wraps a function that returns a proto response and error,
// catching panics and errors and setting them on the response's Error field.
//
// The trick is the two-type-parameter constraint:
//   - T is the underlying struct type (e.g., api.CreateConversationResponse)
//   - PT is constrained to be both *T and proto.Message
//
// This lets us use PT(new(T)) to create a new pointer instance without needing
// a constructor function. Go infers both type parameters from the function signature.
func depanic[T any, PT interface {
	*T
	proto.Message
}](fun func() (PT, error)) PT {
	var result PT
	defer func() {
		if recovered := recover(); recovered != nil {
			result = PT(new(T))
			setResponseError(result, rpcErrorFromGoError(recovered))
		}
	}()

	var err error
	result, err = fun()
	if err != nil {
		result = PT(new(T))
		setResponseError(result, rpcErrorFromGoError(err))
	}
	return result
}
