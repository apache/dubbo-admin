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

import "testing"

func TestProfileList(t *testing.T) {
	tests := []struct {
		desc    string
		cmd     string
		want    string
		wantErr bool
	}{
		{
			desc: "list all profiles provided by dubbo-admin",
			cmd:  "profile list",
			want: `Dubbo-admin profiles:
    default
    demo
`,
		},
		{
			desc: "list all profiles in path specified by user",
			cmd:  "profile list --profiles ./testdata/profile",
			want: `Dubbo-admin profiles:
    test0
    test1
    test2_wrong_format
`,
		},
		{
			desc: "display selected profile",
			cmd:  "profile list test0 --profiles ./testdata/profile",
			want: `apiVersion: dubbo.apache.org/v1alpha1
kind: DubboConfig
metadata:
  namespace: dubbo-system
spec:
  profile: default
  namespace: dubbo-system
  componentsMeta:
    admin:
      enabled: true
    nacos:
      enabled: true`,
		},
		{
			desc:    "display selected profile with wrong format",
			cmd:     "profile list test2_wrong_format --profiles ./testdata/profile",
			wantErr: true,
		},
		{
			desc:    "display selected profiles",
			cmd:     "profile list test0 test1 --profiles ./testdata/profile",
			wantErr: true,
		},
		{
			desc: "display profile that does not exist",
			cmd:  "profile list test2 --profiles ./testdata/profile",
			want: "",
		},
		{
			desc:    "list profile directory that does not exist",
			cmd:     "profile list --profiles ./testdata/profile/non_exist",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res := testExecute(t, test.cmd, test.wantErr)
			if test.want != "" && test.want != res {
				t.Errorf("want:\n%s\nbutgot:\n%s\n", test.want, res)
				return
			}
		})
	}
}

func TestProfileDiff(t *testing.T) {
	tests := []struct {
		desc    string
		cmd     string
		want    string
		wantErr bool
	}{
		{
			desc: "show the difference between two profiles provided by dubbo-admin",
			cmd:  "profile diff default default",
			want: `two profiles are identical
`,
		},
		{
			desc: "show the difference between two profiled specified by user",
			cmd:  "profile diff test0 test1 --profiles ./testdata/profile",
			want: ` apiVersion: dubbo.apache.org/v1alpha1
 kind: DubboConfig
 metadata:
   namespace: dubbo-system
 spec:
   componentsMeta:
     admin:
       enabled: true
-    nacos:
-      enabled: true
   namespace: dubbo-system
   profile: default
 `,
		},
		{
			desc:    "do not specify two profiles",
			cmd:     "profile diff test0 --profiles ./testdata/profile",
			wantErr: true,
		},
		{
			desc:    "diff profiles with wrong format",
			cmd:     "profile diff test0 test2_wrong_format --profiles ./testdata/profile",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res := testExecute(t, test.cmd, test.wantErr)
			if test.want != "" && test.want != res {
				t.Errorf("want:\n%s\nbutgot:\n%s\n", test.want, res)
				return
			}
		})
	}
}
