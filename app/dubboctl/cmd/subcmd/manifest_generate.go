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
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"go.uber.org/zap/zapcore"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest/render"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/operator"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"
	"github.com/spf13/cobra"

	"sigs.k8s.io/yaml"
)

type ManifestGenerateArgs struct {
	FileNames    []string
	ChartsPath   string
	ProfilesPath string
	OutputPath   string
	SetFlags     []string
}

func (mga *ManifestGenerateArgs) setDefault() {
	if mga == nil {
		return
	}
	if mga.ProfilesPath == "" {
		mga.ProfilesPath = identifier.Profiles
	}
	if mga.ChartsPath == "" {
		mga.ChartsPath = identifier.Charts
	}
}

func ConfigManifestGenerateCmd(baseCmd *cobra.Command) {
	mgArgs := &ManifestGenerateArgs{}
	mgCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate dubbo control plane manifest to apply",
		Example: `  # Generate a default Dubbo control plane manifest
  dubboctl manifest generate

  # Setting specification of built-in bootstrap
  dubboctl manifest generate --set spec.components.nacos.replicas=3

  # Setting specification of add-on bootstrap
  dubboctl manifest generate --set spec.components.grafana.replicas=3

  # Disabling bootstrap
  dubboctl manifest generate --set spec.componentsMeta.nacos.enabled=false

  # Setting repository url and version of remote chart
  dubboctl manifest generate --set spec.componentsMeta.grafana.repoURL=https://grafana.github.io/helm-charts --set spec.componentsMeta.grafana.version=6.31.0

  # Generate manifest to target path
  dubboctl manifest generate -o /path/to/temp

  # Input user specified yaml
  dubboctl manifest generate -f /path/to/user.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			mgArgs.setDefault()
			cfg, _, err := generateValues(mgArgs)
			if err != nil {
				return err
			}
			if err := generateManifests(mgArgs, cfg); err != nil {
				return err
			}
			return nil
		},
	}
	addManifestGenerateFlags(mgCmd, mgArgs)

	baseCmd.AddCommand(mgCmd)
}

func addManifestGenerateFlags(cmd *cobra.Command, args *ManifestGenerateArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.FileNames, "filenames", "f", nil,
		"User-defined DubboConfig yaml files, the previous file would be overlaid by later file")
	cmd.PersistentFlags().StringVarP(&args.ChartsPath, "charts", "", "",
		"Path to charts directory, this directory contains components charts")
	cmd.PersistentFlags().StringVarP(&args.ProfilesPath, "profiles", "", "",
		"Path to profiles directory, this directory contains preset profiles")
	cmd.PersistentFlags().StringVarP(&args.OutputPath, "output", "o", "",
		"Path to output manifest, if not set, dubboctl would print the manifest")
	cmd.PersistentFlags().StringArrayVarP(&args.SetFlags, "set", "s", nil,
		"Set DubboConfig fields, see /pkg/dubboctl/internal/apis/dubbo.apache.org/v1alpha1/types.go")
}

// In order to generate values.yaml for helm charts, dubboctl takes following order to overlay:
//
//  1. mergedYaml, profile <- user1.yaml <- user2.yaml <- user3.yaml ...
//
//     User set FileNames, dubboctl reads these user-defined DubboConfig yamls and overlays them from back to front.
//     mergedYaml is the overlaid result.
//     User could set required profile name by user-defined DubboConfig yamls or SetFlag. If none are set, dubboctl
//     would use default profile. The priority is default profile < user-defined DubboConfig yaml < SetFlag.
//
//  2. profileYaml <- profile
//
//     dubboctl reads profile yaml. Profile can be considered the recommended configuration in some classic scenarios.
//     For now, there is only default.yaml.
//
//  3. finalYaml <- profileYaml <- mergedYaml <- SetFlags
//
//     Based on profileYaml, user-defined yaml and setFlags are overlaid in order.
//
//  4. Marshal finalYaml to DubboConfig, And use DubboConfig to represent values.yaml and other information.
func generateValues(mgArgs *ManifestGenerateArgs) (*v1alpha1.DubboConfig, string, error) {
	mergedYaml, profile, err := manifest.ReadYamlAndProfile(mgArgs.FileNames, mgArgs.SetFlags)
	if err != nil {
		return nil, "", fmt.Errorf("process user specification failed, err: %s", err)
	}
	profileYaml, err := manifest.ReadOverlayProfileYaml(mgArgs.ProfilesPath, profile)
	if err != nil {
		return nil, "", fmt.Errorf("process profile failed, err: %s", err)
	}
	finalYaml, err := util.OverlayYAML(profileYaml, mergedYaml)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err = manifest.OverlaySetFlags(finalYaml, mgArgs.SetFlags)
	if err != nil {
		return nil, "", fmt.Errorf("process set flags failed, err: %s", err)
	}
	cfg := &v1alpha1.DubboConfig{}
	if err := yaml.Unmarshal([]byte(finalYaml), cfg); err != nil {
		return nil, "", fmt.Errorf("set flags specification is wrong, err: %s", err)
	}
	// we should ensure that Components field would not be nil
	if cfg.Spec.Components == nil {
		cfg.Spec.Components = &v1alpha1.DubboComponentsSpec{}
	}
	cfg.Spec.ProfilePath = mgArgs.ProfilesPath
	cfg.Spec.ChartPath = mgArgs.ChartsPath
	return cfg, finalYaml, nil
}

func generateManifests(mgArgs *ManifestGenerateArgs, cfg *v1alpha1.DubboConfig) error {
	op, err := operator.NewDubboOperator(cfg.Spec, nil)
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
	if mgArgs.OutputPath == "" {
		// in order to have the same manifest output every time with the same input
		res, err := sortManifests(manifestMap)
		if err != nil {
			return err
		}
		logger.CmdSugar().Print(res)
	} else {
		if err := writeManifests(manifestMap, mgArgs.OutputPath); err != nil {
			return err
		}
	}
	return nil
}

func sortManifests(manifestMap map[operator.ComponentName]string) (string, error) {
	var names []string
	var resBuilder strings.Builder
	for name := range manifestMap {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		file := manifestMap[operator.ComponentName(name)]
		if !strings.HasSuffix(file, render.YAMLSeparator) {
			resBuilder.WriteString(file + render.YAMLSeparator)
		} else {
			resBuilder.WriteString(file)
		}
	}
	return resBuilder.String(), nil
}

func writeManifests(manifestMap map[operator.ComponentName]string, outputPath string) error {
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return err
	}
	for name, val := range manifestMap {
		filename := path.Join(outputPath, string(name)+".yaml")
		if err := os.WriteFile(filename, []byte(val), 0o644); err != nil {
			return err
		}
	}
	return nil
}
