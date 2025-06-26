package cmd

import (
	"fmt"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/tap"
	"github.com/spf13/cobra"
)

// NewTapCmd creates the tap command
func NewTapCmd(cfg *config.Config) *cobra.Command {
	var (
		force   bool
		shallow bool
		quiet   bool
		branch  string
	)

	cmd := &cobra.Command{
		Use:   "tap [OPTIONS] [USER/REPO] [URL]",
		Short: "Tap a formula repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// List taps
				return listTaps(cfg)
			}

			tapName := args[0]
			remote := ""
			if len(args) > 1 {
				remote = args[1]
			}

			tapManager := tap.NewManager(cfg)
			options := &tap.TapOptions{
				Force:   force,
				Quiet:   quiet,
				Shallow: shallow,
				Branch:  branch,
			}

			return tapManager.AddTap(tapName, remote, options)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force tap even if already tapped")
	cmd.Flags().BoolVar(&shallow, "shallow", false, "Perform a shallow clone")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress output")
	cmd.Flags().StringVar(&branch, "branch", "", "Clone specific branch")

	return cmd
}

func listTaps(cfg *config.Config) error {
	tapManager := tap.NewManager(cfg)
	taps, err := tapManager.ListTaps()
	if err != nil {
		return fmt.Errorf("failed to list taps: %w", err)
	}

	for _, t := range taps {
		fmt.Println(t.Name)
	}

	return nil
}
