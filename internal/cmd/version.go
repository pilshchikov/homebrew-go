package cmd

import (
	"fmt"
	"runtime"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd(cfg *config.Config, version, gitCommit, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Homebrew %s\n", version); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Homebrew/brew (git revision %s; last commit %s)\n", gitCommit, buildDate); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Go: %s\n", runtime.Version()); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s\n", runtime.GOOS+"/"+runtime.GOARCH); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
