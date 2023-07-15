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
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/apache/dubbo-admin/app/dubboctl/cmd/subcmd"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManifestGenerate(t *testing.T) {
	tests := []struct {
		desc    string
		cmd     string
		temp    string
		wantErr bool
	}{
		{
			desc: "using default configuration without any flag",
			cmd:  "manifest generate",
		},
		{
			desc: "setting specification of built-in bootstrap",
			cmd:  "manifest generate --set spec.components.nacos.replicas=3",
		},
		{
			desc: "setting specification of add-on bootstrap",
			cmd:  "manifest generate --set spec.components.grafana.replicas=3",
		},
		{
			desc: "disabling bootstrap",
			cmd:  "manifest generate --set spec.componentsMeta.nacos.enabled=false",
		},
		{
			desc: "setting repository url and version of remote chart",
			cmd: "manifest generate --set spec.componentsMeta.grafana.repoURL=https://grafana.github.io/helm-charts" +
				" --set spec.componentsMeta.grafana.version=6.31.0",
		},
		{
			desc: "setting specification of built-in bootstrap with wrong path",
			cmd:  "manifest generate --set components.nacos.replicas=3",
		},
		{
			desc: "setting specification of built-in bootstrap with wrong path",
			cmd:  "manifest generate --set components.grafana.replicas=3",
		},
		{
			desc: "generate manifest to target path",
			cmd:  "manifest generate -o ./testdata/temp",
			temp: "./testdata/temp",
		},
		{
			desc: "input user specified yaml",
			cmd:  "manifest generate -f ./testdata/customization/user.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testExecute(t, test.cmd, test.wantErr)
			// remove temporary dir
			if test.temp != "" {
				os.RemoveAll(test.temp)
			}
		})
	}
}

func TestManifestInstall(t *testing.T) {
	tests := []struct {
		desc    string
		cmd     string
		wantErr bool
	}{
		{
			desc: "without any flag",
			cmd:  "manifest install",
		},
	}
	// For now, we do not use envTest to do black box testing
	subcmd.TestInstallFlag = true
	subcmd.TestCli = fake.NewClientBuilder().Build()

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testExecute(t, test.cmd, test.wantErr)
		})
	}
}

func TestManifestUninstall(t *testing.T) {
	tests := []struct {
		desc string
		// cmd has been executed before
		before  string
		cmd     string
		wantErr bool
	}{
		{
			desc:   "without any flag",
			before: "manifest install",
			cmd:    "manifest uninstall",
		},
	}
	// For now, we do not use envTest to do black box testing
	subcmd.TestInstallFlag = true
	subcmd.TestCli = fake.NewClientBuilder().Build()

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// prepare existing resources
			testExecute(t, test.before, false)
			testExecute(t, test.cmd, test.wantErr)
		})
	}
}

func TestManifestDiff(t *testing.T) {
	tests := []struct {
		desc    string
		befores []string
		cmd     string
		temps   []string
		wantErr bool
	}{
		{
			desc: "compare two dirs",
			befores: []string{
				"manifest generate -f ./testdata/diff/profileA.yaml -o ./testdata/tempA",
				"manifest generate -f ./testdata/diff/profileB.yaml -o ./testdata/tempB",
			},
			cmd: "manifest diff ./testdata/tempA ./testdata/tempB --compareDir=true",
			temps: []string{
				"./testdata/tempA",
				"./testdata/tempB",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			for _, before := range test.befores {
				testExecute(t, before, false)
			}
			testExecute(t, test.cmd, test.wantErr)
			for _, temp := range test.temps {
				if temp != "" {
					os.RemoveAll(temp)
				}
			}
		})
	}
}

func testExecute(t *testing.T, cmd string, wantErr bool) string {
	var out bytes.Buffer
	args := strings.Split(cmd, " ")
	rootCmd := getRootCmd(args)
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		if wantErr {
			return ""
		}
		t.Errorf("execute %s failed, err: %s", cmd, err)
		return ""
	}
	if wantErr {
		t.Errorf("want err but got no err")
		return ""
	}
	return out.String()
}
