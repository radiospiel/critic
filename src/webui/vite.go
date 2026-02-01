package webui

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

// DevServer manages the Vite development server
type DevServer struct {
	frontendDir string
	vitePort    int
	npmProcess  *exec.Cmd
	started     bool
}

// NewDevServer creates a new DevServer instance
func NewDevServer(frontendDir string, vitePort int) *DevServer {
	return &DevServer{
		frontendDir: frontendDir,
		vitePort:    vitePort,
	}
}

// Start starts the Vite dev server if dependencies are available.
// Returns true if the dev server was started successfully, false if it's not available.
func (d *DevServer) Start() bool {
	// Check if node_modules exists
	nodeModulesPath := d.frontendDir + "/node_modules"
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Println("Warning: Dev server not available (node_modules not found)")
		fmt.Println("To enable dev mode with hot reload, run:")
		fmt.Printf("  cd %s && npm install\n", d.frontendDir)
		fmt.Println("Falling back to serving embedded files.")
		return false
	}

	d.npmProcess = exec.Command("npm", "run", "dev")
	d.npmProcess.Dir = d.frontendDir
	d.npmProcess.Stdout = os.Stdout
	d.npmProcess.Stderr = os.Stderr

	if err := d.npmProcess.Start(); err != nil {
		fmt.Printf("Warning: Failed to start dev server: %v\n", err)
		fmt.Println("Falling back to serving embedded files.")
		return false
	}

	logger.Info("Started Vite dev server (PID %d)", d.npmProcess.Process.Pid)
	d.started = true

	// Wait for Vite to be ready by checking if port 5173 is accepting connections
	viteAddr := fmt.Sprintf("localhost:%d", d.vitePort)
	for i := 0; i < 50; i++ { // Try for up to 5 seconds
		conn, err := net.DialTimeout("tcp", viteAddr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			logger.Info("Vite dev server is ready")
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	logger.Warn("Vite dev server may not be fully ready")
	return true
}

// Stop stops the Vite dev server if running
func (d *DevServer) Stop() {
	if d.npmProcess != nil && d.npmProcess.Process != nil {
		logger.Info("Stopping Vite dev server (PID %d)", d.npmProcess.Process.Pid)
		// Send SIGTERM for graceful shutdown
		if err := d.npmProcess.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Error("Failed to send SIGTERM to npm process: %v", err)
			// Try SIGKILL as fallback
			d.npmProcess.Process.Kill()
		}
		d.npmProcess.Wait()
		logger.Info("Vite dev server stopped")
	}
}

// Started returns true if the dev server was started successfully
func (d *DevServer) Started() bool {
	return d.started
}

// Handler returns an HTTP handler that proxies to the Vite dev server.
// This handles both HTTP and WebSocket requests (for HMR).
func (d *DevServer) Handler() http.Handler {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", d.vitePort))
	httpProxy := httputil.NewSingleHostReverseProxy(target)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebSocketRequest(r) {
			proxyWebSocket(w, r, target)
		} else {
			httpProxy.ServeHTTP(w, r)
		}
	})
}

// isWebSocketRequest checks if the request is a WebSocket upgrade request
func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// proxyWebSocket proxies a WebSocket connection to the target URL
func proxyWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	// Connect to the target WebSocket server
	targetHost := target.Host
	conn, err := net.Dial("tcp", targetHost)
	if err != nil {
		http.Error(w, "Failed to connect to upstream", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Forward the original request to the target
	r.URL.Scheme = "ws"
	r.URL.Host = targetHost
	r.Host = targetHost
	if err := r.Write(conn); err != nil {
		return
	}

	// Bidirectional copy
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, conn)
		done <- struct{}{}
	}()
	<-done
}
