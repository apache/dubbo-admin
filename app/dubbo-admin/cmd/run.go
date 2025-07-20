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
	"time"

	"github.com/spf13/cobra"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	dubbocmd "github.com/apache/dubbo-admin/pkg/core/cmd"
	dubboversion "github.com/apache/dubbo-admin/pkg/version"
)

var runLog = adminLog.WithName("run")

const gracefullyShutdownDuration = 3 * time.Second

// This is the open file limit below which the control plane may not
// reasonably have enough descriptors to accept all its clients.
const minOpenFileLimit = 4096

func newRunCmdWithOpts(opts dubbocmd.RunCmdOpts) *cobra.Command {
	args := struct {
		configPath string
	}{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch Admin",
		Long:  `Launch Admin`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 1. config load and validate
			cfg := app.DefaultAdminConfig()
			err := config.Load(args.configPath, &cfg)
			if err != nil {
				runLog.Error(err, "could not load the configuration")
				return err

			}
			cfgForDisplay, err := config.ConfigForDisplay(&cfg)
			if err != nil {
				runLog.Error(err, "unable to prepare config for display")
				return err
			}
			cfgBytes, err := config.ToJson(cfgForDisplay)
			if err != nil {
				runLog.Error(err, "unable to convert config to json")
				return err
			}
			runLog.Info(fmt.Sprintf("Current config %s", cfgBytes))
			runLog.Info(fmt.Sprintf("Running in mode `%s`", cfg.Mode))

			// 2. build components
			gracefulCtx, ctx := opts.SetupSignalHandler()
			rt, err := bootstrap.Bootstrap(gracefulCtx, cfg)
			if err != nil {
				runLog.Error(err, "unable to bootstrap")
				return err
			}

			// 3. start components
			runLog.Info("starting Admin......", "version", dubboversion.Build.Version)
			if err := rt.Start(gracefulCtx.Done()); err != nil {
				runLog.Error(err, "problem running Admin")
				return err
			}

			runLog.Info("stop signal received. Waiting 3 seconds for components to stop gracefully...")
			select {
			case <-ctx.Done():
				runLog.Info("all components have stopped")
			case <-time.After(gracefullyShutdownDuration):
				runLog.Info("forcefully stopped")
			}
			return nil
		},
	}
	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")
	return cmd
}
