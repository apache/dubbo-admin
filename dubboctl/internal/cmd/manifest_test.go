package cmd

import (
	"reflect"
	"testing"
)

func TestGenerateValues(t *testing.T) {
	tests := []struct {
		args       *ManifestGenerateArgs
		expectVals string
	}{
		{
			args: &ManifestGenerateArgs{
				FileNames:    nil,
				ChartsPath:   "../../../deploy/charts/admin-stack/charts",
				ProfilesPath: "../../../deploy/profiles",
				OutputPath:   "",
				SetFlags:     nil,
			},
			expectVals: `
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboOperator
metadata:
  namespace: dubbo-system
spec:
  componentsMeta:
    zookeeper:
      enabled: true
      namespace: dubbo-system
  profile: default
`,
		},
	}

	for _, test := range tests {
		op, vals, err := generateValues(test.args)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(vals, test.expectVals) {
			t.Errorf("expect: %s\n res: %s", test.expectVals, vals)
		}
		if err := generateManifests(test.args, op); err != nil {
			t.Error(err)
		}
	}
}
