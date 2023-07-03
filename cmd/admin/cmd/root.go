package cmd

import (
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/apache/dubbo-admin/pkg/version"

	corecmd "github.com/apache/dubbo-admin/pkg/core/cmd"

	"github.com/spf13/cobra"
	"os"
)

func GetRootCmd(args []string) *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	cmd := &cobra.Command{
		Use:   "dubbo-admin",
		Short: "Console and control plane for microservices built with Apache Dubbo.",
		Long:  `Console and control plane for microservices built with Apache Dubbo.`,

		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			logger.Init()

			// once command line flags have been parsed,
			// avoid printing usage instructions
			cmd.SilenceUsage = true

			return nil
		},
	}

	cmd.SetOut(os.Stdout)

	// root flags
	//cmd.PersistentFlags().StringVar(&args.logLevel, "log-level", kuma_log.InfoLevel.String(), kuma_cmd.UsageOptions("log level", kuma_log.OffLevel, kuma_log.InfoLevel, kuma_log.DebugLevel))
	//cmd.PersistentFlags().StringVar(&args.outputPath, "log-output-path", args.outputPath, "path to the file that will be filled with logs. Example: if we set it to /tmp/admin.log then after the file is rotated we will have /tmp/admin-2021-06-07T09-15-18.265.log")
	//cmd.PersistentFlags().IntVar(&args.maxBackups, "log-max-retained-files", 1000, "maximum number of the old log files to retain")
	//cmd.PersistentFlags().IntVar(&args.maxSize, "log-max-size", 100, "maximum size in megabytes of a log file before it gets rotated")
	//cmd.PersistentFlags().IntVar(&args.maxAge, "log-max-age", 30, "maximum number of days to retain old log files based on the timestamp encoded in their filename")

	// sub-commands
	cmd.AddCommand(newRunCmdWithOpts(corecmd.DefaultRunCmdOpts))
	cmd.AddCommand(version.NewVersionCmd())

	return cmd
}
