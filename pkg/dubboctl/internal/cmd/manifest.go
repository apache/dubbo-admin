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

package cmd

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/controlplane"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/manifest"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/manifest/render"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/util"
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

func ConfigManifestGenerateCmd(baseCmd *cobra.Command) {
	mgArgs := &ManifestGenerateArgs{}
	mgCmd := &cobra.Command{
		Use:     "generate",
		Short:   "",
		Long:    "",
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			op, _, err := generateValues(mgArgs)
			if err != nil {
				return err
			}
			if err := generateManifests(mgArgs, op); err != nil {
				return err
			}
			return nil
		},
	}
	// add manifest generate flag
	mgCmd.PersistentFlags().StringSliceVarP(&mgArgs.FileNames, "filename", "f", nil, "")
	mgCmd.PersistentFlags().StringVarP(&mgArgs.ChartsPath, "charts", "", "", "")
	mgCmd.PersistentFlags().StringVarP(&mgArgs.ProfilesPath, "profiles", "", "", "")
	mgCmd.PersistentFlags().StringVarP(&mgArgs.OutputPath, "output", "o", "", "")
	mgCmd.PersistentFlags().StringArrayVarP(&mgArgs.SetFlags, "set", "s", nil, "")

	baseCmd.AddCommand(mgCmd)
}

func generateValues(mgArgs *ManifestGenerateArgs) (*v1alpha1.DubboOperator, string, error) {
	mergedYaml, profile, err := manifest.ReadYamlAndProfile(mgArgs.FileNames, mgArgs.SetFlags)
	if err != nil {
		return nil, "", fmt.Errorf("generateValues err: %v", err)
	}
	profileYaml, err := manifest.ReadProfileYaml(mgArgs.ProfilesPath, profile)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err := util.OverlayYAML(profileYaml, mergedYaml)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err = manifest.OverlaySetFlags(finalYaml, mgArgs.SetFlags)
	if err != nil {
		return nil, "", err
	}
	op := &v1alpha1.DubboOperator{}
	if err := yaml.Unmarshal([]byte(finalYaml), op); err != nil {
		return nil, "", err
	}
	// todo: validate op
	op.Spec.ProfilePath = mgArgs.ProfilesPath
	op.Spec.ChartPath = mgArgs.ChartsPath
	return op, finalYaml, nil
}

func generateManifests(mgArgs *ManifestGenerateArgs, op *v1alpha1.DubboOperator) error {
	cp, err := controlplane.NewDubboControlPlane(op.Spec)
	if err != nil {
		return err
	}
	if err := cp.Run(); err != nil {
		return err
	}
	manifestMap, err := cp.RenderManifest()
	if err != nil {
		return err
	}
	if mgArgs.OutputPath == "" {
		res, err := sortManifests(manifestMap)
		if err != nil {
			return err
		}
		// todo: using specific logger module
		fmt.Println(res)
	} else {
		if err := writeManifests(manifestMap, mgArgs.OutputPath); err != nil {
			return err
		}
	}
	return nil
}

func sortManifests(manifestMap map[controlplane.ComponentName]string) ([]string, error) {
	var names []string
	var res []string
	for name := range manifestMap {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		file := manifestMap[controlplane.ComponentName(name)]
		if !strings.HasSuffix(file, render.YAMLSeparator) {
			res = append(res, file+render.YAMLSeparator)
		} else {
			res = append(res, file)
		}
	}
	return res, nil
}

func writeManifests(manifestMap map[controlplane.ComponentName]string, outputPath string) error {
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
