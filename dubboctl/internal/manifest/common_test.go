package manifest

import (
	"reflect"
	"testing"
)

func TestOverlaySetFlags(t *testing.T) {
	tests := []struct {
		base     string
		setFlags []string
		expect   string
	}{
		{
			base: `
spec:
  nacos:
    enabled: true
`,
			setFlags: []string{
				"spec.nacos.enabled=false",
				"spec.nacos.default=true",
				"spec.zookeeper.enabled=true",
			},
			expect: `
spec:
  nacos:
    enabled: false
    default: true
  zookeeper:
    enabled: true
`,
		},
	}
	for _, test := range tests {
		res, err := OverlaySetFlags(test.base, test.setFlags)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(res, test.expect) {
			t.Errorf("expect %s\n but got %s\n", test.expect, res)
		}
	}
}
