package cmd

import (
	"github.com/dubbogo/dubbogo-cli/internal/cmd"
	"github.com/spf13/cobra"
)

var manifestCmd = &cobra.Command{
	Use:     "manifest",
	Short:   "",
	Long:    "",
	Example: "",
}

func init() {
	cmd.ConfigManifestGenerateCmd(manifestCmd)
	rootCmd.AddCommand(manifestCmd)
}
