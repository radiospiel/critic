package webui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/messagedb"
	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/logger"
)

//go:embed templates/*.html static/*
var embeddedFS embed.FS

// Config holds the configuration for the web server
type Config struct {
	Port  int
	Bases []string
	Paths []string
}

// Server represents the web UI server
type Server struct {
	config    Config
	templates *template.Template
	messaging critic.Messaging
	hub       *Hub // WebSocket hub
	diff      *types.Diff
	diffMu    sync.RWMutex
}

// NewServer creates a new web UI server
func NewServer(config Config) (*Server, error) {
	// Initialize message database
	gitRoot, err := git.GetGitRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}

	mdb, err := messagedb.New(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message database: %w", err)
	}

	// Parse templates
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(embeddedFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	server := &Server{
		config:    config,
		templates: tmpl,
		messaging: mdb,
		hub:       NewHub(),
	}

	return server, nil
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
	}
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

	// Static files
	staticFS, err := fs.Sub(embeddedFS, "static")
	if err != nil {
		return fmt.Errorf("failed to create static fs: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Pages
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /file/{path...}", s.handleFile)

	// API endpoints (for htmx)
	mux.HandleFunc("GET /api/files", s.handleFileList)
	mux.HandleFunc("GET /api/diff/{path...}", s.handleDiff)
	mux.HandleFunc("GET /api/conversations/{path...}", s.handleConversations)
	mux.HandleFunc("POST /api/comment", s.handleCreateComment)
	mux.HandleFunc("POST /api/reply", s.handleReply)
	mux.HandleFunc("POST /api/resolve/{uuid}", s.handleResolve)
	mux.HandleFunc("POST /api/unresolve/{uuid}", s.handleUnresolve)

	// WebSocket
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", s.config.Port)
	logger.Info("Starting web UI server on http://localhost%s", addr)
	fmt.Printf("Critic Web UI running at http://localhost%s\n", addr)

	return http.ListenAndServe(addr, mux)
}

// loadDiff loads the diff based on current configuration
func (s *Server) loadDiff() error {
	s.diffMu.Lock()
	defer s.diffMu.Unlock()

	// Resolve base to commit SHA
	baseName := "HEAD"
	if len(s.config.Bases) > 0 {
		baseName = s.config.Bases[0]
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
