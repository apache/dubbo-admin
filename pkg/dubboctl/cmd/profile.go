package cmd

import (
	"github.com/apache/dubbo-admin/pkg/dubboctl/cmd/subcmd"
	"github.com/spf13/cobra"
)

func addProfile(rootCmd *cobra.Command) {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Commands related to profiles",
		Long:  "Commands help user to list and describe profiles",
	}
	subcmd.ConfigProfileListArgs(profileCmd)
	subcmd.ConfigProfileDiffArgs(profileCmd)

	rootCmd.AddCommand(profileCmd)
}
