package cmd

import (
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/tap"
	"github.com/spf13/cobra"
)

// NewUntapCmd creates the untap command
func NewUntapCmd(cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "untap [OPTIONS] TAP",
		Short: "Remove a tapped formula repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := args[0]
			tapManager := tap.NewManager(cfg)
			options := &tap.TapOptions{
				Force: force,
			}

			return tapManager.RemoveTap(tapName, options)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Untap even if formulae from this tap are installed")

	return cmd
}
