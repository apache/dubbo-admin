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

package backup

import (
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/herkyl/patchwerk"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

func TestPatchwerk(t *testing.T) {
	tests := []struct {
		desc  string
		file1 string
		file2 string
		file3 string
		file4 string
	}{
		{
			file1: "resources/original.yaml",
			file2: `resources/patch.yaml`,
			file3: `resources/patch2.yaml`,
			file4: `resources/patch.yaml`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			originalYaml, _ := os.ReadFile(strings.TrimSpace(test.file1))
			originalJson, _ := yaml.YAMLToJSON([]byte(originalYaml))

			patchyaml, _ := os.ReadFile(strings.TrimSpace(test.file2))
			patchJson, _ := yaml.YAMLToJSON([]byte(patchyaml))

			patch, _ := patchwerk.DiffBytes(originalJson, patchJson)
			fmt.Printf(string(patch))

		})
	}
}

func TestDiffYAML(t *testing.T) {
	tests := []struct {
		desc  string
		file1 string
		file2 string
		file3 string
		file4 string
	}{
		{
			file1: "resources/original.yaml",
			file2: `resources/patch.yaml`,
			file3: `resources/patch2.yaml`,
			file4: `resources/patch.yaml`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			originalYaml, _ := os.ReadFile(strings.TrimSpace(test.file1))
			originalJson, _ := yaml.YAMLToJSON([]byte(originalYaml))

			patchyaml, _ := os.ReadFile(strings.TrimSpace(test.file2))
			patchJson, _ := yaml.YAMLToJSON([]byte(patchyaml))

			patch, _ := jsonpatch.CreateMergePatch(originalJson, patchJson)
			fmt.Printf(string(patch))
			//timeoutPath := []byte(`{"configs":[{"parameters":{"timeout":"600"},"side":"provider"}]}`)
			//// Let's combine these merge patch documents...
			//combinedPatch, err := jsonpatch.MergeMergePatches(patch, timeoutPath)
			//if err != nil {
			//	panic(err)
			//}
			//mergePatch1, _ := jsonpatch.MergePatch(originalJson, combinedPatch)
			//fmt.Printf(string(mergePatch1))

			patchyaml2, _ := os.ReadFile(strings.TrimSpace(test.file3))
			patchJson2, _ := yaml.YAMLToJSON([]byte(patchyaml2))
			patch2, _ := jsonpatch.CreateMergePatch(originalJson, patchJson2)
			fmt.Printf(string(patch2))
			//mergePatch2, _ := jsonpatch.MergePatch(originalJson, patch2)
			//fmt.Printf(string(mergePatch2))

		})
	}
}

func TestApplyPatch(t *testing.T) {
	tests := []struct {
		desc  string
		file1 string
		file2 string
		file3 string
	}{
		{
			file1: "resources/original.yaml",
			file2: `resources/patch.yaml`,
			file3: `result.yaml`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			originalYaml, _ := os.ReadFile(strings.TrimSpace(test.file1))
			originalJson, err := yaml.YAMLToJSON([]byte(originalYaml))

			//patchyaml, _ := os.ReadFile(strings.TrimSpace(test.file2))

			patchJSON := []byte(`[
				{"op": "add", "path": "/configs/0/parameters/accesslog", "value": "false"}
			]`)

			patch, err := jsonpatch.DecodePatch(patchJSON)
			apply, err := patch.Apply(originalJson)
			if err != nil {
				fmt.Printf("%v", err)
			}
			fmt.Print(string(apply))

			// todo:// inspect that this file only contains one CR
			//resultYaml, err := OverlayYAML(string(originalYaml), string(patchJSON))
			//fmt.Printf("%v", err)
			//fmt.Print(resultYaml)
		})
	}
}

func TestMergePatch(t *testing.T) {
	t.Run("mergePath", func(t *testing.T) {
		//originalYaml, _ := os.ReadFile(strings.TrimSpace("resources/original.yaml"))
		//originalJson, err := yaml.YAMLToJSON([]byte(originalYaml))
		//
		//timeout := []byte(`{"timeout":null,"name":"Jane"}`)
		//accesslog := []byte(`{"height: 3.45":4.23,"name":null}`)
		//
		//// Let's combine these merge patch documents...
		//combinedPatch, err := jsonpatch.MergeMergePatches(nameAndHeight, ageAndEyes)
		//if err != nil {
		//	panic(err)
		//}
		//
		//withCombinedPatch, err := jsonpatch.MergePatch(original, combinedPatch)
		//if err != nil {
		//	panic(err)
		//}

		//fmt.Printf("combined merge patch: %s", withCombinedPatch)
	})
}
