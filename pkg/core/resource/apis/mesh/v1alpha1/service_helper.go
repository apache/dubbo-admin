package v1alpha1

import "github.com/apache/dubbo-admin/pkg/core/consts"

func BuildServiceKey(serviceName, version, group string) string {
	return serviceName + consts.Colon + version + consts.Colon + group
}
