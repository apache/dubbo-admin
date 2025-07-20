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
	"io"
	"os"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/dubboctl/pkg/cli"
	"github.com/apache/dubbo-admin/operator/pkg/install"
	"github.com/apache/dubbo-admin/operator/pkg/render"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/operator/pkg/util/progress"
	"github.com/apache/dubbo-admin/pkg/art"
	"github.com/apache/dubbo-admin/pkg/common/util/pointer"
	"github.com/apache/dubbo-admin/pkg/kube"
	"github.com/spf13/cobra"
)

type installArgs struct {
	// filenames is an array of paths to input DubboOperator CR files.
	// TODO
	// filenames []string
	// sets is a string with the format "path=value".
	sets []string
	// waitTimeout is the maximum time to wait for all Dubbo resources to be ready.
	// This setting takes effect only when the "wait" parameter is set to true.
	waitTimeout time.Duration
	// skipConfirmation determines whether the user is prompted for confirmation.
	// If set to true, the user is not prompted, and a "Yes" response is assumed in all cases.
	skipConfirmation bool
}

func (i *installArgs) String() string {
	var b strings.Builder
	// b.WriteString("filenames:    " + (fmt.Sprint(i.filenames) + "\n"))
	b.WriteString("sets:    " + (fmt.Sprint(i.sets) + "\n"))
	b.WriteString("waitTimeout: " + fmt.Sprint(i.waitTimeout) + "\n")
	return b.String()
}

func addInstallFlags(cmd *cobra.Command, args *installArgs) {
	// cmd.PersistentFlags().StringSliceVarP(&args.filenames, "filenames", "f", nil, `Path to the file containing the dubboOperator's custom resources.`)
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, `Override dubboOperator values, such as selecting profiles, etc.`)
	cmd.PersistentFlags().BoolVarP(&args.skipConfirmation, "skip-confirmation", "y", false, `The skipConfirmation determines whether the user is prompted for confirmation.`)
	cmd.PersistentFlags().DurationVar(&args.waitTimeout, "wait-timeout", 300*time.Second, "Maximum time to wait for Dubbo resources in each component to be ready.")
}

// InstallCmdWithArgs generates an Dubbo install manifest and applies it to a cluster.
func InstallCmdWithArgs(ctx cli.Context, rootArgs *RootArgs, iArgs *installArgs) *cobra.Command {
	ic := &cobra.Command{
		Use:   "install",
		Short: "Applies an Dubbo manifest, installing or reconfiguring Dubbo on a cluster",
		Long:  "The install command generates an Dubbo install manifest and applies it to a cluster",
		Example: ` # Apply a default dubboctl installation.
  dubboctl install -y
 
  # Apply a demo profile.
  dubboctl install --set profile=demo -y
		`,
		Aliases: []string{"apply"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeClient, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			p := NewPrinterForWriter(cmd.OutOrStderr())
			cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr())
			p.Printf("%v\n", art.DubboColoredArt())
			return Install(kubeClient, rootArgs, iArgs, cl, cmd.OutOrStdout(), p)
		},
	}
	AddFlags(ic, rootArgs)
	addInstallFlags(ic, iArgs)
	return ic
}

// InstallCmd generates an Dubbo install manifest and applies it to a cluster.
func InstallCmd(ctx cli.Context) *cobra.Command {
	return InstallCmdWithArgs(ctx, &RootArgs{}, &installArgs{})
}

func Install(kubeClient kube.CLIClient, rootArgs *RootArgs, iArgs *installArgs, cl clog.Logger, stdOut io.Writer, p Printer) error {
	setFlags := applyFlagAliases(iArgs.sets)
	manifests, vals, err := render.GenerateManifest(nil, setFlags, cl, kubeClient)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}
	profile := pointer.NonEmptyOrDefault(vals.GetPathString("spec.profile"), "default")
	if !rootArgs.DryRun && !iArgs.skipConfirmation {
		prompt := fmt.Sprintf("The %q profile will be installed into the cluster. \nDo you want to proceed? (y/N)", profile)
		if !OptionDeterminate(prompt, stdOut) {
			p.Println("Canceled Completed.")
			os.Exit(1)
		}
	}
	i := install.Installer{
		DryRun:       rootArgs.DryRun,
		SkipWait:     false,
		Kube:         kubeClient,
		Values:       vals,
		WaitTimeout:  iArgs.waitTimeout,
		ProgressInfo: progress.NewLog(),
		Logger:       cl,
	}
	if err := i.InstallManifests(manifests); err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}
	return nil
}

// --bar is an alias for --set bar=
// --foo is an alias for --set foo=
func applyFlagAliases(flags []string) []string {
	return flags
}
