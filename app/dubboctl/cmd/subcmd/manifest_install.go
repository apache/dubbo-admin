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

type ManifestInstallArgs struct {
	ManifestGenerateArgs
	KubeConfigPath string
	// selected cluster info of kubeconfig
	Context string
}

func (mia *ManifestInstallArgs) setDefault() {
	mia.ManifestGenerateArgs.setDefault()
}

func ConfigManifestInstallCmd(baseCmd *cobra.Command) {
	miArgs := &ManifestInstallArgs{}
	mgArgs := &miArgs.ManifestGenerateArgs
	miCmd := &cobra.Command{
		Use:   "install",
		Short: "install dubbo control plane",
		Example: `  # Install a default Dubbo control plane
  dubboctl manifest install
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			miArgs.setDefault()
			cfg, _, err := generateValues(mgArgs)
			if err != nil {
				return err
			}
			if err := installManifests(miArgs, cfg); err != nil {
				return err
			}
			return nil
		},
	}
	addManifestGenerateFlags(miCmd, mgArgs)
	miCmd.PersistentFlags().StringVarP(&miArgs.KubeConfigPath, "kubeConfig", "", "",
		"Path to kubeconfig")
	miCmd.PersistentFlags().StringVarP(&miArgs.Context, "context", "", "",
		"Context in kubeconfig to use")

	baseCmd.AddCommand(miCmd)
}

func installManifests(miArgs *ManifestInstallArgs, cfg *v1alpha1.DubboConfig) error {
	var cliOpts []kube.CtlClientOption
	if TestInstallFlag {
		cliOpts = []kube.CtlClientOption{kube.WithCli(TestCli)}
	} else {
		cliOpts = []kube.CtlClientOption{
			kube.WithKubeConfigPath(miArgs.KubeConfigPath),
			kube.WithContext(miArgs.Context),
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
	if err := op.ApplyManifest(manifestMap); err != nil {
		return err
	}
	return nil
}
