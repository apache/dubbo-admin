package cmd

import (
	"github.com/apache/dubbo-admin/pkg/core/cmd"
	"github.com/apache/dubbo-admin/pkg/logger"

	"github.com/spf13/cobra"
)

func newAuthCmdWithOpts(opts cmd.RunCmdOpts) *cobra.Command {
	args := struct {
		configPath string
	}{}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Launch Dubbo Admin auth server.",
		Long:  `Launch Dubbo Admin auth server.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// start CA
			if err := startCA(cmd); err != nil {
				logger.Error("Failed to start auth server.")
				return err
			}
			return nil
		},
	}

	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")

	return cmd
}
