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
	"os"

	"github.com/spf13/cobra"

	"github.com/apache/dubbo-admin/pkg/core/cmd/version"
)

// newRootCmd represents the base command when called without any subcommands.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dubbo-admin",
		Short: "Universal Admin for Dubbo Application",
		Long:  `Universal Admin for Dubbo Application`,
	}

	cmd.SetOut(os.Stdout)
	// sub-commands
	cmd.AddCommand(newRunCmdWithOpts())
	cmd.AddCommand(version.NewVersionCmd())

	return cmd
}

func DefaultRootCmd() *cobra.Command {
	return newRootCmd()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.maixn(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := DefaultRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
