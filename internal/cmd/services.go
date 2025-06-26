package cmd

import (
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewServicesCmd creates the services command
func NewServicesCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services [SUBCOMMAND]",
		Short: "Manage background services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all managed services",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Services list not yet implemented")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "start [SERVICE...]",
		Short: "Start services",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Services start not yet implemented")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop [SERVICE...]",
		Short: "Stop services",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Services stop not yet implemented")
			return nil
		},
	})

	return cmd
}
