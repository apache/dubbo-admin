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
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type ProfileListArgs struct {
	ProfilesPath string
}

func (pla *ProfileListArgs) setDefault() {
	if pla == nil {
		return
	}
	if pla.ProfilesPath == "" {
		pla.ProfilesPath = identifier.Profiles
	}
}

func ConfigProfileListCmd(baseCmd *cobra.Command) {
	plArgs := &ProfileListArgs{}
	plCmd := &cobra.Command{
		Use:   "list",
		Short: "List all existing profiles specification",
		Example: `  # list all profiles provided by dubbo-admin
  dubboctl profile list

  # list all profiles in path specified by user
  dubboctl profile list --profiles path/to/profiles

  # display selected profile
  dubboctl profile list default
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("profile list doesn't support multi profile")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			plArgs.setDefault()
			if err := profileListCmd(plArgs, args); err != nil {
				return err
			}
			return nil
		},
	}
	plCmd.PersistentFlags().StringVarP(&plArgs.ProfilesPath, "profiles", "", "",
		"Path to profiles directory, this directory contains preset profiles")

	baseCmd.AddCommand(plCmd)
}

func profileListCmd(plArgs *ProfileListArgs, args []string) error {
	profiles, err := manifest.ReadProfilesNames(plArgs.ProfilesPath)
	if err != nil {
		return err
	}
	// list all profiles
	if len(args) == 0 {
		var resBuilder strings.Builder
		resBuilder.WriteString("Dubbo-admin profiles:\n")
		for _, profile := range profiles {
			resBuilder.WriteString("    " + profile + "\n")
		}
		logger.CmdSugar().Print(resBuilder.String())
		return nil
	}

	for _, profile := range profiles {
		if profile == args[0] {
			res, err := manifest.ReadProfileYaml(plArgs.ProfilesPath, profile)
			if err != nil {
				return err
			}
			logger.CmdSugar().Print(util.ApplyFilters(res, util.LicenseFilter, util.SpaceFilter))
			return nil
		}
	}

	return nil
}
