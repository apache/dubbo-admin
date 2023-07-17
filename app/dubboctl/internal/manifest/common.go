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

package manifest

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"

	"sigs.k8s.io/yaml"
)

func ReadOverlayProfileYaml(profilePath string, profile string) (string, error) {
	filePath := path.Join(profilePath, profile+".yaml")
	defaultPath := path.Join(profilePath, "default.yaml")
	// overlay selected profile ont default profile
	out, err := ReadAndOverlayYamls([]string{defaultPath, filePath})
	if err != nil {
		return "", err
	}
	// unmarshal and validate
	tempOp := &v1alpha1.DubboConfig{}
	if err := yaml.Unmarshal([]byte(out), tempOp); err != nil {
		return "", fmt.Errorf("%s profile on default profile is wrong, err: %s", profile, err)
	}
	return out, nil
}

func ReadYamlAndProfile(filenames []string, setFlags []string) (string, string, error) {
	mergedYaml, err := ReadAndOverlayYamls(filenames)
	if err != nil {
		return "", "", err
	}
	// unmarshal and validate
	tempOp := &v1alpha1.DubboConfig{}
	if err := yaml.Unmarshal([]byte(mergedYaml), tempOp); err != nil {
		return "", "", fmt.Errorf("user specification is wrong, err: %s", err)
	}
	// get profile field and overlay with setFlags
	profile := "default"
	if opProfile := tempOp.GetProfile(); opProfile != "" {
		profile = opProfile
	}
	if profileVal := GetValueFromSetFlags(setFlags, "profile"); profileVal != "" {
		profile = profileVal
	}

	return mergedYaml, profile, nil
}

func ReadAndOverlayYamls(filenames []string) (string, error) {
	var output string
	for _, filename := range filenames {
		file, err := os.ReadFile(strings.TrimSpace(filename))
		if err != nil {
			return "", err
		}
		// todo:// inspect that this file only contains one CR
		output, err = util.OverlayYAML(output, string(file))
		if err != nil {
			return "", err
		}
	}
	return output, nil
}

func OverlaySetFlags(base string, setFlags []string) (string, error) {
	baseMap := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", err
	}
	for _, setFlag := range setFlags {
		key, val := SplitSetFlag(setFlag)
		pathCtx, _, err := GetPathContext(baseMap, util.PathFromString(key), true)
		if err != nil {
			return "", err
		}
		if err := WritePathContext(pathCtx, ParseValue(val), false); err != nil {
			return "", err
		}
	}
	finalYaml, err := yaml.Marshal(baseMap)
	if err != nil {
		return "", err
	}
	return string(finalYaml), nil
}

// ReadProfilesNames reads all profiles in directory specified by profilesPath.
// It does not traverse recursively.
// It may add some filters in the future.
func ReadProfilesNames(profilesPath string) ([]string, error) {
	var res []string
	dir, err := os.ReadDir(profilesPath)
	if err != nil {
		return nil, err
	}
	yamlSuffix := ".yaml"
	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), yamlSuffix) {
			res = append(res, strings.TrimSuffix(file.Name(), ".yaml"))
		}
	}
	return res, nil
}

// ReadProfileYaml reads profile yaml specified by profilePath/profile.yaml and validates the content.
func ReadProfileYaml(profilePath string, profile string) (string, error) {
	filePath := path.Join(profilePath, profile+".yaml")
	out, err := ReadAndOverlayYamls([]string{filePath})
	if err != nil {
		return "", err
	}
	// unmarshal and validate
	tempOp := &v1alpha1.DubboConfig{}
	if err := yaml.Unmarshal([]byte(out), tempOp); err != nil {
		return "", fmt.Errorf("%s profile is wrong, err: %s", profile, err)
	}
	return out, nil
}
