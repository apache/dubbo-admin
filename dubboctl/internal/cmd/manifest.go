package cmd

import (
	"fmt"
	"github.com/dubbogo/dubbogo-cli/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/dubbogo/dubbogo-cli/internal/controlplane"
	"github.com/dubbogo/dubbogo-cli/internal/manifest"
	"github.com/dubbogo/dubbogo-cli/internal/util"
	"sigs.k8s.io/yaml"
)

type ManifestArgs struct {
	FileNames   []string
	ChartPath   string
	ProfilePath string
	OutputPath  string
	SetFlags    []string
}

func GenerateValues(mArgs *ManifestArgs) (*v1alpha1.DubboOperator, string, error) {
	mergedYaml, profile, err := manifest.ReadYamlAndProfile(mArgs.FileNames, mArgs.SetFlags)
	if err != nil {
		return nil, "", fmt.Errorf("GenerateValues err: %v", err)
	}
	profileYaml, err := manifest.ReadProfileYaml(mArgs.ProfilePath, profile)
	if err != nil {
		return nil, "", err
	}
	finalYaml, err := util.OverlayYAML(profileYaml, mergedYaml)
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
	// todo: validate op
	op.Spec.ProfilePath = mArgs.ProfilePath
	op.Spec.ChartPath = mArgs.ChartPath
	return op, finalYaml, nil
}

func GenerateManifests(mArgs *ManifestArgs, op *v1alpha1.DubboOperator) error {
	cp, err := controlplane.NewDubboControlPlane(op.Spec)
	if err != nil {
		return err
	}
	if err := cp.Run(); err != nil {
		return err
	}
	manifestMap, err := cp.RenderManifest()
	if err != nil {
		return err
	}
	fmt.Print(manifestMap)
	return nil
}
