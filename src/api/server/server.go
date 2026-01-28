package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/fsnotify/fsnotify"
	"github.com/radiospiel/critic/simple-go/logger"
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
	config     Config
	session    *Session
	wsHub      *webui.Hub
	npmProcess *exec.Cmd
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

// startNpmDevServer starts the Vite dev server in the frontend directory.
// Returns true if the dev server was started successfully, false if it's not available.
func (s *Server) startNpmDevServer() bool {
	// Find the frontend directory relative to the working directory
	frontendDir := "src/webui/frontend"

	// Check if node_modules exists
	nodeModulesPath := frontendDir + "/node_modules"
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Println("Warning: Dev server not available (node_modules not found)")
		fmt.Println("To enable dev mode with hot reload, run:")
		fmt.Printf("  cd %s && npm install\n", frontendDir)
		fmt.Println("Falling back to serving embedded files.")
		return false
	}

	s.npmProcess = exec.Command("npm", "run", "dev")
	s.npmProcess.Dir = frontendDir
	s.npmProcess.Stdout = os.Stdout
	s.npmProcess.Stderr = os.Stderr

	if err := s.npmProcess.Start(); err != nil {
		fmt.Printf("Warning: Failed to start dev server: %v\n", err)
		fmt.Println("Falling back to serving embedded files.")
		return false
	}

	logger.Info("Started Vite dev server (PID %d)", s.npmProcess.Process.Pid)
	return true
}

// stopNpmDevServer stops the Vite dev server if running
func (s *Server) stopNpmDevServer() {
	if s.npmProcess != nil && s.npmProcess.Process != nil {
		logger.Info("Stopping Vite dev server (PID %d)", s.npmProcess.Process.Pid)
		// Send SIGTERM for graceful shutdown
		if err := s.npmProcess.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Error("Failed to send SIGTERM to npm process: %v", err)
			// Try SIGKILL as fallback
			s.npmProcess.Process.Kill()
		}
		s.npmProcess.Wait()
		logger.Info("Vite dev server stopped")
	}
}

// startDevFileWatcher watches src/webui for Go file changes and broadcasts reload events
func (s *Server) startDevFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Failed to create file watcher: %v", err)
		return
	}

	// Watch src/webui directory
	webuiDir := "src/webui"
	if err := watcher.Add(webuiDir); err != nil {
		logger.Error("Failed to watch %s: %v", webuiDir, err)
		watcher.Close()
		return
	}

	// Also watch subdirectories (but not frontend/node_modules)
	filepath.Walk(webuiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip frontend directory (Vite handles that)
			if strings.Contains(path, "frontend") {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})

	logger.Info("Watching %s for changes (will trigger browser reload)", webuiDir)

	go func() {
		defer watcher.Close()
		var debounceTimer *time.Timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Only react to Go files
				if !strings.HasSuffix(event.Name, ".go") {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					// Debounce: wait 500ms before broadcasting reload
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
						logger.Info("Backend file changed: %s, broadcasting reload", event.Name)
						s.wsHub.Broadcast([]byte(`{"type":"reload"}`))
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("File watcher error: %v", err)
			}
		}
	}()
}

// Start starts the API server and blocks until it receives an error.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	mux := http.NewServeMux()

	// Connect RPC API with logging interceptor
	interceptors := connect.WithInterceptors(loggingInterceptor())
	path, handler := apiconnect.NewCriticServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	// WebSocket
	mux.HandleFunc("GET /ws", webui.WebSocketHandler(s.wsHub))

	// Serve React app
	devServerStarted := false
	if s.config.Dev {
		// Try to start npm dev server
		devServerStarted = s.startNpmDevServer()
		if devServerStarted {
			// Start file watcher for backend code changes
			s.startDevFileWatcher()

			// Give Vite a moment to start
			time.Sleep(2 * time.Second)

			// Development mode: proxy to Vite dev server
			viteURL, err := url.Parse("http://localhost:5173")
			if err != nil {
				s.stopNpmDevServer()
				return fmt.Errorf("failed to parse vite URL: %w", err)
			}
			proxy := httputil.NewSingleHostReverseProxy(viteURL)
			mux.Handle("/", proxy)
			fmt.Println("Development mode: proxying to Vite dev server at http://localhost:5173")
		}
	}

	if !devServerStarted {
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

	// Stop npm dev server if running
	s.stopNpmDevServer()

	return nil
}
