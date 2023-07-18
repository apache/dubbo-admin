// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package subcmd

import (
	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/kube"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/operator"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type ManifestUninstallArgs struct {
	ManifestGenerateArgs
	KubeConfigPath string
	// selected cluster info of kubeconfig
	Context string
}

func (mua *ManifestUninstallArgs) setDefault() {
	mua.ManifestGenerateArgs.setDefault()
}

func ConfigManifestUninstallCmd(baseCmd *cobra.Command) {
	muArgs := &ManifestUninstallArgs{}
	mgArgs := &muArgs.ManifestGenerateArgs
	muCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall dubbo control plane",
		Example: `  # Uninstall a default Dubbo control plane
  dubboctl manifest uninstall
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			muArgs.setDefault()
			cfg, _, err := generateValues(mgArgs)
			if err != nil {
				return err
			}
			if err := uninstallManifests(muArgs, cfg); err != nil {
				return err
			}
			return nil
		},
	}
	addManifestGenerateFlags(muCmd, mgArgs)
	muCmd.PersistentFlags().StringVarP(&muArgs.KubeConfigPath, "kubeConfig", "", "",
		"Path to kubeconfig")
	muCmd.PersistentFlags().StringVarP(&muArgs.Context, "context", "", "",
		"Context in kubeconfig to use")

	baseCmd.AddCommand(muCmd)
}

func uninstallManifests(muArgs *ManifestUninstallArgs, cfg *v1alpha1.DubboConfig) error {
	var cliOpts []kube.CtlClientOption
	if TestInstallFlag {
		cliOpts = []kube.CtlClientOption{kube.WithCli(TestCli)}
	} else {
		cliOpts = []kube.CtlClientOption{
			kube.WithKubeConfigPath(muArgs.KubeConfigPath),
			kube.WithContext(muArgs.Context),
		}
	}
	cli, err := kube.NewCtlClient(cliOpts...)
	if err != nil {
		return err
	}
	op, err := operator.NewDubboOperator(cfg.Spec, cli)
	if err != nil {
		return err
	}
	if err := op.Run(); err != nil {
		return err
	}
	manifestMap, err := op.RenderManifest()
	if err != nil {
		return err
	}
	if err := op.RemoveManifest(manifestMap); err != nil {
		return err
	}
	return nil
}
