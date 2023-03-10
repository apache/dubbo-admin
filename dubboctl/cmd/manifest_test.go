package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestManifestGenerate(t *testing.T) {
	tests := []struct {
		cmd string
	}{
		{
			cmd: "manifest generate --charts ../../deploy/charts/admin-stack/charts --profiles ../../deploy/profiles",
		},
	}
	for _, test := range tests {
		var out bytes.Buffer
		args := strings.Split(test.cmd, " ")
		rootCmd.SetArgs(args)
		rootCmd.SetOut(&out)
		if err := rootCmd.Execute(); err != nil {
			t.Error(err)
		}
		t.Log(out.String())
	}
}
