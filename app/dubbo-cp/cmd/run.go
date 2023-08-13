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

	"github.com/apache/dubbo-admin/pkg/admin"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient"
	"github.com/apache/dubbo-admin/pkg/dds"
	"github.com/apache/dubbo-admin/pkg/snp"

	"github.com/apache/dubbo-admin/pkg/authority"
	"github.com/apache/dubbo-admin/pkg/config"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	"github.com/apache/dubbo-admin/pkg/core/cert"
	"github.com/apache/dubbo-admin/pkg/core/cmd"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/cp-server"
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

			if err := cert.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up certProvider")
			}

			if err := authority.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up authority")
			}

			if err := dds.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up dds")
			}

			if err := snp.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up snp")
			}

			if err := cp_server.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up grpc server")
			}

			// This must be last, otherwise we will not know which informers to register
			if err := kubeclient.Setup(rt); err != nil {
				logger.Sugar().Error(err, "unable to set up kube client")
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
