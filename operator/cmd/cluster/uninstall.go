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
	"fmt"
	"os"

	"github.com/apache/dubbo-admin/dubboctl/pkg/cli"
	"github.com/apache/dubbo-admin/operator/pkg/render"
	"github.com/apache/dubbo-admin/operator/pkg/uninstall"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/operator/pkg/util/progress"
	"github.com/apache/dubbo-admin/pkg/kube"
	"github.com/spf13/cobra"
)

type uninstallArgs struct {
	// filenames is an array of paths to input DubboOperator CR files.
	// TODO
	// filenames string
	// sets is a string with the format "path=value".
	sets []string
	// manifestPath is a path to a charts and profiles directory in the local filesystem with a release tgz.
	manifestPath string
	// purge results in deletion of all Dubbo resources.
	purge bool
	// skipConfirmation determines whether the user is prompted for confirmation.
	// If set to true, the user is not prompted, and a "Yes" response is assumed in all cases.
	skipConfirmation bool
}

func addUninstallFlags(cmd *cobra.Command, args *uninstallArgs) {
	// cmd.PersistentFlags().StringVarP(&args.filenames, "filenames", "f", "", "The filename of the DubboOperator CR.")
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, `Override dubboOperator values, such as selecting profiles, etc.`)
	cmd.PersistentFlags().BoolVar(&args.purge, "purge", false, `Remove all dubbo related source code.`)
	cmd.PersistentFlags().BoolVarP(&args.skipConfirmation, "skip-confirmation", "y", false, `The skipConfirmation determines whether the user is prompted for confirmation.`)
}

// UninstallCmd command uninstalls Dubbo from a cluster
func UninstallCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	uiArgs := &uninstallArgs{}
	uicmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Dubbo related resources",
		Long:  "The uninstall command will uninstall the dubbo cluster",
		Example: ` # Uninstall all control planes and shared resources
  dubboctl uninstall --purge`,
		Args: func(cmd *cobra.Command, args []string) error {
			if !uiArgs.purge {
				return fmt.Errorf("at least one of the --purge flags must be set")
			}
			if len(args) > 0 {
				return fmt.Errorf("dubboctl uninstall does not take arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return Uninstall(cmd, ctx, rootArgs, uiArgs)
		},
	}
	AddFlags(uicmd, rootArgs)
	addUninstallFlags(uicmd, uiArgs)
	return uicmd
}

// Uninstall uninstalls by deleting specified manifests.
func Uninstall(cmd *cobra.Command, ctx cli.Context, rootArgs *RootArgs, uiArgs *uninstallArgs) error {
	cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr())
	var kubeClient kube.CLIClient
	var err error
	kubeClient, err = ctx.CLIClientWithRevision("")
	if err != nil {
		return err
	}

	pl := progress.NewLog()
	if uiArgs.purge {
		cl.LogAndPrint("Purge uninstall will purge all Dubbo resources, ignoring the specified revision or operator file")
	}

	setFlags := applyFlagAliases(uiArgs.sets)

	files := []string{}

	vals, err := render.MergeInputs(files, setFlags)
	if err != nil {
		return err
	}

	objectsList, err := uninstall.GetRemovedResources(
		kubeClient,
		vals.GetPathString("metadata.name"),
		vals.GetPathString("metadata.namespace"),
		uiArgs.purge,
	)
	if err != nil {
		return err
	}

	preCheck(cmd, uiArgs, cl, rootArgs.DryRun)

	if err := uninstall.DeleteObjectsList(kubeClient, rootArgs.DryRun, cl, objectsList); err != nil {
		return err
	}

	pl.SetState(progress.StateUninstallComplete)

	return nil
}

// preCheck checks for potential major changes.
func preCheck(cmd *cobra.Command, uiArgs *uninstallArgs, _ *clog.ConsoleLogger, dryRun bool) {
	needConfirmation, message := false, ""
	if uiArgs.purge {
		needConfirmation = true
		message += "All Dubbo resources will be pruned from the cluster.\n"
	}
	if dryRun || uiArgs.skipConfirmation {
		return
	}
	message += "Do you want to proceed? (y/N)"
	if needConfirmation && !OptionDeterminate(message, cmd.OutOrStdout()) {
		cmd.Print("Canceled Completed.\n")
		os.Exit(1)
	}
}
