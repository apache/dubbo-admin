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

package cluster

import (
	"github.com/apache/dubbo-admin/dubboctl/pkg/cli"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"

	// "github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*installArgs
}

// UpgradeCmd performs an in-place upgrade with eligibility checks.
func UpgradeCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	upArgs := &upgradeArgs{
		installArgs: &installArgs{},
	}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the Dubbo",
		Long:  "The upgrade command can be used instead of the install command",
		Example: `  # Apply a default dubboctl installation.
  dubboctl upgrade
  
  # Apply a default profile.
  dubboctl upgrade --profile=demo

		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr())
			p := NewPrinterForWriter(cmd.OutOrStderr())
			client, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			return Install(client, rootArgs, upArgs.installArgs, cl, cmd.OutOrStdout(), p)
		},
	}
	AddFlags(cmd, rootArgs)
	addInstallFlags(cmd, upArgs.installArgs)
	return cmd
}
