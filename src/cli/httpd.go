package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api/server"
	"github.com/radiospiel/critic/src/config"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/spf13/cobra"
)

// newHTTPDCmd creates the httpd subcommand
func newHTTPDCmd() *cobra.Command {
	var port int
	var dev bool
	var diffBases []string
	var cpuProfile string
	var projectFile string

	cmd := &cobra.Command{
		Use:   "httpd [flags]",
		Short: "Start the HTTP server",
		Long: `Start the HTTP server.

The HTTP server provides a programmatic interface for interacting with critic.
It uses Connect-RPC. Read more at https://connectrpc.com/

The HTTP server also provides a react-based frontend.

Examples:
  critic httpd                    # Start on default port 65432
  critic httpd --port=8000        # Start on custom port
  critic httpd --dev              # Development mode with Vite hot reload
  critic httpd --cpuprofile=cpu.prof  # Enable CPU profiling
  critic httpd --project=my.critic   # Use custom project config file
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				}
			}()

			// Always include default bases (master, main, HEAD) plus any explicitly added.
			// Deferred to avoid git calls during help.
			diffBases = mergeDefaultBases(diffBases)

			// Start CPU profiling if requested
			if cpuProfile != "" {
				f, err := os.Create(cpuProfile)
				if err != nil {
					return fmt.Errorf("could not create CPU profile: %w", err)
				}
				defer f.Close()
				if err := pprof.StartCPUProfile(f); err != nil {
					return fmt.Errorf("could not start CPU profile: %w", err)
				}
				defer pprof.StopCPUProfile()
				fmt.Fprintf(cmd.ErrOrStderr(), "CPU profiling enabled, writing to %s\n", cpuProfile)
			}

			// Get git root directory
			gitRoot := git.GetGitRoot()

			// Load project config (optional)
			var projectConfigPath string
			if projectFile != "" {
				projectConfigPath = projectFile
			} else {
				projectConfigPath = filepath.Join(gitRoot, "project.critic")
			}
			projectConfig, err := config.LoadProjectConfigFromFile(projectConfigPath)
			if err != nil {
				if projectFile != "" {
					return fmt.Errorf("failed to load project config from %s: %w", projectFile, err)
				}
				logger.Error("failed to load project.critic: %v", err)
				projectConfig = config.DefaultProjectConfig()
			}

			// Initialize the message database
			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			config := server.Config{
				Port:              port,
				Dev:               dev,
				DiffBases:         diffBases,
				GitRoot:           gitRoot,
				Messaging:         mdb,
				ProjectConfig:     projectConfig,
				ProjectConfigPath: projectConfigPath,
			}

			srv := server.NewServer(config)
			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 65432, "Port to run the API server on")
	cmd.Flags().BoolVar(&dev, "dev", false, "Development mode: proxy to Vite dev server for hot reload")
	cmd.Flags().StringSliceVar(&diffBases, "base-commits", nil, "Diff base commits (defaults to main/master/origin/<branch>/HEAD)")
	cmd.Flags().Lookup("base-commits").Shorthand = "b"
	cmd.Flags().StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile to file (use 'go tool pprof' to analyze)")
	cmd.Flags().StringVar(&projectFile, "project", "", "Path to project.critic config file (default: auto-detect in git root)")

	return cmd
}
