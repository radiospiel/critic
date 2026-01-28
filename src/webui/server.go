package webui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/api/apiconnect"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

//go:embed dist/*
var embeddedFS embed.FS

// Config holds the configuration for the web server
type Config struct {
	Port  int
	Bases []string
	Paths []string
}

// Server represents the web UI server
type Server struct {
	config     Config
	messaging  critic.Messaging
	hub        *Hub // WebSocket hub
	diff       *types.Diff
	diffMu     sync.RWMutex
	lastChange time.Time
}

// NewServer creates a new web UI server
func NewServer(config Config) (*Server, error) {
	// Initialize message database
	gitRoot, err := git.GetGitRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}

	// Get bases - use default bases if none specified
	preconditions.Check(len(config.Bases) > 0, "bases must not be empty")

	mdb, err := messagedb.New(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message database: %w", err)
	}

	server := &Server{
		config:    config,
		messaging: mdb,
		hub:       NewHub(),
	}

	return server, nil
}

// Start starts the web server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.hub.Run()

	// Load initial diff
	if err := s.loadDiff(); err != nil {
		logger.Warn("Failed to load initial diff: %v", err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Connect RPC API (proto-based)
	path, handler := apiconnect.NewCriticServiceHandler(s)
	mux.Handle(path, handler)

	// WebSocket
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	// Serve React app (static files from dist/)
	distFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create dist fs: %w", err)
	}
	staticHandler := http.FileServer(http.FS(distFS))
	mux.Handle("/", staticHandler)

	addr := fmt.Sprintf(":%d", s.config.Port)
	logger.Info("Starting web UI server on http://localhost%s", addr)
	fmt.Printf("Critic Web UI running at http://localhost%s\n", addr)

	return http.ListenAndServe(addr, mux)
}

// loadDiff loads the diff based on current configuration
func (s *Server) loadDiff() error {
	s.diffMu.Lock()
	defer s.diffMu.Unlock()

	// Get HEAD commit for comparison
	headCommit, err := git.ResolveRef("HEAD")
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	// Find a base that produces a non-empty diff
	// First, try origin/master or origin/main as good defaults
	candidateBases := []string{}
	if len(s.config.Bases) == 0 {
		// Add origin/master and origin/main as preferred bases
		if sha, err := git.ResolveRef("origin/master"); err == nil && sha != headCommit {
			candidateBases = append(candidateBases, "origin/master")
		}
		if sha, err := git.ResolveRef("origin/main"); err == nil && sha != headCommit {
			candidateBases = append(candidateBases, "origin/main")
		}
	}
	// Then add the configured bases
	candidateBases = append(candidateBases, s.config.Bases...)

	// Find a base that is different from HEAD
	baseName := ""
	for _, b := range candidateBases {
		if b == "HEAD" {
			continue
		}
		resolved, err := resolveBase(b)
		if err != nil {
			logger.Warn("Failed to resolve base %s: %v", b, err)
			continue
		}
		if resolved != headCommit {
			baseName = b
			logger.Info("Using base %s (resolved to %s)", b, resolved)
			break
		}
	}

	if baseName == "" {
		logger.Warn("No suitable base found, diff will be empty")
		baseName = "HEAD"
	}

	baseCommit, err := resolveBase(baseName)
	if err != nil {
		return fmt.Errorf("failed to resolve base %s: %w", baseName, err)
	}

	targetCommit, err := git.ResolveRef("HEAD")
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	paths := s.config.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	diff, err := git.GetDiffBetween(baseCommit, targetCommit, paths)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	s.diff = diff
	return nil
}

// resolveBase resolves a base ref to a commit SHA
func resolveBase(base string) (string, error) {
	if git.IsCommitSHA(base) {
		return git.ResolveRef(base)
	}

	baseSHA, err := git.ResolveRef(base)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %s: %w", base, err)
	}

	mergeBase, err := git.GetMergeBaseBetween("HEAD", baseSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get merge base with %s: %w", base, err)
	}

	return mergeBase, nil
}

// getDiff returns the current diff (thread-safe)
func (s *Server) getDiff() *types.Diff {
	s.diffMu.RLock()
	defer s.diffMu.RUnlock()
	return s.diff
}

// Broadcast sends a message to all connected WebSocket clients
func (s *Server) Broadcast(message []byte) {
	s.hub.Broadcast(message)
}
