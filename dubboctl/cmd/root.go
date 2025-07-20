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
	"flag"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/pack"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/credentials"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/credentials/prompt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/deployer"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/pusher"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/validate"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/version"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	// "os"
)

const ChartFlag = "charts"

type staticClient struct {
	clientFactory ClientFactory
}

type ClientFactory func(...sdk.Option) (*sdk.Client, func())

func NewClientFactory(options ...sdk.Option) (*sdk.Client, func()) {
	var (
		t = newTransport(false)
		c = newCredentialsProvider(util.Dir(), t)
		d = newDeployer()
		o = []sdk.Option{
			sdk.WithRepositoriesPath(util.RepositoriesPath()),
			sdk.WithBuilder(pack.NewBuilder()),
			sdk.WithPusher(pusher.NewPusher(
				pusher.WithCredentialsProvider(c),
				pusher.WithTransport(t))),
			sdk.WithDeployer(d),
		}
	)
	client := sdk.New(append(o, options...)...)

	cleanup := func() {}
	return client, cleanup
}

func newTransport(insecureSkipVerify bool) pusher.RoundTripCloser {
	return pusher.NewRoundTripper(pusher.WithInsecureSkipVerify(insecureSkipVerify))
}

func newCredentialsProvider(configPath string, t http.RoundTripper) pusher.CredentialsProvider {
	options := []credentials.Opt{
		credentials.WithPromptForCredentials(prompt.NewPromptForCredentials(os.Stdin, os.Stdout, os.Stderr)),
		credentials.WithPromptForCredentialStore(prompt.NewPromptForCredentialStore()),
		credentials.WithTransport(t),
	}
	return credentials.NewCredentialsProvider(configPath, options...)
}

func newDeployer() sdk.Deployer {
	var options []deployer.DeployerOption

	return deployer.NewDeployer(options...)
}

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func GetRootCmd(args []string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "dubboctl",
		Short:         "Dubbo command line utilities",
		Long:          `Dubbo configuration command line utility for debug and use dubbo applications.`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	AddFlags(rootCmd)
	rootCmd.SetArgs(args)
	flags := rootCmd.PersistentFlags()
	rootOptions := cli.AddRootFlags(flags)
	ctx := cli.NewCLIContext(rootOptions)
	dcfg := staticClient{}
	factory := dcfg.clientFactory
	if factory == nil {
		factory = NewClientFactory
	}

	installCmd := cluster.InstallCmd(ctx)
	rootCmd.AddCommand(installCmd)
	hideFlags(installCmd, ChartFlag)

	uninstallCmd := cluster.UninstallCmd(ctx)
	rootCmd.AddCommand(uninstallCmd)

	upgradeCmd := cluster.UpgradeCmd(ctx)
	rootCmd.AddCommand(upgradeCmd)

	manifestCmd := cluster.ManifestCmd(ctx)
	rootCmd.AddCommand(manifestCmd)
	hideFlags(manifestCmd, ChartFlag)

	validateCmd := validate.NewValidateCommand(ctx)
	rootCmd.AddCommand(validateCmd)
	hideFlags(validateCmd, ChartFlag)

	versionCmd := version.NewVersionCommand(ctx)
	rootCmd.AddCommand(versionCmd)
	hideFlags(versionCmd, ChartFlag)

	createCmd := CreateCmd(ctx, rootCmd, factory)
	rootCmd.AddCommand(createCmd)
	hideFlags(createCmd, ChartFlag)

	repoCmd := RepoCmd(ctx, rootCmd, factory)
	rootCmd.AddCommand(repoCmd)
	hideFlags(repoCmd, ChartFlag)

	imageCmd := ImageCmd(ctx, rootCmd, factory)
	rootCmd.AddCommand(imageCmd)
	hideFlags(imageCmd, ChartFlag)

	return rootCmd

}

func hideFlags(origin *cobra.Command, hide ...string) {
	origin.SetHelpFunc(func(command *cobra.Command, args []string) {
		for _, hf := range hide {
			_ = command.Flags().MarkHidden(hf)
		}
		origin.SetHelpFunc(nil)
		origin.HelpFunc()(command, args)
	})
}
