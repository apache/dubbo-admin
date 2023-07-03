package cmd

import (
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/providers/mock"
	"github.com/apache/dubbo-admin/pkg/admin/router"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/authority"
	"github.com/apache/dubbo-admin/pkg/core/cmd"
	"github.com/apache/dubbo-admin/pkg/logger"

	caconfig "github.com/apache/dubbo-admin/pkg/authority/config"

	"os"
	"time"

	"github.com/spf13/cobra"
)

const gracefullyShutdownDuration = 3 * time.Second

// This is the open file limit below which the control plane may not
// reasonably have enough descriptors to accept all its clients.
const minOpenFileLimit = 4096

func newRunCmdWithOpts(opts cmd.RunCmdOpts) *cobra.Command {
	args := struct {
		configPath string
	}{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch Dubbo Admin",
		Long:  `Launch Dubbo Admin.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// init config
			config.LoadConfig()

			// subscribe to registries
			go services.StartSubscribe(config.RegistryCenter)
			defer func() {
				services.DestroySubscribe(config.RegistryCenter)
			}()

			// start mock server
			os.Setenv(constant.ConfigFileEnvKey, config.MockProviderConf)
			go mock.RunMockServiceServer()

			// start console server
			router := router.InitRouter()
			if err := router.Run(":38080"); err != nil {
				logger.Error("Failed to start Admin console server.")
				return err
			}

			// start CA
			if err := startCA(cmd); err != nil {
				logger.Error("Failed to start CA server.")
				return err
			}

			// start

			return nil
		},
	}

	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")

	return cmd
}

func startCA(cmd *cobra.Command) error {
	options := caconfig.NewOptions()

	if err := authority.Initialize(cmd); err != nil {
		logger.Fatal("Failed to initialize CA server.")
		return err
	}

	logger.Infof("Authority command Run with options: %+v", options)
	if errs := options.Validate(); len(errs) != 0 {
		logger.Fatal(errs)
		return errs[0]
	}

	if err := authority.Run(options); err != nil {
		logger.Fatal(err)
		return err
	}
	return nil
}
