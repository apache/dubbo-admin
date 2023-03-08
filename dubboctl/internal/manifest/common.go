package manifest

import (
	"github.com/dubbogo/dubbogo-cli/internal/util"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

func ReadProfileYaml(profilePath string, profile string) (string, error) {
	path := profilePath + profile
	// valuate path
	out, err := ReadAndOverlayYamls([]string{path})
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
	// get profile field and overlay with setFlags
	return mergedYaml, "", nil
}

func ReadAndOverlayYamls(filenames []string) (string, error) {
	var output string
	for _, filename := range filenames {
		file, err := os.ReadFile(strings.TrimSpace(filename))
		if err != nil {
			return "", err
		}
		// inspect that this file only contains one CR
		output, err = OverlayYaml(output, string(file))
		if err != nil {
			return "", err
		}
	}
	return output, nil
}

func OverlayYaml(base string, overlay string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return overlay, nil
	}
	if strings.TrimSpace(overlay) == "" {
		return base, nil
	}
	baseJson, err := yaml.YAMLToJSON([]byte(base))
	if err != nil {
		return "", err
	}
	overlayJson, err := yaml.YAMLToJSON([]byte(overlay))
	if err != nil {
		return "", err
	}
	// todo: create a CRD to represent API
	mergedJson, err := strategicpatch.StrategicMergePatch(baseJson, overlayJson, nil)
	if err != nil {
		return "", err
	}
	mergeYaml, err := yaml.JSONToYAML(mergedJson)
	if err != nil {
		return "", err
	}
	return string(mergeYaml), nil
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
