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
