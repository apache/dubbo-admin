package manifest

import (
	"fmt"
	"github.com/dubbogo/dubbogo-cli/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/dubbogo/dubbogo-cli/internal/util"
	"os"
	"path"
	"sigs.k8s.io/yaml"
	"strings"
)

func ReadProfileYaml(profilePath string, profile string) (string, error) {
	filePath := path.Join(profilePath, profile)
	filePath += ".yaml"
	out, err := ReadAndOverlayYamls([]string{filePath})
	if err != nil {
		return "", err
	}
	return out, nil
}

func ReadYamlAndProfile(filenames []string, setFlags []string) (string, string, error) {
	mergedYaml, err := ReadAndOverlayYamls(filenames)
	if err != nil {
		return "", "", err
	}
	// unmarshal and validate
	tempOp := &v1alpha1.DubboOperator{}
	if err := yaml.Unmarshal([]byte(mergedYaml), tempOp); err != nil {
		return "", "", fmt.Errorf("ReadYamlAndProfile failed, err: %s", err)
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
		// inspect that this file only contains one CR
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
