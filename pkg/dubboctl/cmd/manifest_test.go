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
)

const (
	TestPath = "./testdata"
)

func TestManifestGenerate(t *testing.T) {
	tests := []struct {
		desc string
		cmd  string
		temp string
	}{
		{
			desc: "without any flag",
			cmd:  "manifest generate",
		},
		{
			desc: "setting specification of built-in component",
			cmd:  "manifest generate --set spec.components.nacos.replicas=3",
		},
		{
			desc: "setting specification of add-on component",
			cmd:  "manifest generate --set spec.components.grafana.replicas=3",
		},
		{
			desc: "disabling component",
			cmd:  "manifest generate --set spec.componentsMeta.nacos.enabled=false",
		},
		{
			desc: "setting repository url and version of remote chart",
			cmd: "manifest generate --set spec.componentsMeta.grafana.repoURL=https://grafana.github.io/helm-charts" +
				" --set spec.componentsMeta.grafana.version=6.31.0",
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
		var out bytes.Buffer
		args := strings.Split(test.cmd, " ")
		addSubCommands(rootCmd)
		rootCmd.SetArgs(args)
		rootCmd.SetOut(&out)
		if err := rootCmd.Execute(); err != nil {
			t.Error(err)
			return
		}
		// remove temporary dir
		if test.temp != "" {
			os.RemoveAll(test.temp)
		}
		// todo:// use output to test
		// t.Log(out.String())
	}
}

// todo: // need to make use of envtest
//func TestManifestInstall(t *testing.T) {
//	tests := []struct {
//		desc string
//		cmd  string
//	}{
//		{
//			desc: "without any flag",
//			cmd:  "manifest install",
//		},
//	}
//	testEnv := envtest.Environment{}
//	cfg, err := testEnv.Start()
//	if err != nil {
//		t.Fatalf("k8s test env start failed: %s", err)
//	}
//	t.Log(cfg.String())
//
//	//for _, test := range tests {
//	//	var out bytes.Buffer
//	//	args := strings.Split(test.cmd, " ")
//	//	rootCmd.SetArgs(args)
//	//	rootCmd.SetOut(&out)
//	//	if err := rootCmd.Execute(); err != nil {
//	//		t.Error(err)
//	//	}
//	//	t.Log(out.String())
//	//}
//}
