/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin"
	"github.com/apache/dubbo-admin/pkg/config"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	"github.com/apache/dubbo-admin/pkg/core/cmd"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/spf13/cobra"
	"time"
)

func newConsoleCmdWithOpts(opts cmd.RunCmdOpts) *cobra.Command {
	args := struct {
		configPath string
	}{}

	cmd := &cobra.Command{
		Use:   "console",
		Short: "Launch Dubbo Admin console server.",
		Long:  `Launch Dubbo Admin console server.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := dubbo_cp.DefaultConfig()
			err := config.Load(args.configPath, &cfg)
			if err != nil {
				logger.Sugar().Error(err, "could not load the configuration")
				return err
			}
			gracefulCtx, ctx := opts.SetupSignalHandler()

			rt, err := bootstrap.Bootstrap(gracefulCtx, &cfg)
			if err != nil {
				logger.Sugar().Error(err, "unable to set up Control Plane runtime")
				return err
			}
			cfgForDisplay, err := config.ConfigForDisplay(&cfg)
			if err != nil {
				logger.Sugar().Error(err, "unable to prepare config for display")
				return err
			}
			cfgBytes, err := config.ToJson(cfgForDisplay)
			if err != nil {
				logger.Sugar().Error(err, "unable to convert config to json")
				return err
			}
			logger.Sugar().Info(fmt.Sprintf("Current config %s", cfgBytes))

			if err := admin.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up Metrics")
			}
			logger.Sugar().Info("starting Control Plane")
			if err := rt.Start(gracefulCtx.Done()); err != nil {
				logger.Sugar().Error(err, "problem running Control Plane")
				return err
			}

			logger.Sugar().Info("Stop signal received. Waiting 3 seconds for components to stop gracefully...")
			select {
			case <-ctx.Done():
			case <-time.After(gracefullyShutdownDuration):
			}
			logger.Sugar().Info("Stopping Control Plane")
			return nil
		},
	}

	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")

	return cmd
}
