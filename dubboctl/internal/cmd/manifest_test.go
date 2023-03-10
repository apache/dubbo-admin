package cmd

import (
	"reflect"
	"testing"
)

func TestGenerateValues(t *testing.T) {
	tests := []struct {
		args       *ManifestArgs
		expectVals string
	}{
		{
			args: &ManifestArgs{
				FileNames:   nil,
				ChartPath:   "../../../deploy/charts/admin-stack/charts",
				ProfilePath: "../../../deploy/profiles",
				OutputPath:  "",
				SetFlags:    nil,
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
		op, vals, err := GenerateValues(test.args)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(vals, test.expectVals) {
			t.Errorf("expect: %s\n res: %s", test.expectVals, vals)
		}
		if err := GenerateManifests(test.args, op); err != nil {
			t.Error(err)
		}
	}
}
