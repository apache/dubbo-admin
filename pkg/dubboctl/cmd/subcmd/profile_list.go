package subcmd

import (
	"errors"
	"github.com/apache/dubbo-admin/pkg/dubboctl/identifier"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/manifest"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/util"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"strings"
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

func ConfigProfileListArgs(baseCmd *cobra.Command) {
	plArgs := &ProfileListArgs{}
	plCmd := &cobra.Command{
		Use:   "list",
		Short: "List all existing profiles specification",
		Example: `  # list all profiles provided by dubbo-admin
  dubboctl profile list

  # list all profiles in path specified by user
  dubboctl profile list --profiles /path/to/profiles

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
