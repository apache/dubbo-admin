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
	"log"
	"time"

	"github.com/spf13/cobra"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	dubbocmd "github.com/apache/dubbo-admin/pkg/core/cmd"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
	"github.com/apache/dubbo-admin/pkg/mcp/transport/stdio"
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

			// 2. build components
			gracefulCtx, ctx := opts.SetupSignalHandler()
			rt, err := bootstrap.Bootstrap(gracefulCtx, cfg)
			if err != nil {
				runLog.Error(err, "unable to bootstrap")
				return err
			}

			// 3. start components
			runLog.Info("starting Admin......", "version", dubboversion.Build.Version)

			// Start MCP server if enabled
			var mcpTransport *stdio.Transport
			if cfg.MCP != nil && cfg.MCP.Enabled {
				runLog.Info("MCP server enabled, starting...", "serverName", cfg.MCP.ServerName)

				// Create console context for MCP tools
				consoleCtx := consolectx.NewConsoleContext(rt)

				// Create MCP server
				server := core.NewServer(cfg.MCP.ServerName, cfg.MCP.ServerVersion)

				// Register all tools
				reg := server.GetRegistry()
				reg.RegisterRegistrar(&tools.MetricsRegistrar{})
				reg.RegisterRegistrar(&tools.ResourceSearchRegistrar{})
				reg.RegisterRegistrar(&tools.ServiceRegistrar{})
				reg.RegisterRegistrar(&tools.DetailRegistrar{})
				reg.RegisterAll()

				// Set console context
				server.SetConsoleContext(consoleCtx)

				runLog.Info("MCP server initialized", "tools", len(reg.List()))

				// Create and start stdio transport in background
				mcpTransport = stdio.NewTransport(server)
				mcpErrCh := make(chan error, 1)
				go func() {
					if err := mcpTransport.Serve(gracefulCtx); err != nil {
						runLog.Error(err, "MCP transport error")
						mcpErrCh <- err
					}
				}()

				// Log to stderr for Claude Desktop to see
				fmt.Fprintf(log.Writer(), "%s MCP Server v%s started\n", cfg.MCP.ServerName, cfg.MCP.ServerVersion)
				fmt.Fprintf(log.Writer(), "Registered %d tools\n", len(reg.List()))
			}

			// Start HTTP server and other components
			if err := rt.Start(gracefulCtx.Done()); err != nil {
				runLog.Error(err, "problem running Admin")
				return err
			}

			runLog.Info("stop signal received. Waiting 3 seconds for components to stop gracefully...")
			select {
			case <-ctx.Done():
				runLog.Info("all components have stopped")
				// Close MCP transport if it was started
				if mcpTransport != nil {
					mcpTransport.Close()
				}
			case <-time.After(gracefullyShutdownDuration):
				runLog.Info("forcefully stopped")
				// Close MCP transport if it was started
				if mcpTransport != nil {
					mcpTransport.Close()
				}
			}
			return nil
		},
	}
	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")
	return cmd
}
