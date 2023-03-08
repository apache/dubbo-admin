package cmd

import (
	"fmt"
	"github.com/dubbogo/dubbogo-cli/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/dubbogo/dubbogo-cli/internal/manifest"
	"sigs.k8s.io/yaml"
)

type ManifestArgs struct {
	FileNames    []string
	ManifestPath string
	SetFlags     []string
}

func GenerateValues(mArgs *ManifestArgs) (*v1alpha1.DubboOperator, string, error) {
	mergedYaml, profile, err := manifest.ReadYamlAndProfile(mArgs.FileNames, mArgs.SetFlags)
	if err != nil {
		return nil, "", fmt.Errorf("GenerateValues err: %v", err)
	}
	profileYaml, err := manifest.ReadProfileYaml(mArgs.ManifestPath, profile)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err := manifest.OverlayYaml(profileYaml, mergedYaml)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err = manifest.OverlaySetFlags(finalYaml, mArgs.SetFlags)
	if err != nil {
		return nil, "", err
	}
	op := &v1alpha1.DubboOperator{}
	if err := yaml.Unmarshal([]byte(finalYaml), op); err != nil {
		return nil, "", err
	}
	// validate op
	return op, finalYaml, nil
}

func GenerateManifests(mArgs *ManifestArgs) {

}
