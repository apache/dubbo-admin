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
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	dubboversion "github.com/apache/dubbo-admin/pkg/version"
)

const gracefullyShutdownDuration = 3 * time.Second

func newRunCmdWithOpts() *cobra.Command {
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
				return err
			}
			// 2. configure logger
			logger.Init(cfg.Log)
			cfgForDisplay, err := config.ConfigForDisplay(&cfg)
			if err != nil {
				logger.Errorf("unable to prepare config for display, cause: %s", err.Error())
				return err
			}
			cfgBytes, err := config.ToJson(cfgForDisplay)
			if err != nil {
				logger.Errorf("unable to convert config to json, cause: %s", err.Error())
				return err
			}
			logger.Infof("current config:\n %s", cfgBytes)

			// 2. build components
			gracefulCtx, ctx := setupSignalHandler()
			rt, err := bootstrap.Bootstrap(gracefulCtx, cfg)
			if err != nil {
				logger.Errorf("unable to bootstrap, cause: %s", err.Error())
				return err
			}

			// 3. start components
			logger.Infof("starting Admin......, version: %s", dubboversion.Build.Version)
			if err := rt.Start(gracefulCtx.Done()); err != nil {
				logger.Errorf("problem running Admin, cause: %s", err.Error())
				return err
			}
			logger.Info("stop signal received. Waiting 3 seconds for components to stop gracefully...")
			select {
			case <-ctx.Done():
				logger.Info("all components have stopped")
			case <-time.After(gracefullyShutdownDuration):
				logger.Info("forcefully stopped")
			}
			return nil
		},
	}
	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "./dubbo-admin.yaml", "configuration file")
	return cmd
}

func setupSignalHandler() (context.Context, context.Context) {
	gracefulCtx, gracefulCancel := context.WithCancel(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-c
		logger.Warnf("received signal %s, stopping instance gracefully", s.String())
		gracefulCancel()
		s = <-c
		logger.Warnf("received second signal %s, stopping instance", s.String())
		cancel()
		s = <-c
		logger.Warnf("received third signal %s, force exit", s.String())
		os.Exit(1)
	}()
	return gracefulCtx, ctx
}
