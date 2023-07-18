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
	"errors"
	"fmt"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/kube"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type ProfileDiffArgs struct {
	ProfilesPath string
}

func (pda *ProfileDiffArgs) setDefault() {
	if pda == nil {
		return
	}
	if pda.ProfilesPath == "" {
		pda.ProfilesPath = identifier.Profiles
	}
}

func ConfigProfileDiffCmd(baseCmd *cobra.Command) {
	pdArgs := &ProfileDiffArgs{}
	pdCmd := &cobra.Command{
		Use:   "diff",
		Short: "Show the difference between two profiles",
		Example: `  # show the difference between two profiles provided by dubbo-admin
  dubboctl profile diff default demo

  # show the difference between two profiles specified by user
  dubboctl profile diff profile_name0 profile_name1 --profiles /path/to/profiles
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("profile diff needs two profiles")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			pdArgs.setDefault()
			if err := profileDiffCmd(pdArgs, args); err != nil {
				return err
			}
			return nil
		},
	}
	pdCmd.PersistentFlags().StringVarP(&pdArgs.ProfilesPath, "profiles", "", "",
		"Path to profiles directory, this directory contains preset profiles")

	baseCmd.AddCommand(pdCmd)
}

func profileDiffCmd(pdArgs *ProfileDiffArgs, args []string) error {
	profileA, err := manifest.ReadProfileYaml(pdArgs.ProfilesPath, args[0])
	if err != nil {
		return fmt.Errorf("parse %s profile failed, err: %s", args[0], err)
	}
	profileB, err := manifest.ReadProfileYaml(pdArgs.ProfilesPath, args[1])
	if err != nil {
		return fmt.Errorf("parse %s profile failed, err: %s", args[1], err)
	}
	objA, err := kube.ParseObjectFromManifest(profileA)
	if err != nil {
		return fmt.Errorf("parse %s profile to k8s object failed, err: %s", args[0], err)
	}
	objB, err := kube.ParseObjectFromManifest(profileB)
	if err != nil {
		return fmt.Errorf("parse %s profile to k8s object failed, err: %s", args[1], err)
	}
	diff, err := kube.CompareObject(objA, objB)
	if err != nil {
		return fmt.Errorf("compare failed, err: %s", err)
	}
	if diff != "" {
		logger.CmdSugar().Print(diff)
	} else {
		logger.CmdSugar().Print("two profiles are identical\n")
	}
	return nil
}
