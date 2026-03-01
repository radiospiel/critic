package cli

import (
	"fmt"

	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/spf13/cobra"
)

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage critic settings",
	}

	cmd.AddCommand(newListSettingsCmd())
	cmd.AddCommand(newGetSettingCmd())
	cmd.AddCommand(newSetSettingCmd())

	return cmd
}

func newListSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all setting names",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := messagedb.New(git.GetGitRoot())
			if err != nil {
				return err
			}
			defer db.Close()

			settings, err := db.ListSettings()
			if err != nil {
				return err
			}
			for _, s := range settings {
				fmt.Printf("%s\t%s\n", s.Key, s.Value)
			}
			return nil
		},
	}
}

func newGetSettingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a setting value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := messagedb.New(git.GetGitRoot())
			if err != nil {
				return err
			}
			defer db.Close()

			value, err := db.GetSetting(args[0])
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		},
	}
}

func newSetSettingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a setting value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := messagedb.New(git.GetGitRoot())
			if err != nil {
				return err
			}
			defer db.Close()

			return db.SetSetting(args[0], args[1])
		},
	}
}
