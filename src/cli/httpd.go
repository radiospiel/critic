package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

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
	var cpuProfile string

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
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				}
			}()

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

			// Load project config (optional); merges configured diff bases with defaults.
			projectFile, _ := cmd.Flags().GetString("project")
			if projectFile != "" {
				if _, err := os.Stat(projectFile); err != nil {
					return fmt.Errorf("project config file not found: %s", projectFile)
				}
			} else {
				projectFile = filepath.Join(gitRoot, "project.critic")
			}
			projectConfig, err := config.LoadProjectConfig(projectFile, git.GetCurrentBranch(), git.HasRef)
			if err != nil {
				return err
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
				DiffBases:         projectConfig.DiffBases,
				GitRoot:           gitRoot,
				Messaging:         mdb,
				ProjectConfig:     projectConfig,
				ProjectConfigPath: projectConfig.ConfigPath,
			}

			srv := server.NewServer(config)
			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 65432, "Port to run the API server on")
	cmd.Flags().BoolVar(&dev, "dev", false, "Development mode: proxy to Vite dev server for hot reload")
	cmd.Flags().StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile to file (use 'go tool pprof' to analyze)")

	return cmd
}
